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
	Author  string    // GitHub login of the commit author (may be empty)
}

// RepoStatus holds the computed deploy status for a repo.
type RepoStatus struct {
	Owner        string
	Repo         string
	Branch       string
	TagName      string       // latest tag or release name
	RefType      string       // "release" or "tag"
	RefDate      time.Time    // when the release was published / tag commit date
	CommitsAhead int          // commits on main since last tag/release
	Commits      []CommitInfo // commits on default branch ahead of ref, newest first
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
	refSHA, refName, refType, refDate, err := c.getLatestRef(ctx, owner, repo)
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
	result.RefDate = refDate

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

func (c *Client) getLatestRef(ctx context.Context, owner, repo string) (sha, name, refType string, date time.Time, err error) {
	// Try release first
	var release struct {
		TagName     string    `json:"tag_name"`
		PublishedAt time.Time `json:"published_at"`
		CreatedAt   time.Time `json:"created_at"`
	}
	if e := c.get(ctx, fmt.Sprintf("/repos/%s/%s/releases/latest", owner, repo), &release); e == nil && release.TagName != "" {
		s, e2 := c.resolveTagSHA(ctx, owner, repo, release.TagName)
		if e2 == nil {
			d := release.PublishedAt
			if d.IsZero() {
				d = release.CreatedAt
			}
			return s, release.TagName, "release", d, nil
		}
	}

	// Fall back to latest tag
	var tags []struct {
		Name   string `json:"name"`
		Commit struct {
			SHA string `json:"sha"`
		} `json:"commit"`
	}
	if e := c.get(ctx, fmt.Sprintf("/repos/%s/%s/tags?per_page=1", owner, repo), &tags); e != nil {
		return "", "", "", time.Time{}, e
	}
	if len(tags) == 0 {
		return "", "", "", time.Time{}, nil // no tags at all
	}

	tag := tags[0]
	sha = tag.Commit.SHA
	// Resolve annotated tags
	if resolved, e := c.resolveTagSHA(ctx, owner, repo, tag.Name); e == nil {
		sha = resolved
	}

	// Fetch commit date for tag
	date = c.commitDate(ctx, owner, repo, sha)
	return sha, tag.Name, "tag", date, nil
}

func (c *Client) commitDate(ctx context.Context, owner, repo, sha string) time.Time {
	var commit struct {
		Commit struct {
			Author struct {
				Date time.Time `json:"date"`
			} `json:"author"`
			Committer struct {
				Date time.Time `json:"date"`
			} `json:"committer"`
		} `json:"commit"`
	}
	if err := c.get(ctx, fmt.Sprintf("/repos/%s/%s/commits/%s", owner, repo, sha), &commit); err != nil {
		return time.Time{}
	}
	if !commit.Commit.Committer.Date.IsZero() {
		return commit.Commit.Committer.Date
	}
	return commit.Commit.Author.Date
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
			Author *struct {
				Login string `json:"login"`
			} `json:"author"`
		} `json:"commits"`
	}
	path := fmt.Sprintf("/repos/%s/%s/compare/%s...%s", owner, repo, base, head)
	if err := c.get(ctx, path, &cmp); err != nil {
		return 0, nil, err
	}

	// Return all commits (GitHub compare API caps at 250).
	// API returns oldest first; reverse so newest is first.
	all := cmp.Commits
	commits := make([]CommitInfo, len(all))
	for i, c := range all {
		sha := c.SHA
		if len(sha) > 7 {
			sha = sha[:7]
		}
		msg := c.Commit.Message
		if idx := strings.Index(msg, "\n"); idx != -1 {
			msg = msg[:idx]
		}
		login := ""
		if c.Author != nil {
			login = c.Author.Login
		}
		commits[len(all)-1-i] = CommitInfo{SHA: sha, Message: msg, Date: c.Commit.Author.Date, Author: login}
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
