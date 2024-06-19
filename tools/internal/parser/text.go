package parser

import (
	"fmt"
	"strings"
)

// Source is a piece of source text with location information.
//
// A Source is effectively a slice of the input file's lines, with
// some extra information attached. As such, the start/end indexes
// behave the same as in Go slices, and select the half-open interval
// [start:end).
type Source struct {
	// The lines of source text.
	lines []string
	// lineOffset is how many lines are before the beginning of lines,
	// for sources that represent a subset of the input.
	lineOffset int
}

// Text returns the source text of s as a string.
func (s Source) Text() string {
	if len(s.lines) == 1 {
		return s.lines[0]
	}
	return strings.Join(s.lines, "\n")
}

// LocationString returns a short string describing the source
// location.
func (s Source) LocationString() string {
	// For printing diagnostics, 0-indexed [start:end) is confusing
	// and not how editors present text to people. Adjust the offsets
	// to be 1-indexed [start:end] instead.
	start := s.lineOffset + 1
	end := s.lineOffset + len(s.lines)

	if end < start {
		// Zero line Source. We can sometimes produce these internally
		// during parsing, but they should not escape outside the
		// package. We still print them gracefully instead of
		// panicking, because it's useful for debugging the parser.
		return fmt.Sprintf("<invalid Source, 0-line range before line %d>", start)
	}

	if start == end {
		return fmt.Sprintf("line %d", start)
	}
	return fmt.Sprintf("lines %d-%d", start, end)
}

// newSource returns a Source for src.
func newSource(src string) Source {
	return Source{
		lines:      strings.Split(src, "\n"),
		lineOffset: 0,
	}
}

// slice returns the slice of s between startLine and endLine.
//
// startLine and endLine behave like normal slice offsets, i.e. they
// represent the half-open range [startLine:endLine).
func (s Source) slice(startLine, endLine int) Source {
	if startLine < 0 || startLine > len(s.lines) || endLine < startLine || endLine > len(s.lines) {
		panic("invalid input to slice")
	}
	return Source{
		lines:      s.lines[startLine:endLine],
		lineOffset: s.lineOffset + startLine,
	}
}

// line returns the nth line of s.
func (s Source) line(n int) Source {
	return s.slice(n, n+1)
}

// lineSources slices s into one Source per line.
func (s Source) lineSources() []Source {
	if len(s.lines) == 1 {
		return []Source{s}
	}

	ret := make([]Source, len(s.lines))
	for i := range s.lines {
		ret[i] = s.slice(i, i+1)
	}
	return ret
}

// cut slices s at the first cut line, as determined by cutHere. It
// returns two Source blocks: the part of s before the cut line, and
// the rest of s including the cut line. The found result reports
// whether a cut was found. If s does not contain a cut line, cut
// returns s, <invalid>, false.
func (s Source) cut(cutHere func(Source) bool) (before Source, rest Source, found bool) {
	for i := range s.lines {
		if cutHere(s.line(i)) {
			return s.slice(0, i), s.slice(i, len(s.lines)), true
		}
	}
	return s, Source{}, false
}

// split slices s into all sub-blocks separated by lines identified by
// isSeparator, and returns a slice of the non-empty blocks between
// those separators.
//
// Note the semantics are different from strings.Split: sub-blocks
// that contain no lines are not returned. This works better for what
// the PSL format needs.
func (s Source) split(isSeparator func(line Source) bool) []Source {
	ret := []Source{}
	s.forEachRun(isSeparator, func(block Source, isSep bool) {
		if isSep {
			return
		}
		ret = append(ret, block)
	})
	return ret
}

// forEachRun calls processBlock for every run of consecutive lines
// where classify returns the same result.
//
// For example, if classify returns true on lines starting with "//",
// processBlock gets called with alternating blocks consisting of only
// comments, or only non-comments.
func (s Source) forEachRun(classify func(line Source) bool, processBlock func(block Source, classifyResult bool)) {
	if len(s.lines) == 0 {
		return
	}

	currentBlock := 0
	currentVal := classify(s.line(0))
	for i := range s.lines[1:] {
		line := i + 1
		v := classify(s.line(line))
		if v != currentVal {
			processBlock(s.slice(currentBlock, line), currentVal)
			currentVal = v
			currentBlock = line
		}
	}
	if currentBlock != len(s.lines) {
		processBlock(s.slice(currentBlock, len(s.lines)), currentVal)
	}
}
