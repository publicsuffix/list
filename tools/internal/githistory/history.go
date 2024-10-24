// Package githistory provides helpers to look up PSL PR changes in a
// local git repository.
package githistory

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// PRInfo lists commit metadata for a given Github PR.
type PRInfo struct {
	Num int
	// CommitHash is the git hash in which the PSL contains the
	// changes of this PR.
	CommitHash string
	// ParentHash is the git hash immediately before this PR's changes
	// were added to the PSL.
	ParentHash string
}

// History is PR metadata extracted from a local PSL git clone.
type History struct {
	GitPath string // path to the local git clone
	PRs     map[int]PRInfo
}

// gitTopLevel finds the top level of the git repository that contains
// path, if any.
func gitToplevel(path string) (string, error) {
	bs, err := gitStdout(path, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("finding top level of git repo %q: %w", path, err)
	}
	return string(bs), nil
}

// GetPRInfo extracts PR metadata from the git repository at gitPath.
func GetPRInfo(gitPath string) (*History, error) {
	toplevel, err := gitToplevel(gitPath)
	if err != nil {
		return nil, err
	}

	// List all commits that have a description with a '(#1234)' at
	// the end of a line of description or "Merge pull request #1234
	// from" at the start, and print the matching commits in a form
	// that's easy to parse.
	prCommits, err := gitStdout(toplevel, "log",
		"--perl-regexp",
		`--grep=\(#\d+\)$`,
		`--grep=^Merge pull request #\d+ from`,
		"--pretty=%H@%P@%s",
		"master")

	ret := &History{
		GitPath: toplevel,
		PRs:     map[int]PRInfo{},
	}
	for _, line := range strings.Split(string(prCommits), "\n") {
		fs := strings.SplitN(line, "@", 3)
		if len(fs) != 3 {
			return nil, fmt.Errorf("unexpected line format %q", line)
		}
		commit, parentsStr, desc := fs[0], fs[1], fs[2]
		parents := strings.Split(parentsStr, " ")
		// For merge commits, we have multiple parents, and we want
		// the "main branch" side of the merge, i.e. the state of the
		// tree before the PR was merged. Empirically, Github always
		// lists that commit as the 1st parent in merge commits.
		//
		// For squash commits, there is only one parent.
		//
		// This logic cannot handle rebase-and-merge actions, since
		// those by definition erase the PR history from the git
		// history. However, the PSL doesn't use rebase-and-merge by
		// convention, so this works out. Worst case, if this logic
		// does catch a rebase-and-merge, the result will be false
		// positives (suffix flagged for invalid TXT record), if the
		// PR contained more than 1 commit.
		parent := parents[0]
		ms := prNumberRe.FindStringSubmatch(desc)
		if len(ms) != 3 {
			// The grep on git log returned a false positive where the
			// PR number is not on the first line of the commit
			// message. This is not a commit in the standard github
			// format for PRs.
			continue
		}

		var prNum int
		if ms[1] != "" {
			prNum, err = strconv.Atoi(ms[1])
		} else {
			prNum, err = strconv.Atoi(ms[2])
		}
		if err != nil {
			// Shouldn't happen, the regex isolates digits, why can't
			// we parse digits?
			return nil, fmt.Errorf("unexpected invalid PR number string %q", ms[1])
		}

		ret.PRs[prNum] = PRInfo{
			Num:        prNum,
			CommitHash: commit,
			ParentHash: parent,
		}
	}

	return ret, nil
}

// GetPSL returns the PSL file at the given commit hash in the git
// repository at gitPath.
func GetPSL(gitPath string, hash string) ([]byte, error) {
	toplevel, err := gitToplevel(gitPath)
	if err != nil {
		return nil, err
	}

	bs, err := gitStdout(toplevel, "show", fmt.Sprintf("%s:public_suffix_list.dat", hash))
	if err != nil {
		return nil, err
	}

	return bs, nil
}

// Matches either "(#1234)" at the end of a line, or "Merge pull
// request #1234 from" at the start of a line. The first is how github
// formats squash-and-merge commits, the second is how github formats
// 2-parent merge commits.
var prNumberRe = regexp.MustCompile(`(?:\(#(\d+)\)$)|(?:^Merge pull request #(\d+) from)`)

func gitStdout(repoPath string, args ...string) ([]byte, error) {
	args = append([]string{"-C", repoPath}, args...)
	c := exec.Command("git", args...)
	var stderr bytes.Buffer
	c.Stderr = &stderr
	bs, err := c.Output()
	if err != nil {
		// Make the error show the git commandline and captured
		// stderr, not just the plain "exited with code 45" error.
		cmdline := append([]string{"git"}, args...)
		var stderrStr string
		if stderr.Len() != 0 {
			stderrStr = "stderr:\n" + stderr.String()
		}
		return nil, fmt.Errorf("running %q: %w. %s", strings.Join(cmdline, " "), err, stderrStr)
	}
	return bytes.TrimSpace(bs), nil
}
