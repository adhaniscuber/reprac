package ui

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/adhaniscuber/reprac/internal/config"
	"github.com/adhaniscuber/reprac/internal/github"
	"github.com/adhaniscuber/reprac/internal/ui/components"
	"github.com/adhaniscuber/reprac/internal/ui/styles"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ── Messages ──────────────────────────────────────────────────────────────────

type repoCheckedMsg struct {
	key    string
	result github.RepoStatus
}

type repoLoadingMsg struct {
	key string
}

// ── Model ─────────────────────────────────────────────────────────────────────

type Model struct {
	cfg       *config.Config
	cfgPath   string
	gh        *github.Client
	spinner   spinner.Model
	results   map[string]*github.RepoStatus
	loading   map[string]bool
	expanded  map[string]bool
	cursor    int
	width     int
	height    int
	showModal bool
	modal     components.AddRepoModal
	statusMsg string
	noAuth    bool
}

func New(cfgPath string, cfg *config.Config, gh *github.Client) Model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = styles.Faint

	return Model{
		cfg:      cfg,
		cfgPath:  cfgPath,
		gh:       gh,
		spinner:  sp,
		results:  make(map[string]*github.RepoStatus),
		loading:  make(map[string]bool),
		expanded: make(map[string]bool),
		noAuth:   !gh.HasAuth(),
	}
}

func repoKey(owner, repo string) string {
	return owner + "/" + repo
}

// ── Init ──────────────────────────────────────────────────────────────────────

func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{m.spinner.Tick}
	for _, r := range m.cfg.Repos {
		cmds = append(cmds, m.checkRepo(r.Owner, r.Repo))
	}
	return tea.Batch(cmds...)
}

// ── Update ────────────────────────────────────────────────────────────────────

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// If modal is open, route keys to it first
	if m.showModal {
		switch msg := msg.(type) {
		case components.ModalSubmitMsg:
			return m.handleAddRepo(msg.Result)
		case components.ModalCancelMsg:
			m.showModal = false
			m.modal = components.AddRepoModal{}
			return m, nil
		default:
			var cmd tea.Cmd
			m.modal, cmd = m.modal.Update(msg)
			return m, cmd
		}
	}

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case repoLoadingMsg:
		m.loading[msg.key] = true
		return m, nil

	case repoCheckedMsg:
		delete(m.loading, msg.key)
		result := msg.result
		m.results[msg.key] = &result
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	repos := m.cfg.Repos

	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}

	case "down", "j":
		if m.cursor < len(repos)-1 {
			m.cursor++
		}

	case "g":
		m.cursor = 0

	case "G":
		m.cursor = len(repos) - 1

	case "r":
		// Refresh all
		cmds := []tea.Cmd{}
		for _, r := range repos {
			cmds = append(cmds, m.checkRepo(r.Owner, r.Repo))
		}
		m.statusMsg = "Refreshing all..."
		return m, tea.Batch(cmds...)

	case "R":
		// Refresh selected row
		if len(repos) > 0 && m.cursor < len(repos) {
			r := repos[m.cursor]
			m.statusMsg = fmt.Sprintf("Refreshing %s/%s...", r.Owner, r.Repo)
			return m, m.checkRepo(r.Owner, r.Repo)
		}

	case "enter", " ":
		if len(repos) > 0 && m.cursor < len(repos) {
			key := repoKey(repos[m.cursor].Owner, repos[m.cursor].Repo)
			m.expanded[key] = !m.expanded[key]
		}

	case "E":
		for _, r := range repos {
			m.expanded[repoKey(r.Owner, r.Repo)] = true
		}

	case "C":
		m.expanded = make(map[string]bool)

	case "a":
		m.showModal = true
		m.modal = components.NewAddRepoModal(m.width, m.height)
		return m, nil

	case "d":
		if len(repos) > 0 && m.cursor < len(repos) {
			r := repos[m.cursor]
			key := repoKey(r.Owner, r.Repo)
			m.cfg.Repos = append(repos[:m.cursor], repos[m.cursor+1:]...)
			delete(m.results, key)
			delete(m.loading, key)
			if m.cursor >= len(m.cfg.Repos) && m.cursor > 0 {
				m.cursor--
			}
			_ = config.Save(m.cfgPath, m.cfg)
			m.statusMsg = fmt.Sprintf("Removed %s/%s", r.Owner, r.Repo)
		}

	case "o":
		if len(repos) > 0 && m.cursor < len(repos) {
			r := repos[m.cursor]
			url := fmt.Sprintf("https://github.com/%s/%s", r.Owner, r.Repo)
			_ = openBrowser(url)
		}

	case "?":
		// Toggle help via statusMsg
		m.statusMsg = "enter/space=expand  E=expand all  C=collapse all  r=refresh all  R=refresh row  a=add  d=delete  o=browser  j/k=move  q=quit"
	}

	return m, nil
}

