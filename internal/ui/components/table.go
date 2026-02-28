package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/adhaniscuber/reprac/internal/github"
	"github.com/adhaniscuber/reprac/internal/ui/styles"
	"github.com/charmbracelet/lipgloss"
)

// Column definitions
type Column struct {
	Title string
	Width int
}

var Columns = []Column{
	{Title: "STATUS", Width: 18},
	{Title: "REPOSITORY", Width: 30},
	{Title: "BRANCH", Width: 12},
	{Title: "LAST TAG / RELEASE", Width: 22},
	{Title: "UNRELEASED", Width: 14},
	{Title: "NOTES", Width: 24},
	{Title: "CHECKED", Width: 10},
}

// TableRow represents one rendered row.
type TableRow struct {
	RepoKey string
	Cells   []string
}

func RenderHeader(width int) string {
	cells := make([]string, len(Columns))
	for i, col := range Columns {
		cells[i] = styles.TableHeader.
			Width(col.Width).
			Render(truncate(col.Title, col.Width-2))
	}
	row := lipgloss.JoinHorizontal(lipgloss.Top, cells...)
	return styles.TableHeader.Width(width).Render(row)
}

func RenderRow(
	idx int,
	selected bool,
	repoKey string,
	owner, repo, notes string,
	status *github.RepoStatus,
	loading bool,
) string {
	cells := makeRowCells(owner, repo, notes, status, loading)

	rendered := make([]string, len(cells))
	for i, cell := range cells {
		rendered[i] = styles.Cell.Width(Columns[i].Width).Render(cell)
	}

	row := lipgloss.JoinHorizontal(lipgloss.Top, rendered...)

	if selected {
		return styles.RowSelected.Render(row)
	}
	if idx%2 == 0 {
		return styles.RowNormal.Render(row)
	}
	return styles.RowAlt.Render(row)
}

func makeRowCells(owner, repo, notes string, s *github.RepoStatus, loading bool) []string {
	if loading || s == nil {
		return []string{
			styles.BadgeLoading.Render("⏳ loading..."),
			styles.RepoName.Render(truncate(owner+"/"+repo, Columns[1].Width-2)),
			styles.Faint.Render("—"),
			styles.Faint.Render("—"),
			styles.Faint.Render("—"),
			styles.Notes.Render(truncate(notes, Columns[5].Width-2)),
			styles.Faint.Render("—"),
		}
	}

	// Status cell
	var statusCell string
	switch s.Status {
	case github.StatusBehind:
		statusCell = styles.BadgeDeploy.Render("▲ need deploy")
	case github.StatusClean:
		statusCell = styles.BadgeClean.Render("✓ up to date")
	case github.StatusNoRelease:
		statusCell = styles.BadgeNoRelease.Render("◈ no release")
	case github.StatusError:
		statusCell = styles.BadgeError.Render("✗ error")
	default:
		statusCell = styles.BadgeLoading.Render("? unknown")
	}

	// Repo cell
	repoCell := styles.RepoName.Render(truncate(owner+"/"+repo, Columns[1].Width-2))

	// Branch
	branch := s.Branch
	if branch == "" {
		branch = "main"
	}
	branchCell := styles.BranchName.Render(truncate(branch, Columns[2].Width-2))

	// Tag/Release
	var tagCell string
	if s.TagName == "" {
		tagCell = styles.Faint.Render("—")
	} else {
		prefix := ""
		if s.RefType == "release" {
			prefix = "⬡ "
		} else {
			prefix = "⬢ "
		}
		tagCell = styles.TagName.Render(truncate(prefix+s.TagName, Columns[3].Width-2))
	}

	// Commits ahead
	var commitsCell string
	switch s.Status {
	case github.StatusBehind:
		commitsCell = styles.CommitsAhead.Render(fmt.Sprintf("+%d commit(s)", s.CommitsAhead))
	case github.StatusClean:
		commitsCell = styles.BadgeClean.Render("0")
	case github.StatusError:
		commitsCell = styles.BadgeError.Render(truncate(s.ErrorMsg, Columns[4].Width-2))
	default:
		commitsCell = styles.Faint.Render("—")
	}

	// Notes
	notesCell := styles.Notes.Render(truncate(notes, Columns[5].Width-2))

	// Last checked
	var checkedCell string
	if s.LastChecked.IsZero() {
		checkedCell = styles.Faint.Render("—")
	} else {
		checkedCell = styles.Timestamp.Render(s.LastChecked.Local().Format("15:04:05"))
	}

	return []string{statusCell, repoCell, branchCell, tagCell, commitsCell, notesCell, checkedCell}
}

func TableWidth() int {
	total := 0
	for _, c := range Columns {
		total += c.Width + 2 // padding
	}
	return total
}

func truncate(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-1]) + "…"
}

// ── Summary Bar ───────────────────────────────────────────────────────────────

func RenderSummary(total, pending, loading int, width int) string {
	var parts []string

	if loading > 0 {
		parts = append(parts, styles.Faint.Render(fmt.Sprintf("⏳ checking %d...", loading)))
	}

	if pending > 0 {
		parts = append(parts,
			styles.CommitsAhead.Render(fmt.Sprintf("🚀 %d need deploy", pending)),
			styles.BadgeClean.Render(fmt.Sprintf("✓ %d up to date", total-pending-loading)),
		)
	} else if loading == 0 {
		parts = append(parts, styles.BadgeClean.Render("✓ all up to date"))
	}

	parts = append(parts, styles.Faint.Render(fmt.Sprintf("│ %d repos", total)))

	text := strings.Join(parts, "  ")

	if pending > 0 {
		return styles.SummaryBarPending.Width(width).Render(text)
	}
	return styles.SummaryBar.Width(width).Render(text)
}

// ── Footer ────────────────────────────────────────────────────────────────────

func RenderFooter(width int, showModal bool) string {
	var hints []string
	if showModal {
		hints = []string{
			styles.KeyHint("enter", "confirm"),
			styles.KeyHint("tab", "next field"),
			styles.KeyHint("esc", "cancel"),
		}
	} else {
		hints = []string{
			styles.KeyHint("r", "refresh all"),
			styles.KeyHint("R", "refresh row"),
			styles.KeyHint("a", "add repo"),
			styles.KeyHint("d", "delete"),
			styles.KeyHint("o", "open browser"),
			styles.KeyHint("q", "quit"),
		}
	}
	footer := strings.Join(hints, "")
	ts := styles.Timestamp.Render(time.Now().Format("15:04"))
	spacer := lipgloss.NewStyle().Width(width - lipgloss.Width(footer) - lipgloss.Width(ts) - 4).Render("")
	return styles.Footer.Width(width).Render(footer + spacer + ts)
}
