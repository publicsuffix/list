// Package parser implements a validating parser for the PSL files.
package parser

import (
	"fmt"
	"strings"

	"github.com/publicsuffix/list/tools/internal/domain"
)

// Parse parses bs as a PSL file and returns the parse result.
//
// The parser tries to keep going when it encounters errors. Parse and
// validation errors are accumulated in the Errors field of the
// returned File.
//
// If the returned File has a non-empty Errors field, the parsed file
// does not comply with the PSL format (documented at
// https://github.com/publicsuffix/list/wiki/Format), or with PSL
// submission guidelines
// (https://github.com/publicsuffix/list/wiki/Guidelines). A File with
// errors should not be used to calculate public suffixes for FQDNs.
func Parse(bs []byte) (*List, []error) {
	lines, errs := normalizeToUTF8Lines(bs)
	p := &parser{
		input:     lines,
		inputLine: 0,
	}
	for _, err := range errs {
		p.addError(err)
	}
	ret := p.parseTopLevel()
	return ret, p.errs
}

// parser is the state for a single PSL file parse.
type parser struct {
	// input is the remaining unparsed and untokenized source text.
	input []string
	// inputLine is the offset for input[0]. That is, input[0] is line
	// number inputLine of the source text.
	inputLine int
	// peekBuf is a buffer containing zero or one input tokens.
	peekBuf any
	// errs are the accumulated parse errors so far.
	errs []error
}

// addError records err as a parse/validation error.
//
// If err matches a legacy exemption from current validation rules,
// err is recorded as a non-fatal warning instead.
func (p *parser) addError(err error) {
	p.errs = append(p.errs, err)
}

// The following types and functions are the lexer portion of the
// parsing logic. This is a very simplistic lexer, since
// normalizeToUTF8Lines has already done a lot of heavy lifting to
// clean up the input. Each line of input is converted to a token for
// that line's content. The parser then assembles that stream of
// tokens into multiline blocks, and eventually into a parse tree.

const (
	sectionStartPrefix = "// ===BEGIN "
	sectionEndPrefix   = "// ===END "
	sectionPrefix      = "// ==="
	commentPrefix      = "// "
	wildcardPrefix     = "*."
	exceptionPrefix    = "!"
)

type line struct {
	SourceRange
	Text string
}
type tokenEOF struct{}
type tokenBlank struct{ line }
type tokenComment struct{ line }
type tokenSectionUnknown struct{ line }
type tokenSectionStart struct {
	line
	Name string
}
type tokenSectionEnd struct {
	line
	Name string
}
type tokenSuffix struct{ line }
type tokenWildcard struct {
	line
	Suffix string
}
type tokenException struct {
	line
	Suffix string
}

// next lexes the next token of input and returns it.
func (p *parser) next() (ret any) {
	if p.peekBuf != nil {
		ret := p.peekBuf
		p.peekBuf = nil
		return ret
	}

	if len(p.input) == 0 {
		return tokenEOF{}
	}

	// No matter what, next is going to emit the next line of p.input,
	// the rest of the function is just to determine what kind of
	// token to return.
	src := line{
		SourceRange: SourceRange{p.inputLine, p.inputLine + 1},
		Text:        p.input[0],
	}
	p.input = p.input[1:]
	p.inputLine++

	switch {
	case src.Text == "":
		return tokenBlank{src}

	case strings.HasPrefix(src.Text, sectionStartPrefix):
		// To avoid repeated string processing in different portions
		// of the parser code, the lexer tears apart section markers
		// here to extract the section name.
		name := strings.TrimPrefix(src.Text, sectionStartPrefix)
		name, ok := strings.CutSuffix(name, "===")
		if !ok {
			return tokenSectionUnknown{src}
		}
		return tokenSectionStart{src, name}
	case strings.HasPrefix(src.Text, sectionEndPrefix):
		name := strings.TrimPrefix(src.Text, sectionEndPrefix)
		name, ok := strings.CutSuffix(name, "===")
		if !ok {
			return tokenSectionUnknown{src}
		}
		return tokenSectionEnd{src, name}
	case strings.HasPrefix(src.Text, sectionPrefix):
		return tokenSectionUnknown{src}

	case strings.HasPrefix(src.Text, commentPrefix):
		// Similarly, the following do some light processing of the
		// input so that this doesn't need to be repeated in several
		// portions of the parser.
		src.Text = strings.TrimPrefix(src.Text, "// ")
		return tokenComment{src}
	case strings.HasPrefix(src.Text, wildcardPrefix):
		return tokenWildcard{src, strings.TrimPrefix(src.Text, wildcardPrefix)}
	case strings.HasPrefix(src.Text, exceptionPrefix):
		return tokenException{src, strings.TrimPrefix(src.Text, exceptionPrefix)}

	default:
		return tokenSuffix{src}
	}
}

