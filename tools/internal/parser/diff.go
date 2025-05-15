package parser

import (
	"fmt"
)

// SetBaseVersion sets the list's base of comparison to old, and
// updates the changed/unchanged annotations on all Blocks to match.
//
// If wholeSuffixBlocks is true, any changed Suffix or Wildcard within
// a Suffixes block marks all suffixes and wildcards in that block as
// changed.
//
// Precise marking (wholeSuffixBlocks=false) is intended for
// maintainer and machine edits, where change-aware validators should
// exaine only the specific changed items.
//
// Expansive marking (wholeSuffixBlocks=true) is intended for external
// PRs from suffix block owners, to opportunistically point out more
// issues that they have the knowledge and authority to fix.
func (l *List) SetBaseVersion(old *List, wholeSuffixBlocks bool) {
	diff := differ{
		oldCnt:    map[string]int{},
		inCurrent: map[string][][]Block{},
		keys:      map[Block]string{},

		wholeSuffixBlocks: wholeSuffixBlocks,
	}

	// Tree diff is an open area of research, and it's possible to use
	// extremely fancy (and slow) algorithms. Thankfully, the PSL has
	// some additional domain-specific properties that let us take
	// shortcuts and implement something O(n).
	//
	// First, academic tree_diff(OLD,NEW) produces an "edit script" as
	// the output, which describes how to add, delete, move and mutate
	// tree nodes to transform the OLD tree into the NEW tree. For the
	// PSL, we don't care about the exact structural changes, we just
	// need to know if we can skip validation checks. So we have to
	// answer a simple question: is a given block in NEW also present
	// in OLD?
	//
	// Second, all nodes in a well-formed list have a stable unique
	// identity. We can use this to answer the previous question in
	// constant time, instead of having to do complex tree analysis to
	// locate equivalent nodes.
	//
	// Node identities may be duplicated in an ill-formed List, for
	// example a suffix block that lists the same suffix twice. We
	// deal with this using brute force, and mark all duplicate
	// identities as changed. This means that a malformed PSL file
	// might report more changes than the strict minimum, but in
	// practice it's not much more, and in exchange we don't have to
	// do anything complex to decide what to revalidate.
	//
	// Third, how do we propagate child changes to parents? This is
	// where academic algorithms quickly go into O(n^3)
	// territory. Once again, we avoid this with brute force: a
	// changed tree node marks all its parents as changed as
	// well. That means that if you fix a typo in one Suffix, we say
	// that the Suffix changed, but also its parent Suffixes, Section,
	// and List nodes.
	//
	// We could theoretically dirty fewer nodes in some cases, but
	// that introduces a risk of false negatives (we forget to re-run
	// a necessary validation), and it makes the diff harder to reason
	// about when writing validators. In practice, this slightly
	// pessimistic dirtying is cheap for the currently-planned
	// validators, so we stick with the behavior that is easy to
	// reason about and simple to implement.
	//
	// Finally, we need to do something about deleted nodes. We can
	// handle that with a single additional pass through the OLD list,
	// thanks to the node identity property. Again for simplicity, we
	// treat deletions similar to edits: all the parents of a deleted
	// node are marked dirty. Again we could be more precise here, but
	// in practice it's currently cheap to be pessimistic, and makes
	// the code and mental model simpler.
	//
	// There are various optimizations possible for this code. The
	// biggest would be doing something more efficient to track block
	// identities, which are currently expressed as big strings
	// because that makes them convenient to compare and use as map
	// keys. However, this algorithm as currently implemented takes
	// <100ms to diff a full PSL file, so for now we err on the side
	// of simplicity.

	// Compile the identities of all the blocks in old.
	diff.scanOld(old, "")
	// Mark unchanged blocks. Thanks to the previous step, each tree
	// node can be checked in O(1) time.
	diff.scanCurrent(l, "", nil)
	// Dirty the parents of deleted blocks.
	diff.markDeletions(old, "")
}

type differ struct {
	// wholeSuffixBlocks is whether Suffix/Wildcard changes propagate
	// to all children of the parent Suffixes block.
	wholeSuffixBlocks bool

	// oldCnt counts the number of blocks in the old list with a given
	// identity key.
	oldCnt map[string]int

	// inCurrent maps block identity keys to the tree paths in of the
	// current list with that identity. Given a block with identity K,
	// inCurrent[K] is a list of paths. In each path, path[0] is a
	// block with identity K, and path[1..n] are its parents going
	// back to the root of the tree.
	//
	// In a well-formed List, each cache entry has a single path, but
	// we track duplicates in order to function correctly on malformed
	// lists as well.
	inCurrent map[string][][]Block

	// keys caches identity keys by block pointer. There are several
	// passes of traversal through trees, and when old and current are
	// nearly identical (the common case) this can save significant
	// CPU time.
	keys map[Block]string
}

// scanOld records b and its children in d.oldCnt.
func (d *differ) scanOld(b Block, parentKey string) {
	k := d.getKey(b, parentKey)
	d.oldCnt[k]++
	for _, child := range b.Children() {
		d.scanOld(child, k)
	}
}

