// Package github provides a github client with functions tailored to
// the PSL's needs.
package github

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/google/go-github/v63/github"
)

// Client is a GitHub API client that performs PSL-specific
// operations. The zero value is a client that interacts with the
// official publicsuffix/list repository.
type Repo struct {
	// Owner is the github account of the repository to query. If
	// empty, defaults to "publicsuffix".
	Owner string
	// Repo is the repository to query. If empty, defaults to "list".
	Repo string

	client *github.Client
}

func (c *Repo) owner() string {
	if c.Owner != "" {
		return c.Owner
	}
	return "publicsuffix"
}

func (c *Repo) repo() string {
	if c.Repo != "" {
		return c.Repo
	}
	return "list"
}

func (c *Repo) apiClient() *github.Client {
	if c.client == nil {
		c.client = github.NewClient(nil)
		if token := os.Getenv("GITHUB_TOKEN"); token != "" {
			c.client = c.client.WithAuthToken(token)
		}
	}
	return c.client
}

// PSLForPullRequest fetches the PSL files needed to validate the
// given pull request. Returns the PSL file for the target branch, and
// the same but with the PR's changes applied.
func (c *Repo) PSLForPullRequest(ctx context.Context, prNum int) (withoutPR, withPR []byte, err error) {
	// Github sometimes needs a little time to think to update the PR
	// state, so we might need to sleep and retry a few times. Usually
	// the status updates in <5s, but just for safety, give it a more
	// generous timeout.
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var withoutHash, withHash string
	for withoutHash == "" {
		withoutHash, withHash, err = c.getPRCommitInfo(ctx, prNum)
		if errors.Is(err, errMergeInfoNotReady) {
			// PR exists but merge info is stale, need to wait and
			// retry.
			select {
			case <-time.After(2 * time.Second):
				continue
			case <-ctx.Done():
				return nil, nil, ctx.Err()
			}
		} else if err != nil {
			return nil, nil, err
		}
	}

	withoutPR, err = c.PSLForHash(ctx, withoutHash)
	if err != nil {
		return nil, nil, err
	}
	withPR, err = c.PSLForHash(ctx, withHash)
	if err != nil {
		return nil, nil, err
	}
	return withoutPR, withPR, nil
}

var errMergeInfoNotReady = errors.New("PR mergeability information not available yet, please retry later")

// getPRCommitInfo returns the "before" and "after" commit hashes for
// prNum.
//
// The exact meaning of "before" and "after" varies, but in general
// before is the state of the master branch right before the PR is
// merged, and "after" is the same state plus the PR's changes, with
// no unrelated changes.
//
// For an unmerged PR, "after" is a "trial merge commit" created
// automatically by Github to run CI and check that the PR is
// mergeable, and "before" is the master branch state from that trial
// merge - usually the latest current state.
//
// For a merged PR, "after" is the commit where the PR's changes first
// appeared in master, and "before" is the state of master immediately
// before that.
//
// getPRCommitInfo returns the sentinel error errMergeInfoNotReady if
// an open PR exists, but github needs a bit more time to update the
// trial merge commit. The caller is expected to retry with
// appropriate backoff.
func (c *Repo) getPRCommitInfo(ctx context.Context, prNum int) (withoutPRCommit, withPRCommit string, err error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	pr, _, err := c.apiClient().PullRequests.Get(ctx, c.owner(), c.repo(), prNum)
	if err != nil {
		return "", "", err
	}

	mergeCommit := pr.GetMergeCommitSHA()
	if mergeCommit == "" {
		return "", "", fmt.Errorf("no merge commit available for PR %d", prNum)
	}
	commitInfo, _, err := c.apiClient().Git.GetCommit(ctx, c.owner(), c.repo(), mergeCommit)
	if err != nil {
		return "", "", fmt.Errorf("getting info for trial merge SHA %q: %w", mergeCommit, err)
	}

	var beforeMergeCommit string
	if pr.GetMerged() && len(commitInfo.Parents) == 1 {
		// PR was merged, PSL policy is to use squash-and-merge, so
		// the pre-PR commit is simply the parent of the merge commit.
		beforeMergeCommit = commitInfo.Parents[0].GetSHA()
	} else if pr.Mergeable == nil {
		// PR isn't merged, but github needs time to rebase the PR and
		// create a trial merge. Unfortunately the only way to know
		// when it's done is to just poll and wait for the mergeable
		// bool to be valid.
		return "", "", errMergeInfoNotReady
	} else if !pr.GetMergeable() {
		// PR isn't merged, and there's a merge conflict that prevents
		// us from knowing what the pre- and post-merge states are.
		return "", "", fmt.Errorf("cannot get PSL for PR %d, needs rebase to resolve conflicts", prNum)
	} else {
		// PR is either open, or it was merged without squashing. In
		// both cases, mergeCommit has 2 parents: one is the PR head
		// commit, and the other is the master branch without the PR's
		// changes.
		if numParents := len(commitInfo.Parents); numParents != 2 {
			return "", "", fmt.Errorf("unexpected parent count %d for trial merge commit on PR %d, expected 2 parents", numParents, prNum)
		}

		prHeadCommit := pr.GetHead().GetSHA()
		if prHeadCommit == "" {
			return "", "", fmt.Errorf("no commit SHA available for head of PR %d", prNum)
		}
		if commitInfo.Parents[0].GetSHA() == prHeadCommit {
			beforeMergeCommit = commitInfo.Parents[1].GetSHA()
		} else {
			beforeMergeCommit = commitInfo.Parents[0].GetSHA()
		}
	}

	return beforeMergeCommit, mergeCommit, nil
}

// PSLForHash returns the PSL file at the given git commit hash.
func (c *Repo) PSLForHash(ctx context.Context, hash string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	opts := &github.RepositoryContentGetOptions{
		Ref: hash,
	}
	content, _, _, err := c.apiClient().Repositories.GetContents(ctx, c.owner(), c.repo(), "public_suffix_list.dat", opts)
	if err != nil {
		return nil, fmt.Errorf("getting PSL for commit %q: %w", hash, err)
	}
	ret, err := content.GetContent()
	if err != nil {
		return nil, err
	}
	return []byte(ret), nil
}
