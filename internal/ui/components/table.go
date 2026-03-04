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

func RenderHeader(width int, firstCol int) string {
	var cells []string
	used := 0
	for i := firstCol; i < len(Columns); i++ {
		col := Columns[i]
		cellW := col.Width + 2 // content + left/right padding
		if used+cellW > width && len(cells) > 0 {
			break
		}
		cells = append(cells, styles.StyleTableHeader.Width(col.Width).Render(truncate(col.Title, col.Width-2)))
		used += cellW
	}
	row := lipgloss.JoinHorizontal(lipgloss.Top, cells...)
	return styles.StyleTableHeader.Width(width).Render(row)
}

func RenderRow(
	idx int,
	selected bool,
	repoKey string,
	owner, repo, notes string,
	status *github.RepoStatus,
	loading bool,
	expanded bool,
	termWidth int,
	firstCol int,
) string {
	cells := makeRowCells(owner, repo, notes, status, loading)

	var rendered []string
	used := 0
	for i := firstCol; i < len(cells); i++ {
		cellW := Columns[i].Width + 2
		if used+cellW > termWidth && len(rendered) > 0 {
			break
		}
		rendered = append(rendered, styles.StyleCell.Width(Columns[i].Width).Render(cells[i]))
		used += cellW
	}

	row := lipgloss.JoinHorizontal(lipgloss.Top, rendered...)

	var rowStyle lipgloss.Style
	if selected {
		rowStyle = styles.StyleRowSelected
	} else if idx%2 == 0 {
		rowStyle = styles.StyleRowNormal
	} else {
		rowStyle = styles.StyleRowAlt
	}

	header := rowStyle.Width(termWidth).Render(row)

	// If not expanded or no commit data, return just the header
	if !expanded || status == nil || status.Status != github.StatusBehind || len(status.Commits) == 0 {
		return header
	}

	// Fixed column widths for alignment
	const (
		indent = 4  // leading spaces
		dateW  = 18 // "15:04:05 02-Jan-06"
		shaW   = 7  // short SHA
		colGap = 3  // gap between columns
	)

	shaStyle   := styles.StyleCommitSHA.Width(shaW)
	dateStyle  := styles.StyleCommitDate.Width(dateW)
	msgStyle   := styles.StyleCommitMsg
	moreStyle  := styles.StyleCommitMore
	indentStyle := rowStyle.Copy().Bold(false)

	fixedPrefix := indent + dateW + colGap + shaW + colGap
	maxMsgLen := termWidth - fixedPrefix - 2
	if maxMsgLen < 10 {
		maxMsgLen = 10
	}

	pad := strings.Repeat(" ", indent)
	gap := strings.Repeat(" ", colGap)

	lines := []string{header}
	for _, c := range status.Commits {
		dateStr := ""
		if !c.Date.IsZero() {
			dateStr = c.Date.Local().Format("15:04:05 02-Jan-06")
		}
		line := indentStyle.Width(termWidth).Render(
			pad +
				dateStyle.Render(dateStr) +
				gap +
				shaStyle.Render(c.SHA) +
				gap +
				msgStyle.Render(truncate(c.Message, maxMsgLen)),
		)
		lines = append(lines, line)
	}

	// "+N more commits" if needed
	if status.CommitsAhead > len(status.Commits) {
		more := status.CommitsAhead - len(status.Commits)
		moreLine := indentStyle.Width(termWidth).Render(
			pad + moreStyle.Render(fmt.Sprintf("+ %d more commits...", more)),
		)
		lines = append(lines, moreLine)
	}

	return strings.Join(lines, "\n")
}

