// govalidate is a tool that parses a PSL file and prints parse and
// lint errors if there are any.
package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/publicsuffix/list/tools/internal/parser"
)

func main() {
	debugPrintTree := flag.Bool("debug-print", false, "print the parse tree for debugging")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [flags] pslfile\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
	}

	file := flag.Arg(0)

	bs, err := os.ReadFile(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read PSL file: %v", err)
		os.Exit(1)
	}

	psl, errs := parser.Parse(bs)

	if *debugPrintTree {
		debugPrint(psl)
	}

	for _, err := range errs {
		fmt.Println(err)
	}

	verrs := parser.ValidateOffline(psl)
	for _, err := range verrs {
		fmt.Println(err)
	}

	if total := len(errs) + len(verrs); total > 0 {
		fmt.Printf("\nFile has %d errors.\n", total)
		os.Exit(1)
	} else {
		fmt.Println("\nFile is valid.")
	}
}

// debugPrint prints out a PSL syntax tree in a private, subject to
// change text format.
func debugPrint(b parser.Block) {
	debugPrintRec(b, "")
}

func debugPrintRec(b parser.Block, indent string) {
	nextIndent := indent + "    "
	f := func(msg string, args ...any) {
		fmt.Printf(indent+msg+"\n", args...)
	}
	src := b.SrcRange()
	loc := fmt.Sprintf("[%d:%d]", src.FirstLine, src.LastLine)
	if src.FirstLine+1 == src.LastLine {
		loc = strconv.Itoa(src.FirstLine)
	}

	switch v := b.(type) {
	case *parser.List:
		f("List(%s) {", loc)
		for _, b := range v.Blocks {
			debugPrintRec(b, nextIndent)
		}
		f("}")
	case *parser.Comment:
		f("Comment(%s) {", loc)
		for _, t := range v.Text {
			f("    %q,", t)
		}
		f("}")
	case *parser.Section:
		f("Section(%s, %q) {", loc, v.Name)
		for _, b := range v.Blocks {
			debugPrintRec(b, nextIndent)
		}
		f("}")
	case *parser.Suffixes:
		items := []string{loc, fmt.Sprintf("editable=%v", v.Info.MachineEditable)}
		if v.Info.Name != "" {
			items = append(items, fmt.Sprintf("name=%q", v.Info.Name))
		}
		for _, u := range v.Info.URLs {
			items = append(items, fmt.Sprintf("url=%q", u))
		}
		for _, e := range v.Info.Maintainers {
			items = append(items, fmt.Sprintf("contact=%q", e))
		}
		for _, o := range v.Info.Other {
			items = append(items, fmt.Sprintf("other=%q", o))
		}

		f("SuffixBlock(%s) {", strings.Join(items, fmt.Sprintf(",\n%s            ", indent)))
		for _, b := range v.Blocks {
			debugPrintRec(b, nextIndent)
		}
		f("}")
	case *parser.Suffix:
		f("Suffix(%s, %q)", loc, strings.Join(v.Labels, "."))
	case *parser.Wildcard:
		if len(v.Exceptions) > 0 {
			f("Wildcard(%s, %q, except=%v)", loc, strings.Join(v.Labels, "."), v.Exceptions)
		} else {
			f("Wildcard(%s, %q)", loc, strings.Join(v.Labels, "."))
		}
	default:
		panic("unknown block type")
	}
}
