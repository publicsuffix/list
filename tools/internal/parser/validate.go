package parser

import (
	"context"
	"errors"
	"log"
	"net"
	"strconv"
	"strings"

	"github.com/creachadair/mds/mapset"
	"github.com/creachadair/taskgroup"
	"github.com/publicsuffix/list/tools/internal/domain"
	"github.com/publicsuffix/list/tools/internal/githistory"
	"github.com/publicsuffix/list/tools/internal/github"
)

// ValidateOffline runs offline validations on a parsed PSL.
func ValidateOffline(l *List) []error {
	var ret []error

	for _, block := range BlocksOfType[*Section](l) {
		if block.Name == "PRIVATE DOMAINS" {
			ret = append(ret, validateEntityMetadata(block)...)
			break
		}
	}
	validateExpectedSections(l)
	validateSuffixUniqueness(l)

	return ret
}

// validateEntityMetadata verifies that all suffix blocks have some
// kind of entity name.
func validateEntityMetadata(block Block) []error {
	var ret []error
	for _, block := range BlocksOfType[*Suffixes](block) {
		if !block.Changed() {
			continue
		}

		if block.Info.Name == "" {
			ret = append(ret, ErrMissingEntityName{
				Suffixes: block,
			})
		}
		if len(block.Info.Maintainers) == 0 && !exemptFromContactInfo(block.Info.Name) {
			ret = append(ret, ErrMissingEntityEmail{
				Suffixes: block,
			})
		}
	}
	return ret
}

// validateExpectedSections verifies that the two top-level sections
// (ICANN and private domains) exist, are not duplicated, and that no
// other sections are present.
func validateExpectedSections(block Block) (errs []error) {
	// Use an ordered set for the wanted sections, so that we can
	// check section names in O(1) but also report missing sections in
	// a deterministic order.
	wanted := mapset.New("ICANN DOMAINS", "PRIVATE DOMAINS")
	found := map[string]*Section{}
	for _, section := range BlocksOfType[*Section](block) {
		if !wanted.Has(section.Name) && section.Changed() {
			errs = append(errs, ErrUnknownSection{section})
		} else if other, ok := found[section.Name]; ok && (section.Changed() || other.Changed()) {
			errs = append(errs, ErrDuplicateSection{section, other})
		} else {
			found[section.Name] = section
		}
	}

	for _, name := range wanted.Slice() {
		if _, ok := found[name]; !ok {
			errs = append(errs, ErrMissingSection{name})
		}
	}

	return errs
}

// validateSuffixUniqueness verifies that suffixes only appear once
// each.
func validateSuffixUniqueness(block Block) (errs []error) {
	suffixes := map[string]*Suffix{}    // domain.Name.String() -> Suffix
	wildcards := map[string]*Wildcard{} // base domain.Name.String() -> Wildcard

	for _, suffix := range BlocksOfType[*Suffix](block) {
		name := suffix.Domain.String()
		if other, ok := suffixes[name]; ok && (suffix.Changed() || other.Changed()) {
			errs = append(errs, ErrDuplicateSuffix{name, suffix, other})
		} else {
			suffixes[name] = suffix
		}
	}

	for _, wildcard := range BlocksOfType[*Wildcard](block) {
		name := wildcard.Domain.String()
		if other, ok := wildcards[name]; ok && (wildcard.Changed() || other.Changed()) {
			errs = append(errs, ErrDuplicateSuffix{"*." + name, wildcard, other})
		} else {
			wildcards[name] = wildcard
		}

		for _, exc := range wildcard.Exceptions {
			fqdn, err := wildcard.Domain.AddPrefix(exc)
			if err != nil && wildcard.Changed() {
				errs = append(errs, err)
				continue
			}
			name := fqdn.String()
			if suffix, ok := suffixes[name]; ok && (wildcard.Changed() || suffix.Changed()) {
				errs = append(errs, ErrConflictingSuffixAndException{suffix, wildcard})
			}
		}
	}

	return errs
}

