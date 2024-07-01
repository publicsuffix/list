package parser

import (
	"fmt"
	"slices"
	"strings"

	"github.com/creachadair/mds/slice"
)

// Validate runs validations on a parsed File.
//
// Validation only runs on a file that does not yet have any
// errors. The presence of errors may indicate structural issues that
// can break some validations.
func (p *parser) Validate() {
	if len(p.Errors) > 0 {
		return
	}

	p.requireEntityNames()
	p.requirePrivateDomainEmailContact()
	p.requireSortedPrivateSection()
}

// requireEntityNames verifies that all Suffix blocks have some kind
// of entity name.
func (p *parser) requireEntityNames() {
	for _, block := range p.AllSuffixBlocks() {
		if block.Entity == "" {
			p.addError(MissingEntityName{
				Suffixes: block,
			})
		}
	}
}

// requirePrivateDomainEmailContact verifies that all Suffix blocks in
// the private section have email contact information.
func (p *parser) requirePrivateDomainEmailContact() {
	for _, block := range p.File.SuffixBlocksInSection("PRIVATE DOMAINS") {
		if block.Submitter == nil {
			p.addError(MissingEntityEmail{
				Suffixes: block,
			})
		}
	}
}

const (
	amazonSuperblockStart = "Amazon : https://www.amazon.com"
	amazonSuperblockEnd   = "concludes Amazon"
)