// scanCurrent adds b and all its children to b.inCurrent, and updates
// their isUnchanged annotation based on the information in d.oldCnt.
func (d *differ) scanCurrent(curBlock Block, parentKey string, parents []Block) {
	k := d.getKey(curBlock, parentKey)

	path := make([]Block, 0, len(parents)+1)
	path = append(path, curBlock)
	path = append(path, parents...)

	// Assume we're unchanged to start with. The job of the remaining
	// diff code is to falsify this claim and mark the node as changed
	// if needed.
	//
	// Setting this early and unconditionally lets us optimize the
	// logic in markChanged, by ensuring that each node transitions
	// false->true only once, before any possible true->false
	// transitions that affect it.
	curBlock.info().isUnchanged = true

	// Record the path to the current block, and if it's a
	// doppelganger of some other Block, mark changed. Tracking diffs
	// of duplicates requires solving some hard theoretical problems
	// of tree diff, so we don't bother.
	//
	// Duplicate identities only happens on a malformed PSL, and we
	// can save a lot of pain by just over-rechecking such PSLs
	// slightly.
	d.inCurrent[k] = append(d.inCurrent[k], path)
	if l := len(d.inCurrent[k]); l == 2 {
		// This is the first duplicate, previous path didn't know it
		// wasn't unique. Mark both the current and earlier path as
		// changed.
		d.markChanged(d.inCurrent[k]...)
	} else if l > 2 {
		// Previous paths already marked, only curBlock's one needs
		// updating.
		d.markChanged(path)
	}

	// This covers both the case where a block is new (oldCnt of 0),
	// and the case where this block isn't a dupe in current, but was
	// a dupe in old. In that case, like above we avoid algorithmic
	// headaches by just dirtying the block instead of trying to
	// resolve which version of the old dupes we're looking at.
	if d.oldCnt[k] != 1 {
		d.markChanged(path)
	}

	// Scan through child subtrees. These subtrees may call
	// markChanged and set Unchanged=false on us.
	for _, child := range curBlock.Children() {
		d.scanCurrent(child, k, path)
	}

	// If the caller requested, and we're changed anyway, see if we
	// should propagate the change back downwards again.
	if !curBlock.info().isUnchanged {
		d.maybeMarkWholeSuffixBlock(path)
	}
}

// markDeletions marks parents of deleted nodes as changed in current.
//
// For example, if the diff contains a suffix deletion, this will mark
// the enclosing Suffixes block as changed.
func (d *differ) markDeletions(oldBlock Block, parentKey string) bool {
	k := d.getKey(oldBlock, parentKey)

	pathsInCurrent, ok := d.inCurrent[k]
	if !ok {
		// oldBlock was deleted, report to caller.
		return true
	}

	childDeleted := false
	for _, child := range oldBlock.Children() {
		if d.markDeletions(child, k) {
			// Note, can't short-circuit here because there may be
			// other paths under this block that also need to be
			// updated. We're not only trying to update oldBlock, but
			// also all of its children.
			childDeleted = true
		}
	}

	// Children were deleted, mark ourselves changed. This implicitly
	// also marks the parent as changed, so no need to tell it that a
	// change happened, it'll just do extra no-op work.
	if childDeleted {
		d.markChanged(pathsInCurrent...)
	}

	return false
}

// maybeMarkWholeSuffixBlock calls markSuffixAndWildcardChanged on all
// Suffixes in path, if the caller of MarkUnchanged requested
// expansive marking.
func (d *differ) maybeMarkWholeSuffixBlock(path []Block) {
	if !d.wholeSuffixBlocks {
		return
	}

	switch path[0].(type) {
	case *Suffixes, *Suffix, *Wildcard:
		for i, parent := range path {
			if _, ok := parent.(*Suffixes); ok {
				d.markSuffixAndWildcardChanged(parent, path[i+1:])
			}
		}
	}
}

// markSuffixAndWildcardChanged marks as changed all Suffix and
// Wildcard blocks in the tree rooted at curBlock.
func (d *differ) markSuffixAndWildcardChanged(curBlock Block, parents []Block) {
	path := append([]Block{curBlock}, parents...)

	switch curBlock.(type) {
	case *Suffix, *Wildcard:
		d.markChanged(path)
	default:
		for _, child := range curBlock.Children() {
			d.markSuffixAndWildcardChanged(child, path)
		}
	}
}

// markChanged marks as changed all the blocks in paths.
func (d *differ) markChanged(paths ...[]Block) {
pathLoop:
	for _, path := range paths {
		for _, b := range path {
			if b.info().isUnchanged == false {
				// We never mark a node as changed in isolation, we
				// always propagate the change to all its
				// parents. Therefore, we can stop the upwards
				// traversal in this path as soon as we find any node
				// that's already in the correct state.
				continue pathLoop
			}
			b.info().isUnchanged = false
		}
	}
}

// getKey returns the identity key for blk, which must be a direct
// child of parentKey. getKey keeps a cache of all keys built in the
// lifetime of this differ, to make future calls more efficient.
func (d *differ) getKey(blk Block, parentKey string) string {
	ret, ok := d.keys[blk]
	if !ok {
		ret = d.makeKey(blk, parentKey)
		d.keys[blk] = ret
	}
	return ret
}

// makeKey builds the identity key of blk, which must be a child node
// of parentKey.
func (d *differ) makeKey(b Block, parentKey string) string {
	switch v := b.(type) {
	case *List:
		return fmt.Sprintf("%s;List", parentKey)
	case *Section:
		return fmt.Sprintf("%s;Section,%q", parentKey, v.Name)
	case *Suffixes:
		// Note parsed suffix metadata isn't included in the identity,
		// to avoid marking all suffixes in a block changed when
		// someone adjusts their URL or email. Such edits will still
		// indirectly dirty the block, because the metadata comment
		// includes the entire comment text in its identity, and will
		// dirty the parent Suffixes.
		ret := fmt.Sprintf("%s;Suffixes,%q", parentKey, v.Info.Name)
		return ret
	case *Suffix:
		return fmt.Sprintf("%s;Suffix,%q", parentKey, v.Domain)
	case *Wildcard:
		return fmt.Sprintf("%s;Wildcard,%q,%#v", parentKey, v.Domain, v.Exceptions)
	case *Comment:
		return fmt.Sprintf("%s;Comment,%#v", parentKey, v.Text)
	default:
		panic("unknown ast node")
	}
}