// ValidateOnline runs online validations on a parsed PSL. Online
// validations are slower than offline validation, especially when
// checking the entire PSL. All online validations respect
// cancellation on the given context.
func ValidateOnline(ctx context.Context, l *List, client *github.Repo, prHistory *githistory.History) (errs []error) {
	for _, section := range BlocksOfType[*Section](l) {
		if section.Name == "PRIVATE DOMAINS" {
			errs = append(errs, validateTXTRecords(ctx, section, client, prHistory)...)
			break
		}
	}
	return errs
}

const (
	// concurrentDNSRequests is the maximum number of in-flight TXT
	// lookups. This is primarily limited by the ability of the local
	// stub resolver and LAN resolver to absorb the traffic without
	// dropping packets. This is set very conservatively to avoid
	// false positives.
	concurrentDNSRequests = 10
	// concurrentGithubRequests is the maximum number of in-flight
	// Github API requests. This is limited to avoid tripping Github
	// DoS protections. Github seems to have made their rate limits
	// more aggressive recently, so we stick to something very slow.
	concurrentGithubRequests = 10

	// txtRecordPrefix is the prefix of valid _psl TXT records.
	txtRecordPrefix = "https://github.com/publicsuffix/list/pull/"
)

// prExpected is the set of changed suffixes we expect to see in a
// particular PR, given TXT record information.
type prExpected struct {
	// suffixes maps a domain string (Suffix.Domain.String()) to the
	// Suffix entry in the PSL that's being validated.
	suffixes map[string]*Suffix
	// wildcards maps a wildcard domain (Wildcard.Domain.String()) to
	// the Wildcard entry in the PSL that's being validated.
	wildcards map[string]*Wildcard
}

// txtRecordChecker is the in-flight state for validateTXTRecords.
type txtRecordChecker struct {
	// ctx is the parent context for the validation. In-flight
	// validation should abort if this context gets canceled
	// (e.g. user hit ctrl+c, or timeout was reached)
	ctx context.Context

	// gh is the Github API client used to look up PRs and commits.
	gh *github.Repo
	// resolver is the DNS resolver used to do TXT lookups.
	resolver net.Resolver

	// errs accumulates the TXT validation errors discovered during
	// the checking process.
	errs []error

	// prExpected records what changes we expect to see in PRs, based
	// on TXT lookups. For example, if a TXT lookup for _psl.foo.com
	// points to PR 123, this map will have an entry for 123 ->
	// {suffixes: [foo.com]}, meaning if we look at PR 123 on github,
	// we want to see a change for foo.com.
	prExpected map[int]*prExpected

	hist *githistory.History
}

// validateTXTRecords checks the TXT records of all Suffix and
// Wildcard blocks found under b.
func validateTXTRecords(ctx context.Context, b Block, client *github.Repo, prHistory *githistory.History) (errs []error) {
	checker := txtRecordChecker{
		ctx:        ctx,
		prExpected: map[int]*prExpected{},
		gh:         client,
		hist:       prHistory,
	}

	// TXT checking happens in two phases: first, look up all TXT
	// records and construct a map of suffix<>github PR map (and
	// record errors for missing/malformed records, of course).
	//
	// Then, check the Github PRs and verify that they are changing
	// the expected domains. We do this so that someone cannot point
	// their TXT record to some random PR and pass our validation
	// check, the PR must be changing the correct suffix(es).
	//
	// Both TXT lookups and PR verification are parallelized. This
	// complicates the code below slightly due to the use of
	// taskgroup, but it results in a 10-100x speedup compared to
	// serialized verification.

	// TXT record checking. The checkTXT function (below) is the
	// parallel part of the process, processLookupResult collects the
	// results/errors.
	collect := taskgroup.NewCollector(checker.processLookupResult)
	group, start := taskgroup.New(nil).Limit(concurrentDNSRequests)

	for _, suf := range BlocksOfType[*Suffix](b) {
		if !suf.Changed() || exemptFromTXT(suf.Domain) {
			continue
		}
		start(collect.NoError(func() txtResult { return checker.checkTXT(suf, suf.Domain) }))
	}
	for _, wild := range BlocksOfType[*Wildcard](b) {
		if !wild.Changed() || exemptFromTXT(wild.Domain) {
			continue
		}
		start(collect.NoError(func() txtResult { return checker.checkTXT(wild, wild.Domain) }))
	}

	group.Wait()

	// PR verification. Now that TXT lookups are complete,
	// checker.prExpected has a list of PRs, and the list of changes
	// we want to see in each PR. The checkPR function is the parallel
	// part and does all the work, the collector just concats all the
	// validation errors together.
	collectPR := taskgroup.NewCollector(func(errs []error) {
		checker.errs = append(checker.errs, errs...)
	})
	group, start = taskgroup.New(nil).Limit(concurrentGithubRequests)

	for prNum, info := range checker.prExpected {
		start(collectPR.NoError(func() []error { return checker.checkPR(prNum, info) }))
	}
	group.Wait()

	return checker.errs
}

