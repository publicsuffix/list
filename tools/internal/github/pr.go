// Package github provides a github client with functions tailored to
// the PSL's needs.
package github

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/google/go-github/v63/github"
)

// Client is a GitHub API client that performs PSL-specific
// operations. The zero value is a client that interacts with the
// official publicsuffix/list repository.
type Client struct {
	// Owner is the github account of the repository to query. If
	// empty, defaults to "publicsuffix".
	Owner string
	// Repo is the repository to query. If empty, defaults to "list".
	Repo string

	client *github.Client
}

func (c *Client) owner() string {
	if c.Owner != "" {
		return c.Owner
	}
	return "publicsuffix"
}

func (c *Client) repo() string {
	if c.Repo != "" {
		return c.Repo
	}
	return "list"
}

func (c *Client) apiClient() *github.Client {
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
func (c *Client) PSLForPullRequest(ctx context.Context, prNum int) (withoutPR, withPR []byte, err error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	pr, _, err := c.apiClient().PullRequests.Get(ctx, c.owner(), c.repo(), prNum)
	if err != nil {
		return nil, nil, err
	}

	mergeCommit := pr.GetMergeCommitSHA()
	if mergeCommit == "" {
		return nil, nil, fmt.Errorf("no merge commit available for PR %d", prNum)
	}
	commitInfo, _, err := c.apiClient().Git.GetCommit(ctx, c.owner(), c.repo(), mergeCommit)
	if err != nil {
		return nil, nil, fmt.Errorf("getting info for trial merge SHA %q: %w", mergeCommit, err)
	}

	var beforeMergeCommit string
	if pr.GetMerged() && len(commitInfo.Parents) == 1 {
		// PR was merged, PSL policy is to use squash-and-merge, so
		// the pre-PR commit is simply the parent of the merge commit.
		beforeMergeCommit = commitInfo.Parents[0].GetSHA()
	} else if !pr.GetMergeable() {
		// PR isn't merged, and there's a merge conflict that prevents
		// us from knowing what the pre- and post-merge states are.
		return nil, nil, fmt.Errorf("cannot get PSL for PR %d, needs rebase", prNum)
	} else {
		// PR is open, which means the merge commit is a "trial merge"
		// that shows what would happen if you merged the PR. The
		// trial merge commit has 2 parents, one is the PR head commit
		// and the other is master without the PR's changes.
		if numParents := len(commitInfo.Parents); numParents != 2 {
			return nil, nil, fmt.Errorf("unexpected parent count %d for trial merge commit on PR %d, expected 2 parents", numParents, prNum)
		}

		prHeadCommit := pr.GetHead().GetSHA()
		if prHeadCommit == "" {
			return nil, nil, fmt.Errorf("no commit SHA available for head of PR %d", prNum)
		}
		if commitInfo.Parents[0].GetSHA() == prHeadCommit {
			beforeMergeCommit = commitInfo.Parents[1].GetSHA()
		} else {
			beforeMergeCommit = commitInfo.Parents[0].GetSHA()
		}
	}

	withoutPR, err = c.PSLForHash(ctx, beforeMergeCommit)
	if err != nil {
		return nil, nil, err
	}
	withPR, err = c.PSLForHash(ctx, mergeCommit)
	if err != nil {
		return nil, nil, err
	}
	return withoutPR, withPR, nil
}

// PSLForHash returns the PSL file at the given git commit hash.
func (c *Client) PSLForHash(ctx context.Context, hash string) ([]byte, error) {
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
