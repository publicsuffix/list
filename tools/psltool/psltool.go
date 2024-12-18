// psltool is a CLI tool to manipulate and validate PSL files.
package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode"

	"github.com/creachadair/command"
	"github.com/creachadair/flax"
	"github.com/creachadair/mds/mdiff"
	"github.com/natefinch/atomic"
	"github.com/publicsuffix/list/tools/internal/githistory"
	"github.com/publicsuffix/list/tools/internal/github"
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
				Usage: "<path or git commit hash>",
				Help: `Check that a file is a valid PSL file.

Validation includes basic issues like parse errors, as well as
conformance with the PSL project's style rules and policies.

The argument can be either a local file, or a git commit hash to fetch
from https://github.com/publicsuffix/list.`,
				SetFlags: command.Flags(flax.MustBind, &validateArgs),
				Run:      command.Adapt(runValidate),
			},
			{
				Name:  "check-pr",
				Usage: "<number>",
				Help: `Validate an open PR on GitHub.

Validation includes basic issues like parse errors, as well as
conformance with the PSL project's style rules and policies.`,
				SetFlags: command.Flags(flax.MustBind, &checkPRArgs),
				Run:      command.Adapt(runCheckPR),
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
	Owner  string `flag:"gh-owner,default=publicsuffix,Owner of the github repository to check"`
	Repo   string `flag:"gh-repo,default=list,Github repository to check"`
	Clone  string `flag:"gh-local-clone,Path to a local clone of the repository specified by gh-owner/gh-repo"`
	Online bool   `flag:"online-checks,Run validations that require querying third-party servers"`
}

func isHex(s string) bool {
	for _, r := range s {
		if !unicode.In(r, unicode.ASCII_Hex_Digit) {
			return false
		}
	}
	return true
}

func runValidate(env *command.Env, pathOrHash string) error {
	var bs []byte
	var err error

	client := github.Repo{
		Owner: checkPRArgs.Owner,
		Repo:  checkPRArgs.Repo,
	}

	isPath := false
	if _, err = os.Stat(pathOrHash); err == nil {
		// input is a local file
		isPath = true
		bs, err = os.ReadFile(pathOrHash)
	} else if isHex(pathOrHash) {
		// input looks like a git hash
		bs, err = client.PSLForHash(context.Background(), pathOrHash)
	} else {
		return fmt.Errorf("Failed to read PSL file %q, not a local file or a git commit hash", pathOrHash)
	}
	if err != nil {
		return fmt.Errorf("Failed to read PSL file %q: %w", pathOrHash, err)
	}

	psl, errs := parser.Parse(bs)
	errs = append(errs, psl.Clean()...)
	errs = append(errs, parser.ValidateOffline(psl)...)
	if validateArgs.Online {
		if validateArgs.Clone == "" && isPath {
			// Assume the PSL file being validated might be in a git
			// clone, and try to use that as the reference for history.
			validateArgs.Clone = filepath.Dir(pathOrHash)
		}
		if validateArgs.Clone == "" {
			return errors.New("--gh-local-clone is required for full validation")
		}
		prHistory, err := githistory.GetPRInfo(validateArgs.Clone)
		if err != nil {
			return fmt.Errorf("failed to get local PR history, refusing to run full validation to avoid Github DoS: %w", err)
		}

		ctx, cancel := context.WithTimeout(env.Context(), 1200*time.Second)
		defer cancel()
		errs = append(errs, parser.ValidateOnline(ctx, psl, &client, prHistory)...)
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

var checkPRArgs struct {
	Owner  string `flag:"gh-owner,default=publicsuffix,Owner of the github repository to check"`
	Repo   string `flag:"gh-repo,default=list,Github repository to check"`
	Clone  string `flag:"gh-local-clone,Path to a local clone of the repository specified by gh-owner/gh-repo"`
	Online bool   `flag:"online-checks,Run validations that require querying third-party servers"`
}

func runCheckPR(env *command.Env, prStr string) error {
	pr, err := strconv.Atoi(prStr)
	if err != nil {
		return fmt.Errorf("invalid PR number %q: %w", prStr, err)
	}

	client := github.Repo{
		Owner: checkPRArgs.Owner,
		Repo:  checkPRArgs.Repo,
	}
	withoutPR, withPR, err := client.PSLForPullRequest(env.Context(), pr)
	if err != nil {
		return err
	}

	before, _ := parser.Parse(withoutPR)
	after, errs := parser.Parse(withPR)
	after.SetBaseVersion(before, true)
	errs = append(errs, after.Clean()...)
	errs = append(errs, parser.ValidateOffline(after)...)
	if checkPRArgs.Online {
		var prHistory *githistory.History
		if validateArgs.Clone != "" {
			prHistory, err = githistory.GetPRInfo(validateArgs.Clone)
			if err != nil {
				return fmt.Errorf("failed to get local PR history: %w", err)
			}
		}

		ctx, cancel := context.WithTimeout(env.Context(), 300*time.Second)
		defer cancel()
		errs = append(errs, parser.ValidateOnline(ctx, after, &client, prHistory)...)
	}

	clean := after.MarshalPSL()
	if !bytes.Equal(withPR, clean) {
		errs = append(errs, errors.New("file needs reformatting, run 'psltool fmt' to fix"))
	}

	// Print the blocks marked changed, so a human can check that
	// something was actually checked by validations.
	var changed []*parser.Suffixes
	for _, block := range parser.BlocksOfType[*parser.Suffixes](after) {
		if block.Changed() {
			changed = append(changed, block)
		}
	}
	if len(changed) == 0 {
		fmt.Fprintln(env, "No suffix blocks changed. This can happen if only top-level comments have been edited.")
	} else {
		fmt.Fprintln(env, "Checked the following changed suffix blocks:")
		for _, block := range changed {
			fmt.Fprintf(env, "  %q (%s)\n", block.Info.Name, block.LocationString())
		}
	}
	io.WriteString(env, "\n")

	if len(errs) > 0 {
		for _, err := range errs {
			fmt.Fprintln(env, err)
		}
		io.WriteString(env, "\n")
	}

	if l := len(errs); l == 0 {
		fmt.Fprintln(env, "PSL change is valid")
		return nil
	} else if l == 1 {
		return errors.New("change has 1 error")
	} else {
		return fmt.Errorf("change has %d errors", l)
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
