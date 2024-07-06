package parser

import (
	"bytes"
	"fmt"
	"io"
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
