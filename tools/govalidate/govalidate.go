// govalidate is a tool that parses a PSL file and prints parse and
// lint errors if there are any.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/publicsuffix/list/tools/internal/parser"
)

func main() {
	warnings := flag.Bool("with-warnings", false, "also print errors that were downgraded to warnings")
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
		fmt.Fprintf(os.Stderr, "Reading PSL file: %v", err)
		os.Exit(1)
	}

	psl := parser.Parse(string(bs))

	for _, err := range psl.Errors {
		fmt.Println(err)
	}
	if *warnings {
		for _, err := range psl.Warnings {
			fmt.Println(err, "(ignored)")
		}
	}
	if len(psl.Errors) > 0 {
		os.Exit(1)
	} else {
		fmt.Printf("%q seems to be a valid PSL file.\n", file)
	}
}
