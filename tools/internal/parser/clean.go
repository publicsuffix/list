package parser

import (
	"fmt"
	"slices"
	"strings"

	"github.com/publicsuffix/list/tools/internal/domain"
)

// Clean cleans the list, editing its contents as necessary to conform
// to PSL style rules such as ordering of suffixes.
//
// Clean does not make any semantic changes. The list after cleaning
// represents exactly the same set of public suffixes and attached
// metadata.
//
// If the list's structure prevents Clean from applying some necessary
// changes, Clean applies as many changes as possible then returns
// errors describing the cleanups that could not take place.
func (l *List) Clean() []error {
	return cleanBlock(l)
}

func cleanBlock(b Block) []error {
	var ret []error

	switch v := b.(type) {
	case *List:
		for _, child := range v.Blocks {
			ret = append(ret, cleanBlock(child)...)
		}
	case *Section:
		switch v.Name {
		case "PRIVATE DOMAINS":
			for _, child := range v.Blocks {
				ret = append(ret, cleanBlock(child)...)
			}
			// The private domains section must be sorted according to
			// the names of the maintainers of suffix blocks.
			ret = append(ret, sortSection(v)...)
		case "ICANN DOMAINS":
			// We don't currently clean the ICANN section, since it
			// has a lot of historical and non-normalized data. We
			// want to focus on the private domains section first.
			//
			// TODO: when we're ready, apply the same cleaning as the
			// private section.
		}
	case *Suffixes:
		for _, child := range v.Blocks {
			ret = append(ret, cleanBlock(child)...)
		}
		ret = append(ret, sortSuffixes(v)...)
		rewriteSuffixesMetadata(v)
	case *Wildcard:
		cleanWildcard(v)
	case *Comment, *Suffix:
		// No cleaning required
	default:
		panic("unknown ast node")
	}

	return ret
}

func cleanWildcard(w *Wildcard) {
	// Sort and deduplicate exception domains.
	slices.SortFunc(w.Exceptions, domain.Label.Compare)
	w.Exceptions = slices.Compact(w.Exceptions)
}

