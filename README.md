# reprac

Track unreleased commits across your GitHub repos — never forget to deploy again.

Built with [bubbletea](https://github.com/charmbracelet/bubbletea), [lipgloss](https://github.com/charmbracelet/lipgloss), and [cobra](https://github.com/spf13/cobra).

## Install

### Homebrew (macOS / Linux)

```bash
brew install adhaniscuber/tap/reprac
```

### Shell script (macOS / Linux)

```bash
curl -fsSL https://raw.githubusercontent.com/adhaniscuber/reprac/main/install.sh | sh
```

### Go

```bash
go install github.com/adhaniscuber/reprac@latest
```

### Manual

Download the latest binary from [Releases](https://github.com/adhaniscuber/reprac/releases).

## How it works

For each tracked repo, reprac:
1. Fetches the latest **release** (falls back to latest **tag**)
2. Compares that ref against the **default branch** (main/master)
3. Shows how many commits are ahead → those are unreleased changes

## Setup

```bash
# Create sample config
reprac init

# Default config location: ~/.config/reprac/repos.yaml
```

```yaml
# repos.yaml
repos:
  - owner: your-org
    repo: your-app
    notes: "Production frontend"
  - owner: your-org
    repo: your-api
    notes: "Backend API"
```

## Auth

reprac uses the GitHub API. Without a token you're limited to 60 requests/hour. With a token, 5000/hour.

### Generate a token

1. Go to https://github.com/settings/tokens/new
2. Note: `reprac`
3. Expiration: your preference
4. Scopes: check **`repo`** (for private repos) or **`public_repo`** (for public repos only)
5. Click **Generate token** → copy the token

### Set the token (permanent)

Add to `~/.zshrc` (or `~/.bashrc` if you use bash):

```bash
echo 'export GITHUB_TOKEN=ghp_xxxxx' >> ~/.zshrc
source ~/.zshrc
```

Replace `ghp_xxxxx` with your generated token.

### Alternative: gh CLI

If you already use [gh CLI](https://cli.github.com), the token is detected automatically with no extra setup:

```bash
gh auth login
```

## Usage

```bash
reprac                          # default config
reprac --config ~/repos.yaml   # custom config
reprac init                     # create sample config
reprac version
```

## Keyboard shortcuts

| Key | Action |
|---|---|
| `j` / `k` or `↑` / `↓` | Move cursor |
| `r` | Refresh all repos |
| `R` | Refresh selected repo |
| `a` | Add repo (modal form) |
| `d` | Delete selected repo |
| `o` | Open repo in browser |
| `g` / `G` | Jump to top / bottom |
| `?` | Show key hints in status bar |
| `q` | Quit |

## Status indicators

| Icon | Meaning |
|---|---|
| 🚀 `DEPLOY` | Has unreleased commits — needs deploy |
| `✓ up to date` | All commits are tagged/released |
| `◈ no release` | Repo has no tags or releases yet |
| `✗ error` | Failed to fetch (private repo, typo, etc.) |
