package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

// CommitInfo holds short info about a single commit.
type CommitInfo struct {
	SHA     string    // 7-char short SHA
	Message string    // first line of commit message
	Date    time.Time // author date
}

// RepoStatus holds the computed deploy status for a repo.
type RepoStatus struct {
	Owner        string
	Repo         string
	Branch       string
	TagName      string       // latest tag or release name
	RefType      string       // "release" or "tag"
	CommitsAhead int          // commits on main since last tag/release
	Commits      []CommitInfo // up to 5 most recent, newest first
	Status       Status
	ErrorMsg     string
	LastChecked  time.Time
}

type Status int

const (
	StatusLoading Status = iota
	StatusClean          // up to date
	StatusBehind         // has unreleased commits
	StatusNoRelease      // no tags/releases yet
	StatusError
)

func (s Status) String() string {
	switch s {
	case StatusLoading:
		return "loading"
	case StatusClean:
		return "clean"
	case StatusBehind:
		return "behind"
	case StatusNoRelease:
		return "no_release"
	case StatusError:
		return "error"
	}
	return "unknown"
}

// Client handles GitHub API requests.
type Client struct {
	token      string
	httpClient *http.Client
}

// New creates a new GitHub client. Tries GITHUB_TOKEN env var first, then gh CLI.
func New() *Client {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		token = os.Getenv("GH_TOKEN")
	}
	if token == "" {
		token = tokenFromGHCLI()
	}
	return &Client{
		token: token,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func tokenFromGHCLI() string {
	out, err := exec.Command("gh", "auth", "token").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func (c *Client) HasAuth() bool {
	return c.token != ""
}

func (c *Client) get(ctx context.Context, path string, v any) error {
	url := "https://api.github.com" + path
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return ErrNotFound
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return json.NewDecoder(resp.Body).Decode(v)
}

var ErrNotFound = fmt.Errorf("not found")

// CheckRepo fetches and computes the deploy status of a repo.
func (c *Client) CheckRepo(ctx context.Context, owner, repo string) RepoStatus {
	result := RepoStatus{
		Owner:       owner,
		Repo:        repo,
		LastChecked: time.Now(),
	}

	// 1. Get default branch
	branch, err := c.getDefaultBranch(ctx, owner, repo)
	if err != nil {
		result.Status = StatusError
		result.ErrorMsg = shortErr(err)
		return result
	}
	result.Branch = branch

	// 2. Try latest release, fall back to latest tag
	refSHA, refName, refType, err := c.getLatestRef(ctx, owner, repo)
	if err != nil {
		result.Status = StatusError
		result.ErrorMsg = shortErr(err)
		return result
	}
	if refSHA == "" {
		result.Status = StatusNoRelease
		return result
	}

	result.TagName = refName
	result.RefType = refType

	// 3. Compare ref..branch
	ahead, commits, err := c.compareCommits(ctx, owner, repo, refSHA, branch)
	if err != nil {
		result.Status = StatusError
		result.ErrorMsg = shortErr(err)
		return result
	}

	result.CommitsAhead = ahead
	result.Commits = commits
	if ahead > 0 {
		result.Status = StatusBehind
	} else {
		result.Status = StatusClean
	}

	return result
}

func (c *Client) getDefaultBranch(ctx context.Context, owner, repo string) (string, error) {
	var r struct {
		DefaultBranch string `json:"default_branch"`
	}
	if err := c.get(ctx, fmt.Sprintf("/repos/%s/%s", owner, repo), &r); err != nil {
		return "", err
	}
	if r.DefaultBranch == "" {
		return "main", nil
	}
	return r.DefaultBranch, nil
}

func (c *Client) getLatestRef(ctx context.Context, owner, repo string) (sha, name, refType string, err error) {
	// Try release first
	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := c.get(ctx, fmt.Sprintf("/repos/%s/%s/releases/latest", owner, repo), &release); err == nil && release.TagName != "" {
		sha, err := c.resolveTagSHA(ctx, owner, repo, release.TagName)
		if err == nil {
			return sha, release.TagName, "release", nil
		}
	}

	// Fall back to latest tag
	var tags []struct {
		Name   string `json:"name"`
		Commit struct {
			SHA string `json:"sha"`
		} `json:"commit"`
	}
	if err := c.get(ctx, fmt.Sprintf("/repos/%s/%s/tags?per_page=1", owner, repo), &tags); err != nil {
		return "", "", "", err
	}
	if len(tags) == 0 {
		return "", "", "", nil // no tags at all
	}

	tag := tags[0]
	sha = tag.Commit.SHA
	// Resolve annotated tags
	if resolved, err := c.resolveTagSHA(ctx, owner, repo, tag.Name); err == nil {
		sha = resolved
	}

	return sha, tag.Name, "tag", nil
}

func (c *Client) resolveTagSHA(ctx context.Context, owner, repo, tag string) (string, error) {
	var ref struct {
		Object struct {
			Type string `json:"type"`
			SHA  string `json:"sha"`
		} `json:"object"`
	}
	if err := c.get(ctx, fmt.Sprintf("/repos/%s/%s/git/ref/tags/%s", owner, repo, tag), &ref); err != nil {
		return "", err
	}

	// Annotated tag: need to dereference
	if ref.Object.Type == "tag" {
		var tagObj struct {
			Object struct {
				SHA string `json:"sha"`
			} `json:"object"`
		}
		if err := c.get(ctx, fmt.Sprintf("/repos/%s/%s/git/tags/%s", owner, repo, ref.Object.SHA), &tagObj); err != nil {
			return ref.Object.SHA, nil
		}
		return tagObj.Object.SHA, nil
	}

	return ref.Object.SHA, nil
}

func (c *Client) compareCommits(ctx context.Context, owner, repo, base, head string) (int, []CommitInfo, error) {
	var cmp struct {
		AheadBy int `json:"ahead_by"`
		Commits []struct {
			SHA    string `json:"sha"`
			Commit struct {
				Message string `json:"message"`
				Author  struct {
					Date time.Time `json:"date"`
				} `json:"author"`
			} `json:"commit"`
		} `json:"commits"`
	}
	path := fmt.Sprintf("/repos/%s/%s/compare/%s...%s", owner, repo, base, head)
	if err := c.get(ctx, path, &cmp); err != nil {
		return 0, nil, err
	}

	// Take up to 5 most recent commits (API returns oldest first, so take from end)
	const maxCommits = 5
	all := cmp.Commits
	start := 0
	if len(all) > maxCommits {
		start = len(all) - maxCommits
	}
	recent := all[start:]

	// Reverse so newest is first
	commits := make([]CommitInfo, len(recent))
	for i, c := range recent {
		sha := c.SHA
		if len(sha) > 7 {
			sha = sha[:7]
		}
		// Only first line of commit message
		msg := c.Commit.Message
		if idx := strings.Index(msg, "\n"); idx != -1 {
			msg = msg[:idx]
		}
		commits[len(recent)-1-i] = CommitInfo{SHA: sha, Message: msg, Date: c.Commit.Author.Date}
	}

	return cmp.AheadBy, commits, nil
}

func shortErr(err error) string {
	s := err.Error()
	if len(s) > 40 {
		return s[:40]
	}
	return s
}