func (m Model) handleAddRepo(res components.AddRepoResult) (tea.Model, tea.Cmd) {
	m.showModal = false
	m.modal = components.AddRepoModal{}

	// Check for duplicate
	key := repoKey(res.Owner, res.Repo)
	for _, r := range m.cfg.Repos {
		if repoKey(r.Owner, r.Repo) == key {
			m.statusMsg = fmt.Sprintf("Repo %s already tracked", key)
			return m, nil
		}
	}

	m.cfg.Repos = append(m.cfg.Repos, config.RepoConfig{
		Owner: res.Owner,
		Repo:  res.Repo,
		Notes: res.Notes,
	})
	_ = config.Save(m.cfgPath, m.cfg)
	m.statusMsg = fmt.Sprintf("Added %s", key)
	return m, m.checkRepo(res.Owner, res.Repo)
}

// ── Async check ───────────────────────────────────────────────────────────────

func (m Model) checkRepo(owner, repo string) tea.Cmd {
	key := repoKey(owner, repo)
	m.loading[key] = true
	return func() tea.Msg {
		// First emit loading state
		_ = key
		result := m.gh.CheckRepo(context.Background(), owner, repo)
		return repoCheckedMsg{key: key, result: result}
	}
}

// ── View ──────────────────────────────────────────────────────────────────────

func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	if m.showModal {
		return m.modal.View()
	}

	const leftWidth = 52

	// ── Left panel: ASCII art + tagline ───────────────────────────────────
	asciiArt := styles.HeaderTitle.Render(
		"    ________  ____  _________  _____\n" +
			"   / ___/ _ \\/ __ \\/ ___/ __ `/ ___/\n" +
			"  / /  /  __/ /_/ / /  / /_/ / /__  \n" +
			" /_/   \\___/ ____/_/   \\__,_/\\___/  \n" +
			"          /_/",
	)
	leftContent := "\n" + asciiArt + "\n\n" +
		styles.HeaderSub.Render("  track unreleased changes") + "\n"
	leftPanel := components.RenderTitledPanel("", leftContent, leftWidth, 9, styles.ColorPrimary)

	// ── Right panel: repo overview ─────────────────────────────────────────
	rightWidth := m.width - leftWidth
	total := len(m.cfg.Repos)
	pending, clean, noRelease := 0, 0, 0
	loading := len(m.loading)
	for _, r := range m.cfg.Repos {
		key := repoKey(r.Owner, r.Repo)
		if res, ok := m.results[key]; ok {
			switch res.Status {
			case github.StatusBehind:
				pending++
			case github.StatusClean:
				clean++
			case github.StatusNoRelease:
				noRelease++
			}
		}
	}
	rightContent := buildOverview(total, pending, clean, noRelease, loading, m.noAuth)
	rightPanel := components.RenderTitledPanel("overview", rightContent, rightWidth, 9, styles.ColorSubtle)

	topRow := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)
	topRowHeight := lipgloss.Height(topRow)

	// ── Table panel ───────────────────────────────────────────────────────
	footerHeight := 1
	statusHeight := 1
	tablePanelHeight := m.height - topRowHeight - footerHeight - statusHeight
	if tablePanelHeight < 4 {
		tablePanelHeight = 4
	}

	tableInner := m.width - 2 // panel left+right border
	headerStr := components.RenderHeader(tableInner)
	headerHeight := lipgloss.Height(headerStr)
	dataHeight := tablePanelHeight - 2 - headerHeight // -2 for panel top+bottom border
	if dataHeight < 1 {
		dataHeight = 1
	}

	repos := m.cfg.Repos
	start, end := scrollWindow(m.cursor, repos, m.expanded, m.results, dataHeight)

	var rows []string
	usedHeight := 0
	for i := start; i < end && i < len(repos); i++ {
		r := repos[i]
		key := repoKey(r.Owner, r.Repo)
		res := m.results[key]
		isLoading := m.loading[key]
		isExpanded := m.expanded[key]

		row := components.RenderRow(i, i == m.cursor, key, r.Owner, r.Repo, r.Notes, res, isLoading, isExpanded, tableInner)
		rows = append(rows, row)
		usedHeight += rowHeight(key, isExpanded, m.results)
		if usedHeight >= dataHeight {
			break
		}
	}
	for usedHeight < dataHeight {
		rows = append(rows, strings.Repeat(" ", tableInner))
		usedHeight++
	}

	tableContent := headerStr + "\n" + strings.Join(rows, "\n")
	tablePanel := components.RenderTitledPanel("repositories", tableContent, m.width, 0, styles.ColorSubtle)

	// ── Status bar ────────────────────────────────────────────────────────
	var statusBar string
	if m.statusMsg != "" {
		statusBar = styles.Faint.Width(m.width).Render("  " + m.statusMsg)
	} else {
		statusBar = styles.Faint.Width(m.width).Render("")
	}

	// ── Footer ────────────────────────────────────────────────────────────
	footer := components.RenderFooter(m.width, false)

	return strings.Join([]string{topRow, tablePanel, statusBar, footer}, "\n")
}

