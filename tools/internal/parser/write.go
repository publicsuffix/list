package parser

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// MarshalPSL returns the list serialized to standard PSL text format.
func (l *List) MarshalPSL() []byte {
	var ret bytes.Buffer
	writeBlockPSL(&ret, l)
	return ret.Bytes()
}

func writeBlockPSL(w io.Writer, b Block) {
	f := func(msg string, args ...any) {
		fmt.Fprintf(w, msg+"\n", args...)
	}

	switch v := b.(type) {
	case *List:
		for i, child := range v.Blocks {
			if i > 0 {
				f("")
			}
			writeBlockPSL(w, child)
		}
	case *Section:
		f("// ===BEGIN %s===", v.Name)
		for _, child := range v.Blocks {
			f("")
			writeBlockPSL(w, child)
		}
		f("")
		f("// ===END %s===", v.Name)
	case *Suffixes:
		for _, child := range v.Blocks {
			writeBlockPSL(w, child)
		}
	case *Suffix:
		f("%s", v.Domain)
	case *Wildcard:
		base := v.Domain
		f("*.%s", base)
		for _, exc := range v.Exceptions {
			f("!%s.%s", exc, base)
		}
	case *Comment:
		for _, line := range v.Text {
			f("// %s", line)
		}
	default:
		panic("unknown ast node")
	}
}

// MarhsalDebug returns the list serialized to a verbose debugging
// format. This format is private to this package and for development
// use only. The format may change drastically without notice.
func (l *List) MarshalDebug() []byte {
	var ret bytes.Buffer
	writeBlockDebug(&ret, l, "")
	return ret.Bytes()
}

func writeBlockDebug(w io.Writer, b Block, indent string) {
	changemark := ""
	if b.Changed() {
		changemark = "!!"
	}
	f := func(msg string, args ...any) {
		fmt.Fprintf(w, indent+msg+"\n", args...)
	}

	src := b.SrcRange()
	loc := fmt.Sprintf("%d-%d", src.FirstLine, src.LastLine)
	if src.FirstLine+1 == src.LastLine {
		loc = strconv.Itoa(src.FirstLine)
	}

	const extraIndent = "   "
	nextIndent := indent + extraIndent

	switch v := b.(type) {
	case *List:
		f("%sList(%s) {", changemark, loc)
		for _, child := range v.Blocks {
			writeBlockDebug(w, child, nextIndent)
		}
		f("} // List")
	case *Section:
		f("%sSection(%s, name=%q) {", changemark, loc, v.Name)
		for _, child := range v.Blocks {
			writeBlockDebug(w, child, nextIndent)
		}
		f("} // Section(name=%q)", v.Name)
	case *Suffixes:
		items := []string{loc, fmt.Sprintf("editable=%v", v.Info.MachineEditable)}
		if v.Info.Name != "" {
			items = append(items, fmt.Sprintf("name=%q", v.Info.Name))
		}
		for _, u := range v.Info.URLs {
			items = append(items, fmt.Sprintf("url=%q", u))
		}
		for _, e := range v.Info.Maintainers {
			email := strings.TrimSpace(fmt.Sprintf("%s <%s>", e.Name, e.Address))
			items = append(items, fmt.Sprintf("contact=%q", email))
		}
		for _, o := range v.Info.Other {
			items = append(items, fmt.Sprintf("other=%q", o))
		}

		const open = "SuffixBlock("
		pad := strings.Repeat(" ", len(open))
		f("%s%s%s) {", changemark, open, strings.Join(items, fmt.Sprintf(",\n%s%s", indent, pad)))
		for _, child := range v.Blocks {
			writeBlockDebug(w, child, nextIndent)
		}
		f("} // SuffixBlock(name=%q)", v.Info.Name)
	case *Suffix:
		f("%sSuffix(%s, %q)", changemark, loc, v.Domain)
	case *Wildcard:
		w := fmt.Sprintf("*.%s", v.Domain)
		if len(v.Exceptions) > 0 {
			f("%sWildcard(%s, %q, except=%v)", changemark, loc, w, v.Exceptions)
		} else {
			f("%sWildcard(%s, %q)", changemark, loc, w)
		}
	case *Comment:
		f("%sComment(%s) {", changemark, loc)
		for _, line := range v.Text {
			f("%s%s", extraIndent, line)
		}
		f("}")
	default:
		panic("unknown ast node")
	}
}