func sortSection(s *Section) []error {
	// There are two difficult parts aspects to sorting a section:
	// free-floating comments, and Amazon.
	//
	// We cannot safely reorder suffix blocks across a free-floating
	// comment, because we don't know if that breaks human-readable
	// meaning (e.g. a comment that says "all companies with 'cloud'
	// anywhere in their names come after this"). These comments act
	// as barriers that prevent us from moving suffixes across it.
	//
	// We can still fix some sorting problems however. We do this by
	// sorting each group of suffix blocks that come between
	// free-floating comments, and check (but don't fix) that the
	// order between the groups is correct. This means we fix as many
	// ordering issues as possible, and report errors for the ones we
	// can't fix without help.
	//
	// Amazon has a semi-automated group of suffix blocks, which are
	// in the PSL at the correct sort location for "Amazon" but are
	// not correctly sorted relative to non-Amazon blocks. This causes
	// a lot of churn in the sorting for what is currently an edge
	// case.
	//
	// For a smoother transition, we treat the entire amazon group as
	// a single logical block for sorting, so that it stays in the
	// correct position for the name "Amazon" but can do whatever it
	// likes within.
	//
	// TODO: now that we have automated formatting, maybe stop
	// treating Amazon specially? Machine edits can just update the
	// Amazon entries and re-Clean as needed.
	type group struct {
		Key    string // name of block maintainer, or "" for barrier comments
		Blocks []Block
	}
	var groups []group

	const (
		amazonGroupStart = "Amazon : https://www.amazon.com"
		amazonGroupEnd   = "concludes Amazon"
	)

	inAmazonGroup := false
	for _, block := range s.Blocks {
		switch v := block.(type) {
		case *Comment:
			switch {
			case !inAmazonGroup && strings.Contains(v.Text[0], amazonGroupStart):
				// Start of the Amazon group. We will accumulate
				// suffix blocks into here further down.
				inAmazonGroup = true
				groups = append(groups, group{
					Key:    "Amazon",
					Blocks: []Block{block},
				})
			case inAmazonGroup:
				last := len(groups) - 1
				groups[last].Blocks = append(groups[last].Blocks, block)
				if strings.Contains(v.Text[0], amazonGroupEnd) {
					// End of Amazon group, go back to normal
					// behavior.
					inAmazonGroup = false
				}
			default:
				// Barrier comment
				groups = append(groups, group{
					Key:    "",
					Blocks: []Block{block},
				})
			}
		case *Suffixes:
			if inAmazonGroup {
				last := len(groups) - 1
				groups[last].Blocks = append(groups[last].Blocks, v)
			} else {
				groups = append(groups, group{v.Info.Name, []Block{block}})
			}
		default:
			panic("unknown ast node")
		}
	}

	// Scan through the groups, looking for barrier comments. Sort the
	// groups between the barriers, and just check ordering across
	// barriers.
	var (
		errs           []error
		prevGroupEnd   *Suffixes
		prevComment    *Comment
		thisGroupStart int
	)

	sortAndCheck := func(groups []group) {
		if len(groups) == 0 {
			return
		}

		slices.SortFunc(groups, func(a, b group) int {
			return compareCommentText(a.Key, b.Key)
		})

		if prevGroupEnd == nil {
			// First group.
			lastGroup := groups[len(groups)-1]
			prevGroupEnd = lastGroup.Blocks[len(lastGroup.Blocks)-1].(*Suffixes)
		} else if compareCommentText(prevGroupEnd.Info.Name, groups[0].Key) <= 0 {
			// Inter-group order is correct.
			lastGroup := groups[len(groups)-1]
			prevGroupEnd = lastGroup.Blocks[len(lastGroup.Blocks)-1].(*Suffixes)
		} else {
			// Wrong inter-group order. Report error and keep the same
			// prevGroupEnd, since it's bigger.
			errs = append(errs, ErrCommentPreventsSectionSort{prevComment.SourceRange})
		}
	}

	for i, group := range groups {
		if group.Key != "" {
			// Not a boundary, keep going.
			continue
		}

		// Found a boundary.
		sortAndCheck(groups[thisGroupStart:i])
		prevComment = group.Blocks[0].(*Comment)
		thisGroupStart = i + 1
	}
	if thisGroupStart != len(groups) {
		sortAndCheck(groups[thisGroupStart:])
	}

	// Reassemble the new s.Blocks from groups. Note, we must not
	// reuse the s.Blocks slice, because all the slices in the groups
	// are referencing the same backing array. If we start overwriting
	// that array, we might corrupt future groups and end up with a
	// list that deletes a bunch of suffix blocks and duplicates a
	// bunch of others in the wrong place.
	ret := make([]Block, 0, len(s.Blocks))
	for _, group := range groups {
		ret = append(ret, group.Blocks...)
	}
	s.Blocks = ret

	return errs
}

func sortSuffixes(s *Suffixes) []error {
	// Suffix sorting has the same problem as section sorting: inline
	// comments act as barriers that prevent movement of a suffix
	// across them (e.g. "all 3rd-level public suffixes come after
	// this").
	//
	// We do the same thing as for section sorting: sort as much as we
	// can, report errors for the rest.

	var (
		errs           []error
		prevGroupEnd   Block    // last suffix/wildcard of previous group
		prevComment    *Comment // last Comment seen, for error reporting
		thisGroupStart int
		// We have to construct a new output slice, because the
		// deduplicating sort below might shrink s.Blocks.
		out = make([]Block, 0, len(s.Blocks))
	)

	sortAndCheck := func(group []Block) {
		if len(group) == 0 {
			return
		}

		// Sort and deduplicate. Within a single group inside a suffix
		// block, duplicate suffixes are semantically equivalent so
		// it's safe to remove dupes.
		slices.SortFunc(group, compareSuffixAndWildcard)
		group = slices.CompactFunc(group, func(a, b Block) bool {
			return compareSuffixAndWildcard(a, b) == 0
		})
		out = append(out, group...)

		if prevGroupEnd == nil {
			// First group.
			prevGroupEnd = group[len(group)-1]
		} else if compareSuffixAndWildcard(prevGroupEnd, group[0]) <= 0 {
			// Correct order.
			prevGroupEnd = group[len(group)-1]
		} else {
			errs = append(errs, ErrCommentPreventsSuffixSort{prevComment.SourceRange})
			// Keep the same prevGroupEnd, since it's the
			// largest value seen so far and future groups
			// _should_ sort after it, if future comments
			// aren't a problem.
		}
	}

	for i, b := range s.Blocks {
		switch v := b.(type) {
		case *Suffix, *Wildcard:
			continue
		case *Comment:
			group := s.Blocks[thisGroupStart:i]
			if len(group) > 0 {
				sortAndCheck(group)
				prevComment = v
			}
			out = append(out, v)
			thisGroupStart = i + 1
		default:
			panic("unknown ast node")
		}
	}
	if thisGroupStart != len(s.Blocks) {
		sortAndCheck(s.Blocks[thisGroupStart:])
	}

	s.Blocks = out

	return errs
}