func buildOverview(total, pending, clean, noRelease, loading int, noAuth bool) string {
	var lines []string
	lines = append(lines, "")
	if loading > 0 {
		lines = append(lines, styles.Faint.Render(fmt.Sprintf("  ⏳  checking %d...", loading)))
	}
	if pending > 0 {
		lines = append(lines, styles.CommitsAhead.Render(fmt.Sprintf("  ▲  %d  need deploy", pending)))
	}
	if clean > 0 {
		lines = append(lines, styles.BadgeClean.Render(fmt.Sprintf("  ✓  %d  up to date", clean)))
	}
	if noRelease > 0 {
		lines = append(lines, styles.BadgeNoRelease.Render(fmt.Sprintf("  ◈  %d  no release", noRelease)))
	}
	lines = append(lines, styles.Faint.Render(fmt.Sprintf("  ·  %d  repos", total)))
	if noAuth {
		lines = append(lines, "")
		lines = append(lines, styles.Faint.Render("  ⚠  no auth · set GITHUB_TOKEN"))
	}
	return strings.Join(lines, "\n")
}

// rowHeight returns how many terminal lines a repo row occupies.
func rowHeight(key string, expanded bool, results map[string]*github.RepoStatus) int {
	if !expanded {
		return 1
	}
	res := results[key]
	if res == nil || res.Status != github.StatusBehind || len(res.Commits) == 0 {
		return 1
	}
	h := 1 + len(res.Commits) // header + commit lines
	if res.CommitsAhead > len(res.Commits) {
		h++ // "+N more commits" line
	}
	return h
}

func scrollWindow(cursor int, repos []config.RepoConfig, expanded map[string]bool, results map[string]*github.RepoStatus, height int) (start, end int) {
	total := len(repos)
	if total == 0 {
		return 0, 0
	}

	// Try to center around cursor
	// First pass: find a start such that cursor is roughly in the middle
	start = cursor - height/2
	if start < 0 {
		start = 0
	}

	// Forward pass from start: accumulate height until we fill tableHeight
	used := 0
	end = start
	for end < total && used < height {
		key := repoKey(repos[end].Owner, repos[end].Repo)
		used += rowHeight(key, expanded[key], results)
		end++
	}

	// If cursor is beyond end, shift start forward
	if cursor >= end {
		start = cursor
		used = 0
		end = start
		for end < total && used < height {
			key := repoKey(repos[end].Owner, repos[end].Repo)
			used += rowHeight(key, expanded[key], results)
			end++
		}
	}

	return start, end
}

func openBrowser(url string) error {
	var cmd string
	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
	case "windows":
		cmd = "start"
	default:
		cmd = "xdg-open"
	}
	return exec.Command(cmd, url).Start()
}
