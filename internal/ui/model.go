package ui

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/adhaniscuber/reprac/internal/config"
	"github.com/adhaniscuber/reprac/internal/github"
	"github.com/adhaniscuber/reprac/internal/ui/components"
	"github.com/adhaniscuber/reprac/internal/ui/styles"
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
	cfg        *config.Config
	cfgPath    string
	gh         *github.Client
	spinner    spinner.Model
	results    map[string]*github.RepoStatus
	loading    map[string]bool
	cursor     int
	width      int
	height     int
	showModal  bool
	modal      components.AddRepoModal
	statusMsg  string
	noAuth     bool
}

func New(cfgPath string, cfg *config.Config, gh *github.Client) Model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = styles.Faint

	return Model{
		cfg:     cfg,
		cfgPath: cfgPath,
		gh:      gh,
		spinner: sp,
		results: make(map[string]*github.RepoStatus),
		loading: make(map[string]bool),
		noAuth:  !gh.HasAuth(),
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
		m.statusMsg = "r=refresh all  R=refresh row  a=add  d=delete  o=browser  j/k=move  q=quit"
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

	var sections []string

	// Header
	title := styles.Header.Width(m.width).Render(
		"  🚀  reprac  " + styles.Faint.Render("— track unreleased changes"),
	)
	sections = append(sections, title)

	// Summary bar
	total := len(m.cfg.Repos)
	pending := 0
	loading := len(m.loading)
	for _, r := range m.cfg.Repos {
		key := repoKey(r.Owner, r.Repo)
		if res, ok := m.results[key]; ok && res.Status == github.StatusBehind {
			pending++
		}
	}
	sections = append(sections, components.RenderSummary(total, pending, loading, m.width))

	// Auth warning
	if m.noAuth {
		warn := lipgloss.NewStyle().
			Foreground(styles.ColorYellow).
			Background(lipgloss.Color("#2a2500")).
			Padding(0, 2).
			Width(m.width).
			Render("⚠  No GitHub auth detected. Set GITHUB_TOKEN or run: gh auth login  (rate limit: 60 req/hr)")
		sections = append(sections, warn)
	}

	// Table header
	sections = append(sections, components.RenderHeader(m.width))

	// Table rows
	tableHeight := m.height - lipgloss.Height(strings.Join(sections, "\n")) - 4 // -4 for footer+statusbar
	if tableHeight < 1 {
		tableHeight = 1
	}

	repos := m.cfg.Repos
	// Determine scroll window
	start, end := scrollWindow(m.cursor, len(repos), tableHeight)

	var rows []string
	for i := start; i < end && i < len(repos); i++ {
		r := repos[i]
		key := repoKey(r.Owner, r.Repo)
		res := m.results[key]
		isLoading := m.loading[key]

		row := components.RenderRow(i, i == m.cursor, key, r.Owner, r.Repo, r.Notes, res, isLoading)
		rows = append(rows, row)
	}

	// Fill empty rows
	for len(rows) < tableHeight {
		rows = append(rows, lipgloss.NewStyle().Width(m.width).Render(""))
	}

	sections = append(sections, strings.Join(rows, "\n"))

	// Status message bar
	var statusBar string
	if m.statusMsg != "" {
		statusBar = styles.Faint.Width(m.width).Render("  " + m.statusMsg)
	} else {
		statusBar = styles.Faint.Width(m.width).Render("")
	}
	sections = append(sections, statusBar)

	// Footer
	sections = append(sections, components.RenderFooter(m.width, false))

	return strings.Join(sections, "\n")
}

func scrollWindow(cursor, total, height int) (start, end int) {
	if total <= height {
		return 0, total
	}
	start = cursor - height/2
	if start < 0 {
		start = 0
	}
	end = start + height
	if end > total {
		end = total
		start = end - height
		if start < 0 {
			start = 0
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
