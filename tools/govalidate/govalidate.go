// govalidate is a tool that parses a PSL file and prints parse and
// lint errors if there are any.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"

	"github.com/natefinch/atomic"
	"github.com/publicsuffix/list/tools/internal/parser"
)

func main() {
	debugPrintTree := flag.Bool("debug-print", false, "print the parse tree for debugging")
	reformat := flag.Bool("reformat", true, "if input is valid, fix formatting errors")

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

	// Errors during the base parse means we may have thrown away
	// information, and that means we can't round-trip the file back
	// to disk without potentially destroying stuff.
	safeToRewrite := len(errs) == 0

	errs = append(errs, psl.Clean()...)
	errs = append(errs, parser.ValidateOffline(psl)...)

	if *debugPrintTree {
		bs := psl.MarshalDebug()
		os.Stdout.Write(bs)
		fmt.Println("")
	}

	// Maybe write out the reformatted file.
	out := psl.MarshalPSL()
	changed := !bytes.Equal(bs, out)
	switch {
	case !safeToRewrite:
		// Can't rewrite without potentially destroying information, do
		// nothing.
	case !changed:
		// No changes needed, don't rewrite so that timestamps etc. don't
		// change.
	case !*reformat:
		// We were ordered to not reformat, and format is wrong.
		errs = append(errs, fmt.Errorf("file has formatting errors, rerun with --reformat=true to fix"))
	default:
		if err := atomic.WriteFile(file, bytes.NewReader(out)); err != nil {
			errs = append(errs, fmt.Errorf("formatting %q: %v", file, err))
		}
	}

	for _, err := range errs {
		fmt.Println(err)
	}
	fmt.Println("")

	if total := len(errs); total > 0 {
		fmt.Printf("File has %d errors.\n", total)
		os.Exit(1)
	} else if changed {
		fmt.Println("File is valid, rewrote to canonical format.")
	} else {
		fmt.Println("File is valid.")
	}
}