// compareSuffixAndWildcard compares a and b, which can be any
// combination of Suffix and Wildcard structs. Returns -1 if a < b, +1
// if a > b, and 0 if a == b.
func compareSuffixAndWildcard(a, b Block) int {
	var da, db domain.Name
	var wilda, wildb bool
	switch v := a.(type) {
	case *Suffix:
		da = v.Domain
	case *Wildcard:
		da = v.Domain
		wilda = true
	default:
		panic(fmt.Sprintf("can't compare non-suffix type %T", a))
	}
	switch v := b.(type) {
	case *Suffix:
		db = v.Domain
	case *Wildcard:
		db = v.Domain
		wildb = true
	default:
		panic(fmt.Sprintf("can't compare non-suffix type %T", b))
	}

	if ret := da.Compare(db); ret != 0 {
		return ret
	}

	// Strings are equal. If one of the inputs is a wildcard, it goes
	// after any non-wildcards.
	if wilda == wildb {
		return 0
	} else if wilda {
		return +1
	} else {
		return -1
	}
}

// rewriteSuffixesMetadata rewrites the raw metadata comment inside s,
// but only if the information in s.Info no longer matches the
// contents of the parsed comment.
//
// Comments whose information still matches are left untouched. In
// other words, this applies edits that have been made in s.Info, but
// existing comments aren't gratuitously reformatted.
func rewriteSuffixesMetadata(s *Suffixes) {
	inf := s.Info
	var (
		out        *Comment
		hasOldInfo bool
	)

	// We need to handle a bunch of annoying cases: a comment could
	// exist or not in the AST, info may have changed or not
	// (including deletion of all info), and the info may not be
	// machine-editable in the first place.
	if len(s.Blocks) > 0 {
		out, hasOldInfo = s.Blocks[0].(*Comment)
	}
	switch {
	case !inf.MachineEditable:
		// Re-extract info from the comment, to resync the raw and
		// parsed data. Note extraction works on nil comments, so this
		// one case covers all options of non-machine editable data.
		s.Info = extractMaintainerInfo(out)
		return
	case hasOldInfo && inf.HasInfo():
		// Skip update if the existing info matches what's already
		// there, even if it's not in canonical format.
		oldInf := extractMaintainerInfo(out)
		if inf.Compare(&oldInf) == 0 {
			return
		}
	case hasOldInfo && !inf.HasInfo():
		// Unusual, but a comment existed and we need to delete it.
		s.Blocks = slices.Delete(s.Blocks, 0, 1)
		return
	case !hasOldInfo && inf.HasInfo():
		// Need to create a fresh comment. It will get populated
		// below.
		out = &Comment{}
		s.Blocks = slices.Insert(s.Blocks, 0, Block(out))
	case !hasOldInfo && !inf.HasInfo():
		// No comment, and no data. Nothing to do.
		return
	default:
		// The above should handle all possible states. This is just a
		// safeguard to enforce exhaustiveness.
		panic("unreachable")
	}

	// Clear out the source range and previous text. If anything wants
	// to refer to this block, we don't want to confuse the user by
	// pointing at the original source location even though we've
	// rewritten the entire comment to something else.
	out.SourceRange = SourceRange{}
	out.Text = out.Text[:0]

	urls := inf.URLs
	if inf.Name != "" && len(urls) > 0 {
		// First line that looks like "<name> : <url>"
		out.Text = append(out.Text, fmt.Sprintf("%s : %s", inf.Name, urls[0]))
		urls = urls[1:]
	} else if inf.Name != "" {
		// First line that looks like "<name>"
		out.Text = append(out.Text, inf.Name)
	}

	for _, u := range urls {
		out.Text = append(out.Text, u.String())
	}
	for _, m := range inf.Maintainers {
		// We could use m.String(), but doing that quotes the name,
		// and ends up inserting a lot of escape chars for non-ascii
		// names. The mail package can parse unquoted names just fine,
		// so we prefer the form that is more human readable.
		emailStr := strings.TrimSpace(fmt.Sprintf("%s <%s>", m.Name, m.Address))
		out.Text = append(out.Text, strings.TrimSpace(fmt.Sprintf("Submitted by %s", emailStr)))
	}
	for _, o := range inf.Other {
		out.Text = append(out.Text, o)
	}
}
