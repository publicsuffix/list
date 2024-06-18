package parser

import (
	"strings"
)

// source is a block of zero or more lines of source text. It can be
// decomposed into smaller sub-blocks using its parsing methods. It
// also preserves source line information and can be turned into an
// exported Source when needed.
type source struct {
	// The lines of source text.
	lines []string
	// lineOffset is how many lines are before the beginning of lines,
	// for sources that represent a subset of the input.
	lineOffset int
}

// newSource returns a source for src.
func newSource(src string) source {
	return source{
		lines:      strings.Split(src, "\n"),
		lineOffset: 0,
	}
}

// Source returns the exported Source counterpart of s.
func (s source) Source() Source {
	if len(s.lines) == 0 {
		// Letting a zero-byte source range escape from the parser
		// package is the sign of a bug somewhere, so blow up and make
		// it visible.
		panic("can't construct a zero-line Source")
	}
	start := s.lineOffset + 1
	end := start + len(s.lines) - 1
	return Source{
		StartLine: start,
		EndLine:   end,
		Raw:       s.Text(),
	}
}

// slice returns the slice of s between startLine and endLine.
//
// startLine and endLine behave like normal slice offsets, i.e. they
// represent the half-open range [startLine:endLine).
func (s source) slice(startLine, endLine int) source {
	if startLine < 0 || startLine > len(s.lines) || endLine < startLine || endLine > len(s.lines) {
		panic("invalid input to slice")
	}
	return source{
		lines:      s.lines[startLine:endLine],
		lineOffset: s.lineOffset + startLine,
	}
}

// Line returns the nth line of s.
func (s source) line(n int) source {
	return s.slice(n, n+1)
}

// Lines slices s into one source per line.
func (s source) Lines() []source {
	if len(s.lines) == 1 {
		return []source{s}
	}

	ret := make([]source, len(s.lines))
	for i := range s.lines {
		ret[i] = s.slice(i, i+1)
	}
	return ret
}

// Text returns the source text of s as a string.
func (s source) Text() string {
	if len(s.lines) == 1 {
		return s.lines[0]
	}
	return strings.Join(s.lines, "\n")
}

// Cut slices s at the first cut line, as determined by cutHere. It
// returns two source blocks: the part of s before the cut line, and
// the rest of s including the cut line. The found result reports
// whether a cut was found. If s does not contain a cut line, Cut
// returns s, <invalid>, false.
func (s source) Cut(cutHere func(source) bool) (before source, rest source, found bool) {
	for i := range s.lines {
		if cutHere(s.line(i)) {
			return s.slice(0, i), s.slice(i, len(s.lines)), true
		}
	}
	return s, source{}, false
}

// Split slices s into all sub-blocks separated by lines identified by
// isSeparator, and returns a slice of the non-empty blocks between
// those separators.
//
// Note the semantics are different from strings.Split: sub-blocks
// that contain no lines are not returned. This works better for what
// the PSL format needs.
func (s source) Split(isSeparator func(line source) bool) []source {
	ret := []source{}
	s.ForEachRun(isSeparator, func(block source, isSep bool) {
		if isSep {
			return
		}
		ret = append(ret, block)
	})
	return ret
}

// ForEachRun calls processBlock for every run of consecutive lines
// where classify returns the same result.
//
// For example, if classify returns true on lines starting with "//",
// processBlock gets called with alternating blocks consisting of only
// comments, or only non-comments.
func (s source) ForEachRun(classify func(line source) bool, processBlock func(block source, classifyResult bool)) {
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