// txtResult is the result of one DNS TXT lookup.
type txtResult struct {
	// Block is the Suffix or Wildcard that this lookup is for. We
	// track it here because we need it to provide context in
	// validation errors that could happen in future.
	Block
	// prs is the list of PR numbers that were found in TXT records.
	prs []int
	// err is the DNS lookup error we got, or nil if the query
	// succeeded.
	err error
}

// checkTXT checks the _psl TXT record for one domain.
func (c *txtRecordChecker) checkTXT(block Block, domain domain.Name) txtResult {
	log.Printf("Checking TXT for _psl.%s", domain.String())
	// The trailing dot is to prevent search path expansion.
	res, err := c.resolver.LookupTXT(c.ctx, "_psl."+domain.ASCIIString()+".")
	if err != nil {
		return txtResult{
			Block: block,
			err:   err,
		}
	}
	return txtResult{
		Block: block,
		prs:   extractPSLRecords(res),
	}
}

// extractPSLRecords extracts github PR numbers from raw TXT
// records. TXT records which do not match the required PSL record
// format (e.g. SPF records, DKIM records, other unrelated
// verification records) are ignored.
func extractPSLRecords(txts []string) []int {
	// You might think that since we put PSL records under
	// _psl.<domain>, we would only see well formed PSL
	// records. However, a large number of domains return SPF and
	// other "well known" TXT records types as well, probably because
	// their DNS server is hardcoded to return those for all TXT
	// query. So, we have to gracefully ignore "malformed" TXT records
	// here.
	var ret []int
	for _, txt := range txts {
		if !strings.HasPrefix(txt, txtRecordPrefix) {
			continue
		}
		for _, f := range strings.Fields(strings.TrimSpace(txt)) {
			prStr, ok := strings.CutPrefix(f, txtRecordPrefix)
			if !ok {
				continue
			}
			prNum, err := strconv.Atoi(prStr)
			if err != nil {
				continue
			}

			if prNum == 0 {
				// At least one _psl record points to a bogus PR
				// number 0. Presumably this was a placeholder that
				// was never updated. Treat it identically to other
				// missing/malformed records.
				continue
			}

			// Apply special cases where the PR listed in DNS is not
			// quite right, but due to procedural issues on the PSL
			// side rather than suffix owner error. See exceptions.go
			// for more explanation.
			prNum = adjustTXTPR(prNum)

			ret = append(ret, prNum)
		}
	}
	return ret
}

