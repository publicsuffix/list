// psltool is a CLI tool to manipulate and validate PSL files.
package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/creachadair/command"
	"github.com/creachadair/flax"
	"github.com/creachadair/mds/mdiff"
	"github.com/natefinch/atomic"
	"github.com/publicsuffix/list/tools/internal/parser"
)

func main() {
	log.SetFlags(0)

	root := &command.C{
		Name:  filepath.Base(os.Args[0]),
		Usage: "command [flags] ...\nhelp [command]",
		Help:  "A command-line tool to edit and validate PSL files.",
		Commands: []*command.C{
			{
				Name:  "fmt",
				Usage: "<path>",
				Help: `Format a PSL file.

By default, the given file is updated in place.`,
				SetFlags: command.Flags(flax.MustBind, &fmtArgs),
				Run:      command.Adapt(runFmt),
			},
			{
				Name:  "validate",
				Usage: "<path>",
				Help: `Check that a file is a valid PSL file.

Validation includes basic issues like parse errors, as well as
conformance with the PSL project's style rules and policies.`,
				SetFlags: command.Flags(flax.MustBind, &validateArgs),
				Run:      command.Adapt(runValidate),
			},
			{
				Name: "debug",
				Commands: []*command.C{
					{
						Name:     "dump",
						Usage:    "<path>",
						Help:     "Print a debug dump of a PSL file.",
						SetFlags: command.Flags(flax.MustBind, &debugDumpArgs),
						Run:      command.Adapt(runDebugDump),
					},
				},
			},

			command.HelpCommand(nil),
			command.VersionCommand(),
		},
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	env := root.NewEnv(nil).SetContext(ctx).MergeFlags(true)
	command.RunOrFail(env, os.Args[1:])
}

var fmtArgs struct {
	Diff bool `flag:"d,Output a diff of changes instead of rewriting the file"`
}

func runFmt(env *command.Env, path string) error {
	bs, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("Failed to read PSL file: %w", err)
	}

	psl, parseErrs := parser.Parse(bs)
	fmtErrs := psl.Clean()

	for _, err := range parseErrs {
		fmt.Fprintln(env, err)
	}
	for _, err := range fmtErrs {
		fmt.Fprintln(env, err)
	}

	clean := psl.MarshalPSL()
	changed := !bytes.Equal(bs, clean)

	if changed {
		if fmtArgs.Diff {
			lhs, rhs := strings.Split(string(bs), "\n"), strings.Split(string(clean), "\n")
			diff := mdiff.New(lhs, rhs).AddContext(3)
			mdiff.FormatUnified(os.Stdout, diff, &mdiff.FileInfo{
				Left:  "a/" + path,
				Right: "b/" + path,
			})
			return errors.New("File needs reformatting, rerun without -d to fix")
		}
		if len(parseErrs) > 0 {
			return errors.New("Cannot reformat file due to parse errors")
		}
		if err := atomic.WriteFile(path, bytes.NewReader(clean)); err != nil {
			return fmt.Errorf("Failed to reformat: %w", err)
		}
	}

	return nil
}

var validateArgs struct {
	Online bool `flag:"online-checks,Run validations that require querying third-party servers"`
}

func runValidate(env *command.Env, path string) error {
	bs, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("Failed to read PSL file: %w", err)
	}

	psl, errs := parser.Parse(bs)
	errs = append(errs, psl.Clean()...)
	errs = append(errs, parser.ValidateOffline(psl)...)
	if validateArgs.Online {
		// TODO: no online validations implemented yet.
	}

	clean := psl.MarshalPSL()
	if !bytes.Equal(bs, clean) {
		errs = append(errs, errors.New("file needs reformatting, run 'psltool fmt' to fix"))
	}

	for _, err := range errs {
		fmt.Fprintln(env, err)
	}

	if l := len(errs); l == 0 {
		fmt.Fprintln(env, "PSL file is valid")
		return nil
	} else if l == 1 {
		return errors.New("file has 1 error")
	} else {
		return fmt.Errorf("file has %d errors", l)
	}
}

var debugDumpArgs struct {
	Clean  bool   `flag:"c,Clean AST before dumping"`
	Format string `flag:"f,default=ast,Format to dump in, one of 'ast' or 'psl'"`
}

func runDebugDump(env *command.Env, path string) error {
	var dumpFn func(*parser.List) []byte
	switch debugDumpArgs.Format {
	case "ast":
		dumpFn = (*parser.List).MarshalDebug
	case "psl":
		dumpFn = (*parser.List).MarshalPSL
	default:
		return fmt.Errorf("unknown dump format %q", debugDumpArgs.Format)
	}

	bs, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read PSL file: %w", err)
	}

	psl, errs := parser.Parse(bs)

	if debugDumpArgs.Clean {
		errs = append(errs, psl.Clean()...)
	}

	for _, err := range errs {
		fmt.Fprintln(env, err)
	}

	bs = dumpFn(psl)
	os.Stdout.Write(bs)
	return nil
}