// peek returns the next token of input, without consuming it.
func (p *parser) peek() any {
	if p.peekBuf == nil {
		p.peekBuf = p.next()
	}
	return p.peekBuf
}

// The rest of this file is the parser itself. It follows the common
// recursive descent structure.

// blockEmitter returns a function that appends blocks to a given
// output list, and also updates an output SourceRange to cover the
// superset of all emitted blocks.
//
// This is a helper to make the functions that parse intermediate AST
// nodes (which have to accumulate a list of children) more readable.
func blockEmitter(out *[]Block, srcRange *SourceRange) func(...Block) {

	return func(bs ...Block) {
		for _, b := range bs {
			if b == nil {
				// Sub-parsers sometimes return nil to indicate the
				// thing they tried to parse was bad and they have
				// nothing to contribute to the output.
				continue
			}

			*out = append(*out, b)

			if srcRange == nil {
				continue
			} else if *srcRange == (SourceRange{}) {
				// Zero value, this is the first emitted block.
				*srcRange = b.SrcRange()
			} else {
				*srcRange = (*srcRange).merge(b.SrcRange())
			}
		}
	}
}

// parseTopLevel parses the top level of a PSL file.
func (p *parser) parseTopLevel() *List {
	ret := &List{}
	emit := blockEmitter(&ret.Blocks, nil)

	for {
		switch tok := p.peek().(type) {
		case tokenEOF:
			return ret
		case tokenBlank:
			p.next()
		case tokenComment:
			emit(p.parseCommentOrSuffixBlock())
		case tokenSectionStart:
			emit(p.parseSection())
		case tokenSectionEnd:
			p.addError(ErrUnstartedSection{tok.SourceRange, tok.Name})
			p.next()
		case tokenSectionUnknown:
			p.addError(ErrUnknownSectionMarker{tok.SourceRange})
			p.next()
		case tokenSuffix, tokenWildcard, tokenException:
			emit(p.parseSuffixBlock(nil))
		default:
			panic("unhandled token")
		}
	}
}

// parseSection parses the contents of a PSL file section.
func (p *parser) parseSection() *Section {
	// Initialize with the start-of-section marker's data.
	start := p.next().(tokenSectionStart)
	ret := &Section{
		blockInfo: blockInfo{
			SourceRange: start.SourceRange,
		},
		Name: start.Name,
	}
	emit := blockEmitter(&ret.Blocks, &ret.SourceRange)

	for {
		switch tok := p.peek().(type) {
		case tokenEOF:
			p.addError(ErrUnclosedSection{ret})
			return ret
		case tokenBlank:
			p.next()
		case tokenComment:
			emit(p.parseCommentOrSuffixBlock())
		case tokenSectionStart:
			// The PSL doesn't allow nested sections, so we pretend
			// like the inner section never existed and grab all its
			// blocks for ourselves. Still record an error for the
			// nested section though.
			inner := p.parseSection()
			emit(inner.Blocks...)
			p.addError(ErrNestedSection{inner.SourceRange, inner.Name, ret})
		case tokenSectionEnd:
			p.next()
			if tok.Name != ret.Name {
				p.addError(ErrMismatchedSection{tok.SourceRange, tok.Name, ret})
			}
			ret.SourceRange.LastLine = tok.SourceRange.LastLine
			return ret
		case tokenSectionUnknown:
			p.next()
			p.addError(ErrUnknownSectionMarker{tok.SourceRange})
		case tokenSuffix, tokenWildcard, tokenException:
			emit(p.parseSuffixBlock(nil))
		default:
			panic("unhandled token")
		}
	}
}