// processLookupResult updates the record checker's state with the
// information in the given txtResult.
func (c *txtRecordChecker) processLookupResult(res txtResult) {
	var dnsErr *net.DNSError
	if errors.As(res.err, &dnsErr) {
		if dnsErr.IsNotFound {
			// DNS server returned NXDOMAIN. Use a specific error for
			// that.
			c.errs = append(c.errs, ErrMissingTXTRecord{res.Block})
		} else {
			// Other DNS errors, e.g. SERVFAIL or REFUSED.
			c.errs = append(c.errs, ErrTXTCheckFailure{res.Block, res.err})
		}
		return
	} else if res.err != nil {
		// Other non-DNS errors, e.g. timeout or "no route to host".
		c.errs = append(c.errs, ErrTXTCheckFailure{res.Block, res.err})
		return
	}
	if len(res.prs) == 0 {
		// _psl.<domain> has TXT records, but none of them look like
		// valid PSL records. Behave the same as NXDOMAIN.
		c.errs = append(c.errs, ErrMissingTXTRecord{res.Block})
		return
	}
	for _, prNum := range res.prs {
		// Found some PRs for this suffix, record them in preparation
		// for PR validation.
		inf := c.prInfo(prNum)
		switch v := res.Block.(type) {
		case *Suffix:
			inf.suffixes[v.Domain.String()] = v
		case *Wildcard:
			inf.wildcards[v.Domain.String()] = v
		default:
			panic("unexpected AST node")
		}
	}
}

func (c *txtRecordChecker) getPRPSLs(prNum int) (before, after []byte, err error) {
	if c.hist != nil {
		if inf, ok := c.hist.PRs[prNum]; ok {
			before, err = githistory.GetPSL(c.hist.GitPath, inf.ParentHash)
			if err != nil {
				return nil, nil, err
			}
			after, err = githistory.GetPSL(".", inf.CommitHash)
			if err != nil {
				return nil, nil, err
			}
			return before, after, nil
		}
	}

	return c.gh.PSLForPullRequest(c.ctx, prNum)
}

// checkPRs looks up the given Github PR, and verifies that it changes
// all the suffixes and wildcards provided in info.
func (c *txtRecordChecker) checkPR(prNum int, info *prExpected) []error {
	log.Printf("Checking PR %d", prNum)

	// Some PRs have broken state in Github, the API returns nonsense
	// information. These cases were verified manually and recorded in
	// exceptions.go, so we skip those checks to avoid spurious
	// errors.
	for _, suf := range info.suffixes {
		if acceptPRForDomain(suf.Domain, prNum) {
			delete(info.suffixes, suf.Domain.String())
		}
	}
	if len(info.suffixes) == 0 && len(info.wildcards) == 0 {
		return nil
	}

	var ret []error

	beforeBs, afterBs, err := c.getPRPSLs(prNum)
	if err != nil {
		for _, suf := range info.suffixes {
			ret = append(ret, ErrTXTCheckFailure{suf, err})
		}
		for _, wild := range info.wildcards {
			ret = append(ret, ErrTXTCheckFailure{wild, err})
		}
		return ret
	}

	before, _ := Parse(beforeBs)
	after, _ := Parse(afterBs)
	after.SetBaseVersion(before, true)

	// Look at the changed suffixes/wildcards in the PR, and remove
	// those from info. At the end of this, info is either empty (no
	// issues, we found all the suffixes we wanted in the correct PR),
	// or has leftover suffixes (validation error: a TXT record
	// references an unrelated PR).
	for _, suf := range BlocksOfType[*Suffix](after) {
		if !suf.Changed() {
			continue
		}

		// delete is a no-op if the key doesn't exist, so we can do
		// the deletion unconditionally.
		delete(info.suffixes, suf.Domain.String())
	}

	for _, wild := range BlocksOfType[*Wildcard](after) {
		if !wild.Changed() {
			continue
		}

		delete(info.wildcards, wild.Domain.String())
	}

	// Anything left over now is a validation error.
	for _, suf := range info.suffixes {
		ret = append(ret, ErrTXTRecordMismatch{suf, prNum})
	}
	for _, wild := range info.wildcards {
		ret = append(ret, ErrTXTRecordMismatch{wild, prNum})
	}
	return ret
}

// prInfo is a helper to get-or-create c.prExpected[pr].
func (c *txtRecordChecker) prInfo(pr int) *prExpected {
	if ret, ok := c.prExpected[pr]; ok {
		return ret
	}
	ret := &prExpected{
		suffixes:  map[string]*Suffix{},
		wildcards: map[string]*Wildcard{},
	}
	c.prExpected[pr] = ret
	return ret
}
