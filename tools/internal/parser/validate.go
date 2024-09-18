package parser

import (
	"context"
	"errors"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"

	"github.com/creachadair/mds/mapset"
	"github.com/creachadair/taskgroup"
	"github.com/publicsuffix/list/tools/internal/domain"
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
func ValidateOnline(ctx context.Context, l *List) (errs []error) {
	for _, section := range BlocksOfType[*Section](l) {
		if section.Name == "PRIVATE DOMAINS" {
			errs = append(errs, validateTXTRecords(ctx, section)...)
			break
		}
	}
	return errs
}

const (
	// concurrentDNSRequests is the maximum number of in-flight TXT
	// lookups. This is primarily limited by the ability of the local
	// stub resolver and LAN resolver to absorb the traffic without
	// dropping packets. Empirically 100 is easily manageable (most
	// ad-ridden websites do more than this).
	concurrentDNSRequests    = 100
	concurrentGithubRequests = 25

	// txtRecordPrefix is the prefix of valid _psl TXT records.
	txtRecordPrefix = "https://github.com/publicsuffix/list/pull/"
)

// prExpected is the set of changed suffixes we expect to see in a PR,
// given TXT record information.
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
	gh       github.Client
	resolver net.Resolver
	ctx      context.Context
	errs     []error
	// prExpected maps a Github PR number to the PSL changes we expect
	// to see in that PR.
	prExpected map[int]*prExpected

	// Somehow, x/unicode isn't thread-safe when collating?? Hacky
	// workaround for now, just serialize parses.
	parseMu sync.Mutex
}

// validateTXTRecords checks the TXT records of all Suffix and
// Wildcard blocks found under b.
func validateTXTRecords(ctx context.Context, b Block) (errs []error) {
	checker := txtRecordChecker{
		ctx:        ctx,
		prExpected: map[int]*prExpected{},
	}

	// TXT checking happens in two phases: first, look up all TXT
	// records and construct a suffix<>github PR map. Then, check
	// those Github PRs to verify that the suffixes were indeed
	// modified in those PRs.
	//
	// TXT record lookup is very parallel, so we use a taskgroup to do
	// (rate-limited) concurrent processing. On the other hand,
	// checking Github PRs requires hitting Github once for each PR,
	// so that is serialized to avoid tripping abuse protections.
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

	// Wait for all TXT lookups to complete. When that's done,
	// checker.prToSuffix and checker.domainsToCheck has complete
	// state for which PRs need to be checked, and we can move on.
	group.Wait()

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

// txtResult is the result of a DNS TXT lookup for Block.
type txtResult struct {
	Block
	prs []int
	err error
}

// checkTXT checks the _psl TXT record for domain. The given block is
// passed through in the txtResult so that the TXT lookup can be
// associated with the appropriate PSL entry. It should be a Suffix or
// a Wildcard, but checkTXT doesn't use it other than to include it in
// the txtResult.
func (c *txtRecordChecker) checkTXT(block Block, domain domain.Name) txtResult {
	log.Printf("Checking TXT for _psl.%s", domain.String())
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
		// _psl.<domain> exists, but contains no valid Github PR URLs.
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

// checkPRs looks up the Github PRs discovered during TXT record
// checking, and verifies that each PR is touching the suffix that
// claimed to be related to the PR.
func (c *txtRecordChecker) checkPR(prNum int, info *prExpected) []error {
	log.Printf("Checking PR %d", prNum)

	// Some PRs have broken state, so we can't use Github's API to
	// check them even though they _are_ a correct record for some
	// suffixes. Handle those first.
	//
	// There are no wildcard entries in this state, so we just check
	// suffixes.
	for _, suf := range info.suffixes {
		if acceptPRForDomain(suf.Domain, prNum) {
			delete(info.suffixes, suf.Domain.String())
		}
	}
	if len(info.suffixes) == 0 && len(info.wildcards) == 0 {
		return nil
	}

	var ret []error

	beforeBs, afterBs, err := c.gh.PSLForPullRequest(c.ctx, prNum)
	if err != nil {
		for _, suf := range info.suffixes {
			ret = append(ret, ErrTXTCheckFailure{suf, err})
		}
		for _, wild := range info.wildcards {
			ret = append(ret, ErrTXTCheckFailure{wild, err})
		}
		return ret
	}

	//c.parseMu.Lock()
	before, _ := Parse(beforeBs)
	after, _ := Parse(afterBs)
	after.SetBaseVersion(before, true)
	//c.parseMu.Unlock()

	for _, suf := range BlocksOfType[*Suffix](after) {
		if !suf.Changed() {
			continue
		}

		// If the changed suffix is in our prExpected list, remove
		// it. We found a change to the suffix in the PR that the
		// TXT record said, everything is good. This doesn't need
		// to be conditional, because delete() is a no-op if the
		// key isn't present in the map.
		delete(info.suffixes, suf.Domain.String())
	}

	for _, wild := range BlocksOfType[*Wildcard](after) {
		if !wild.Changed() {
			continue
		}

		delete(info.wildcards, wild.Domain.String())
	}

	// At this point we've eliminated all suffixes that were
	// changed by this PR. Whatever is left over are TXT records
	// that incorrectly claim PR #N is related to them.
	for _, suf := range info.suffixes {
		ret = append(ret, ErrTXTRecordMismatch{suf, prNum})
	}
	for _, wild := range info.wildcards {
		ret = append(ret, ErrTXTRecordMismatch{wild, prNum})
	}
	return ret
}

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

// extractPSLRecords extracts github PR numbers from the given TXT
// records. TXT records which do not match the required PSL record
// format are ignored.
func extractPSLRecords(txts []string) []int {
	// You might think that since we put PSL records under
	// _psl.<domain>, we would only ever see well formed PSL
	// records. However, a large number of domains return SPF and
	// other "well known" records as well, likely because their DNS
	// server is hardcoded to return those for any TXT query. So, we
	// have to gracefully ignore "malformed" TXT records here.
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
				// At least one _psl record points to a bogus zero PR
				// - presumably a placeholder that was never
				// updated. Treat it identically to other
				// missing/malformed records.
				continue
			}

			// Apply special cases where the PR in DNS is not quite
			// right, but not due to procedural issues on the PSL side
			// rather than suffix owner error. See exceptions.go for
			// more explanation.
			prNum = adjustTXTPR(prNum)

			ret = append(ret, prNum)
		}
	}
	return ret
}