// parseCommentOrSuffixBlock parses a comment, then either returns it
// as a lone comment or chains into suffix block parsing, depending on
// what follows the comment.
//
// This is used to resolve an ambiguity in the PSL format when parsing
// linearly: if we see a comment, that could be a standalone comment,
// or it could be the beginning of a suffix block. In the latter case,
// it's very important to attach the comment to the suffix block,
// since it contains metadata about those suffixes.
func (p *parser) parseCommentOrSuffixBlock() Block {
	comment := p.parseComment()
	switch p.peek().(type) {
	case tokenSuffix, tokenWildcard, tokenException:
		return p.parseSuffixBlock(comment)
	default:
		return comment
	}
}

// parseSuffixBlock parses a suffix block, starting with the provided
// optional initial comment.
func (p *parser) parseSuffixBlock(initialComment *Comment) *Suffixes {
	ret := &Suffixes{
		Info: extractMaintainerInfo(initialComment),
	}
	emit := blockEmitter(&ret.Blocks, &ret.SourceRange)

	if initialComment != nil {
		emit(initialComment)
	}

	for {
		switch tok := p.peek().(type) {
		case tokenBlank:
			return ret
		case tokenComment:
			emit(p.parseComment())
		case tokenSectionUnknown:
			p.next()
			p.addError(ErrUnknownSectionMarker{tok.SourceRange})
		case tokenSectionStart:
			p.next()
			p.addError(ErrSectionInSuffixBlock{tok.SourceRange})
		case tokenSectionEnd:
			p.next()
			p.addError(ErrSectionInSuffixBlock{tok.SourceRange})
		case tokenSuffix:
			emit(p.parseSuffix())
		case tokenWildcard:
			emit(p.parseWildcard())
		case tokenException:
			// Note we don't emit here, exceptions receive a list of
			// existing blocks and attach the exception to the
			// corresponding wildcard entry.
			p.parseException(ret.Blocks)
		case tokenEOF:
			return ret
		default:
			panic("unhandled token")
		}
	}
}

// parseSuffix parses a basic public suffix entry (i.e. not a wildcard
// or an exception.
func (p *parser) parseSuffix() Block {
	tok := p.next().(tokenSuffix)

	domain, err := domain.Parse(tok.Text)
	if err != nil {
		p.addError(ErrInvalidSuffix{tok.SourceRange, tok.Text, err})
		return nil
	}

	return &Suffix{
		blockInfo: blockInfo{
			SourceRange: tok.SourceRange,
		},
		Domain: domain,
	}
}

// parseWildcard parses a public suffix wildcard entry, of the form
// "*.example.com".
func (p *parser) parseWildcard() Block {
	tok := p.next().(tokenWildcard)

	domain, err := domain.Parse(tok.Suffix)
	if err != nil {
		p.addError(ErrInvalidSuffix{tok.SourceRange, tok.Suffix, err})
		return nil
	}

	return &Wildcard{
		blockInfo: blockInfo{
			SourceRange: tok.SourceRange,
		},
		Domain: domain,
	}
}

// parseException parses a public suffix wildcard exception, of the
// form "!foo.example.com". The parsed exception is attached to the
// related Wildcard block in previous. If no such block exists, the
// exception is dropped and a parse error recorded.
func (p *parser) parseException(previous []Block) {
	tok := p.next().(tokenException)

	domain, err := domain.Parse(tok.Suffix)
	if err != nil {
		p.addError(ErrInvalidSuffix{tok.SourceRange, tok.Suffix, err})
		return
	}

	for _, block := range previous {
		w, ok := block.(*Wildcard)
		if !ok {
			continue
		}

		if rest, ok := domain.CutSuffix(w.Domain); ok && len(rest) == 1 {
			w.Exceptions = append(w.Exceptions, domain.Labels()[0])
			return
		}
	}
	p.addError(ErrInvalidSuffix{tok.SourceRange, tok.Suffix, fmt.Errorf("exception %q does not match any wildcard", tok.Suffix)})
}

// parseComment parses a multiline comment block.
func (p *parser) parseComment() *Comment {
	tok := p.next().(tokenComment)
	ret := &Comment{
		blockInfo: blockInfo{
			SourceRange: tok.SourceRange,
		},
		Text: []string{tok.Text},
	}
	for {
		if tok, ok := p.peek().(tokenComment); ok {
			p.next()
			ret.SourceRange = ret.SourceRange.merge(tok.SourceRange)
			ret.Text = append(ret.Text, tok.Text)
		} else {
			return ret
		}
	}
}
