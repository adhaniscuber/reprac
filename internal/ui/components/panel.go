package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// RenderTitledPanel renders a bordered panel.
// Title is right-aligned on the top border (empty = no title label).
// innerHeight = minimum inner content lines, 0 = auto height.
// titleColor (optional) overrides the border color for the title label text.
func RenderTitledPanel(title, content string, width, innerHeight int, borderColor lipgloss.Color, titleColor ...lipgloss.Color) string {
	if width < 4 {
		width = 4
	}
	bStyle := lipgloss.NewStyle().Foreground(borderColor)

	tColor := borderColor
	if len(titleColor) > 0 {
		tColor = titleColor[0]
	}
	tStyle := lipgloss.NewStyle().Foreground(tColor)

	inner := width - 2 // subtract left + right border chars

	lines := strings.Split(content, "\n")

	// Pad content to minimum height
	for innerHeight > 0 && len(lines) < innerHeight {
		lines = append(lines, "")
	}

	// Top border: ╭──────────── Title ─╮  (title right-aligned)
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
		top = bStyle.Render("╭"+strings.Repeat("─", dashes)) + tStyle.Render(t) + bStyle.Render("─╮")
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