// requireSortedPrivateSection verifies that the blocks in the private
// domains section is sorted according to PSL policy.
func (p *parser) requireSortedPrivateSection() {
	// Amazon has a semi-automated "superblock" of suffix blocks,
	// which are in the PSL at the correct sort location for "Amazon",
	// but are not correctly interleaved with other non-Amazon
	// blocks. This causes a lot of churn in the sorting for what is
	// currently an edge case.
	//
	// For a smooth transition, this validation arranges suffix blocks
	// into superblocks, and only checks ordering between
	// superblocks. Currently, only the Amazon superblock contains
	// more than 1 suffix block.
	//
	// This is also where we filter out exempted blocks, which are
	// allowed to be in the wrong order.
	//
	// TODO: rework the parse tree structure so these "superblocks"
	// can be tracked more cleanly than this.
	type superblock struct {
		Name     string
		Suffixes []*Suffixes
	}
	var blocks []superblock

	inAmazonSuperblock := false
	for _, block := range allBlocksInPrivateSection(&p.File) {
		if comm, ok := block.(*Comment); ok {
			if !inAmazonSuperblock && strings.Contains(comm.Text(), amazonSuperblockStart) {
				// Start of the Amazon superblock. We will accumulate
				// suffix blocks into here further down.
				inAmazonSuperblock = true
				blocks = append(blocks, superblock{
					Name: "Amazon",
				})
			} else if inAmazonSuperblock && strings.Contains(comm.Text(), amazonSuperblockEnd) {
				// End of Amazon superblock, go back to normal
				// behavior.
				inAmazonSuperblock = false
			}
			continue
		}

		// Aside from the Amazon superblock comments, we only care
		// about Suffix blocks in this validation.
		suffixes, ok := block.(*Suffixes)
		if !ok {
			continue
		}

		// While we're inside the Amazon superblock, all suffix blocks
		// get grouped into one. Outside of the Amazon superblock,
		// each suffix block gets its own superblock.
		if inAmazonSuperblock {
			last := len(blocks) - 1
			blocks[last].Suffixes = append(blocks[last].Suffixes, suffixes)
			continue
		} else if exemptFromSorting(suffixes.Source) {
			continue
		} else {
			blocks = append(blocks, superblock{
				Name:     suffixes.Entity,
				Suffixes: []*Suffixes{suffixes},
			})
		}
	}

	// We need to know what order superblocks _should_ be in. This
	// comparison function tells us that, by comparing the names of
	// the block owners.
	compareSuperblocks := func(a, b superblock) int {
		return compareCommentText(a.Name, b.Name)
	}

	// Calculate the "longest non-decreasing subsequence" of
	// superblocks.
	//
	// Given a list that _should_ be sorted, the LNDS algorithm gives
	// us back the subset that _is_ correctly sorted. For example,
	// given the input [1 2 5 3 2 6 8], LNDS returns [1 2 2 6 8] and
	// the remaining elements [5, 3] need to be re-sorted.
	sorted := slice.LNDSFunc(blocks, compareSuperblocks)

	if len(sorted) == len(blocks) {
		// Already sorted, we're done.
		return
	}

	// Scan through the superblocks and find where the incorrectly
	// sorted blocks should go.
	//
	// Generating an error that a human can fix is difficult, because
	// describing an "edit script" to fix the input is tricky: we have
	// to describe the changes in a way that are unambiguous, and also
	// easy to follow. There are no _great_ solutions to this, but
	// empirically the least confusing option is to give people an
	// ordered sequence of "move X to Y", where each instruction can
	// assume that all previous instructions have already been
	// executed.
	//
	// To do this, we have to maintain 3 lists: the original input,
	// the "already sorted" subset we computed above, and the
	// "incrementally fixed" list where we've moved some elements to
	// the correct position. We can through the first 2 lists side by
	// side to identify which elements are currently misplaced, and we
	// use the 3rd list to work out the correct "move X to Y"
	// instruction.
	//
	// TODO: all this complexity exists because we're forcing humans
	// to sort text, instead of giving them an automatic formatter
	// that does the work. We can delete all this when we have an
	// automatic formatter.

	// The "incrementally fixed" list starts out as a copy of
	// `sorted`. Preallocate storage for the final size, so that the
	// sorted insertions don't have to reallocate.
	fixed := make([]superblock, 0, len(blocks))
	fixed = append(fixed, sorted...)

	err := SuffixBlocksInWrongPlace{
		EditScript: make([]MoveSuffixBlock, 0, len(blocks)-len(sorted)),
	}

	// Indexes into `blocks` and `sorted` respectively, for the
	// synchronized traversal. Note that sortedIdx <= blocksIdx
	// always, because by definition `sorted` is a subset of `blocks`.
	blocksIdx, sortedIdx := 0, 0
	for blocksIdx < len(blocks) {
		if sortedIdx < len(sorted) && compareSuperblocks(blocks[blocksIdx], sorted[sortedIdx]) == 0 {
			// Current positions in the two lists are in sync, we are
			// currently looking at an already-sorted part of
			// `blocks`. Nothing to do, just keep advancing.
			blocksIdx++
			sortedIdx++
			continue
		}

		// The two lists are out of sync. blocks[blocksIdx] is one of
		// the incorrectly sorted elements. Record the necessary
		// movement.
		toMove := blocks[blocksIdx]

		targetIdx, _ := slices.BinarySearchFunc(fixed, toMove, compareSuperblocks)
		fixed = slices.Insert(fixed, targetIdx, toMove)

		insertAfter := ""
		if targetIdx > 0 {
			suffixesOfPrev := fixed[targetIdx-1].Suffixes
			insertAfter = suffixesOfPrev[len(suffixesOfPrev)-1].Entity
		}

		// Superblocks can contain many suffixes. Move entire
		// superblocks as one.
		if len(toMove.Suffixes) > 1 {
			err.EditScript = append(err.EditScript, MoveSuffixBlock{
				Name:        fmt.Sprintf(`%s (all blocks until "concludes ..." comment)`, toMove.Name),
				InsertAfter: insertAfter,
			})
		} else {
			block := toMove.Suffixes[0]
			err.EditScript = append(err.EditScript, MoveSuffixBlock{
				Name:        block.Entity,
				InsertAfter: insertAfter,
			})
		}

		// blocks and sorted are out of sync, only advance blocksIdx
		// until we resynchronize.
		blocksIdx++
	}

	// At last, we can report the ordering error.
	p.addError(err)
}

func allBlocksInPrivateSection(f *File) []Block {
	start := 0
	for i, block := range f.Blocks {
		switch v := block.(type) {
		case *StartSection:
			if v.Name != "PRIVATE DOMAINS" {
				continue
			}
			start = i + 1
		case *EndSection:
			if v.Name != "PRIVATE DOMAINS" {
				continue
			}
			return f.Blocks[start:i]
		}
	}
	// We can only get here if there's no private section (so nothing
	// to validate), or if the file has structural issues (but we
	// don't run validations in that case).
	return []Block{}
}
