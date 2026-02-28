package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// RenderTitledPanel renders a Surge-style bordered panel.
// Title is right-aligned on the top border (empty = no title label).
// innerHeight = minimum inner content lines, 0 = auto height.
func RenderTitledPanel(title, content string, width, innerHeight int, borderColor lipgloss.Color) string {
	bStyle := lipgloss.NewStyle().Foreground(borderColor)
	inner := width - 2 // subtract left + right border chars

	lines := strings.Split(content, "\n")

	// Pad content to minimum height
	for innerHeight > 0 && len(lines) < innerHeight {
		lines = append(lines, "")
	}

	// Top border: ╭──────────── Title ─╮  (title right-aligned, Surge style)
	var top string
	if title == "" {
		top = bStyle.Render("╭" + strings.Repeat("─", inner) + "╮")
	} else {
		t := " " + title + " "
		tw := lipgloss.Width(t)
		dashes := inner - tw - 1
		if dashes < 1 {
			dashes = 1
		}
		top = bStyle.Render("╭" + strings.Repeat("─", dashes) + t + "─╮")
	}

	// Content lines: │ content <padding> │
	var mid []string
	for _, l := range lines {
		lw := lipgloss.Width(l)
		rpad := ""
		if lw < inner {
			rpad = strings.Repeat(" ", inner-lw)
		}
		mid = append(mid, bStyle.Render("│")+l+rpad+bStyle.Render("│"))
	}

	// Bottom border
	bot := bStyle.Render("╰" + strings.Repeat("─", inner) + "╯")

	all := append([]string{top}, mid...)
	all = append(all, bot)
	return strings.Join(all, "\n")
}