func makeRowCells(owner, repo, notes string, s *github.RepoStatus, loading bool) []string {
	if loading || s == nil {
		return []string{
			styles.StyleBadgeLoading.Render("⏳ loading..."),
			styles.StyleRepoName.Render(truncate(owner+"/"+repo, Columns[1].Width-2)),
			styles.StyleFaint.Render("—"),
			styles.StyleFaint.Render("—"),
			styles.StyleFaint.Render("—"),
			styles.StyleNotes.Render(truncate(notes, Columns[5].Width-2)),
			styles.StyleFaint.Render("—"),
		}
	}

	// Status cell
	var statusCell string
	switch s.Status {
	case github.StatusBehind:
		statusCell = styles.StyleBadgeDeploy.Render("▲ need deploy")
	case github.StatusClean:
		statusCell = styles.StyleBadgeClean.Render("✓ up to date")
	case github.StatusNoRelease:
		statusCell = styles.StyleBadgeNoRelease.Render("◈ no release")
	case github.StatusError:
		statusCell = styles.StyleBadgeError.Render("✗ error")
	default:
		statusCell = styles.StyleBadgeLoading.Render("? unknown")
	}

	// Repo cell
	repoCell := styles.StyleRepoName.Render(truncate(owner+"/"+repo, Columns[1].Width-2))

	// Branch
	branch := s.Branch
	if branch == "" {
		branch = "main"
	}
	branchCell := styles.StyleBranchName.Render(truncate(branch, Columns[2].Width-2))

	// Tag/Release
	var tagCell string
	if s.TagName == "" {
		tagCell = styles.StyleFaint.Render("—")
	} else {
		prefix := ""
		if s.RefType == "release" {
			prefix = "⬡ "
		} else {
			prefix = "⬢ "
		}
		tagCell = styles.StyleTagName.Render(truncate(prefix+s.TagName, Columns[3].Width-2))
	}

	// Commits ahead
	var commitsCell string
	switch s.Status {
	case github.StatusBehind:
		commitsCell = styles.StyleCommitsAhead.Render(fmt.Sprintf("+%d commit(s)", s.CommitsAhead))
	case github.StatusClean:
		commitsCell = styles.StyleBadgeClean.Render("0")
	case github.StatusError:
		commitsCell = styles.StyleBadgeError.Render(truncate(s.ErrorMsg, Columns[4].Width-2))
	default:
		commitsCell = styles.StyleFaint.Render("—")
	}

	// Notes
	notesCell := styles.StyleNotes.Render(truncate(notes, Columns[5].Width-2))

	// Last checked
	var checkedCell string
	if s.LastChecked.IsZero() {
		checkedCell = styles.StyleFaint.Render("—")
	} else {
		checkedCell = styles.StyleTimestamp.Render(s.LastChecked.Local().Format("15:04:05"))
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
		parts = append(parts, styles.StyleFaint.Render(fmt.Sprintf("⏳ checking %d...", loading)))
	}

	if pending > 0 {
		parts = append(parts,
			styles.StyleCommitsAhead.Render(fmt.Sprintf("🚀 %d need deploy", pending)),
			styles.StyleBadgeClean.Render(fmt.Sprintf("✓ %d up to date", total-pending-loading)),
		)
	} else if loading == 0 {
		parts = append(parts, styles.StyleBadgeClean.Render("✓ all up to date"))
	}

	parts = append(parts, styles.StyleFaint.Render(fmt.Sprintf("│ %d repos", total)))

	text := strings.Join(parts, "  ")

	if pending > 0 {
		return styles.StyleSummaryBarPending.Width(width).Render(text)
	}
	return styles.StyleSummaryBar.Width(width).Render(text)
}

// ── Footer ────────────────────────────────────────────────────────────────────

func RenderFooter(width int, showModal bool, colOffset int) string {
	var hints []string
	if showModal {
		hints = []string{
			styles.KeyHint("enter", "confirm"),
			styles.KeyHint("tab", "next field"),
			styles.KeyHint("esc", "cancel"),
		}
	} else {
		hints = []string{
			styles.KeyHint("enter", "expand"),
			styles.KeyHint("E/C", "expand/collapse"),
			styles.KeyHint("r", "refresh"),
			styles.KeyHint("←→", "scroll"),
			styles.KeyHint("a", "add"),
			styles.KeyHint("d", "delete"),
			styles.KeyHint("o", "browser"),
			styles.KeyHint("q", "quit"),
		}
	}
	footer := strings.Join(hints, "")

	// Scroll position indicator
	var scrollIndicator string
	if colOffset > 0 {
		colName := Columns[colOffset].Title
		scrollIndicator = styles.StyleFaint.Render(fmt.Sprintf("  col %d/%d (%s)", colOffset+1, len(Columns), colName))
	}

	ts := styles.StyleTimestamp.Render(time.Now().Format("15:04"))
	rightSide := scrollIndicator + "  " + ts
	spacerW := width - lipgloss.Width(footer) - lipgloss.Width(rightSide) - 2
	if spacerW < 0 {
		spacerW = 0
	}
	spacer := lipgloss.NewStyle().Width(spacerW).Render("")
	return styles.StyleFooter.Width(width).Render(footer + spacer + rightSide)
}
