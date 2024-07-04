package parser

import (
	"fmt"
	"slices"
	"strings"

	"github.com/creachadair/mds/slice"
)

// ValidateOffline runs offline validations on a parsed PSL.
func ValidateOffline(l *List) []error {
	var ret []error

	for _, block := range BlocksOfType[*Section](l) {
		if block.Name == "PRIVATE DOMAINS" {
			ret = append(ret, validateEntityMetadata(block)...)
			if err := validatePrivateSectionOrder(block); err != nil {
				ret = append(ret, err)
			}
			break
		}
	}

	return ret
}

// validateEntityMetadata verifies that all suffix blocks have some
// kind of entity name.
func validateEntityMetadata(block *Section) []error {
	var ret []error
	for _, block := range BlocksOfType[*Suffixes](block) {
		if block.Info.Name == "" {
			ret = append(ret, ErrMissingEntityName{
				Suffixes: block,
			})
		} else if block.Info.Submitter == nil && !exemptFromContactInfo(block.Info.Name) {
			ret = append(ret, ErrMissingEntityEmail{
				Suffixes: block,
			})
		}
	}
	return ret
}

const (
	amazonSuperblockStart = "Amazon : https://www.amazon.com"
	amazonSuperblockEnd   = "concludes Amazon"
)

// validatePrivateSectionOrder verifies that the blocks in the private
// domains section is sorted according to PSL policy.
func validatePrivateSectionOrder(block *Section) error {
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
	for _, block := range block.Children() {
		switch v := block.(type) {
		case *Comment:
			if !inAmazonSuperblock && strings.Contains(v.Text[0], amazonSuperblockStart) {
				// Start of the Amazon superblock. We will accumulate
				// suffix blocks into here further down.
				inAmazonSuperblock = true
				blocks = append(blocks, superblock{
					Name: "Amazon",
				})
			} else if inAmazonSuperblock && strings.Contains(v.Text[0], amazonSuperblockEnd) {
				// End of Amazon superblock, go back to normal
				// behavior.
				inAmazonSuperblock = false
			}
		case *Suffixes:
			if inAmazonSuperblock {
				last := len(blocks) - 1
				blocks[last].Suffixes = append(blocks[last].Suffixes, v)
			} else if !exemptFromSorting(v.Info.Name) {
				blocks = append(blocks, superblock{v.Info.Name, []*Suffixes{v}})
			}
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
		return nil
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

	err := ErrSuffixBlocksInWrongPlace{
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
			insertAfter = suffixesOfPrev[len(suffixesOfPrev)-1].Info.Name
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
				Name:        block.Info.Name,
				InsertAfter: insertAfter,
			})
		}

		// blocks and sorted are out of sync, only advance blocksIdx
		// until we resynchronize.
		blocksIdx++
	}

	return err
}
