package styles

import "github.com/charmbracelet/lipgloss"

// ── Palette ───────────────────────────────────────────────────────────────────
var (
	ColorAccent  = lipgloss.Color("205") // hot pink — primary accent
	ColorBlue    = lipgloss.Color("111") // blue
	ColorGreen   = lipgloss.Color("114") // green
	ColorAmber   = lipgloss.Color("222") // amber / yellow
	ColorRed     = lipgloss.Color("204") // red
	ColorCyan    = lipgloss.Color("51")  // bright cyan
	ColorMuted   = lipgloss.Color("60")  // muted gray — footer, faint text
	ColorSubtle  = lipgloss.Color("238") // surface subtle — borders, dividers
	ColorDim     = lipgloss.Color("241") // dim — timestamps
	ColorBg      = lipgloss.Color("234") // base background
	ColorBgAlt   = lipgloss.Color("233") // mantle — header, footer bg
	ColorSurface = lipgloss.Color("236") // surface — selected row bg
	ColorText    = lipgloss.Color("252") // main text

	// ── Layout ────────────────────────────────────────────────────────────────

	StyleApp = lipgloss.NewStyle().
			Background(ColorBg).
			Foreground(ColorText)

	StyleHeader = lipgloss.NewStyle().
			Background(ColorBgAlt).
			Padding(1, 3).
			BorderStyle(lipgloss.ThickBorder()).
			BorderBottom(true).
			BorderForeground(ColorAccent)

	StyleHeaderTitle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorAccent)

	StyleHeaderSub = lipgloss.NewStyle().
			Foreground(ColorText).
			Faint(true)

	StyleFooter = lipgloss.NewStyle().
			Foreground(ColorMuted).
			Background(ColorBgAlt).
			Padding(0, 1)

	StyleSummaryBar = lipgloss.NewStyle().
				Foreground(ColorText).
				Background(ColorSurface).
				Padding(0, 2)

	StyleSummaryBarPending = lipgloss.NewStyle().
				Foreground(ColorAmber).
				Background(ColorSurface).
				Bold(true).
				Padding(0, 2)

	// ── Table ─────────────────────────────────────────────────────────────────

	StyleTableHeader = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorBlue).
				Padding(0, 1).
				BorderStyle(lipgloss.NormalBorder()).
				BorderBottom(true).
				BorderForeground(ColorSubtle)

	StyleRowSelected = lipgloss.NewStyle().
				Background(ColorSurface).
				Foreground(ColorText)

	StyleRowNormal = lipgloss.NewStyle().
			Foreground(ColorText)

	StyleRowAlt = lipgloss.NewStyle().
			Foreground(ColorText)

	StyleCell = lipgloss.NewStyle().Padding(0, 1)

	// ── Badges ────────────────────────────────────────────────────────────────

	StyleBadgeDeploy = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorBg).
				Background(ColorAmber).
				Padding(0, 1)

	StyleBadgeClean     = lipgloss.NewStyle().Foreground(ColorGreen)
	StyleBadgeNoRelease = lipgloss.NewStyle().Foreground(ColorCyan)
	StyleBadgeError     = lipgloss.NewStyle().Foreground(ColorRed)
	StyleBadgeLoading   = lipgloss.NewStyle().Foreground(ColorMuted)

	// ── Text ──────────────────────────────────────────────────────────────────

	StyleBold        = lipgloss.NewStyle().Bold(true)
	StyleFaint       = lipgloss.NewStyle().Foreground(ColorMuted)
	StyleSubtle      = lipgloss.NewStyle().Foreground(ColorSubtle)
	StyleRepoName    = lipgloss.NewStyle().Bold(true).Foreground(ColorBlue)
	StyleCommitsAhead = lipgloss.NewStyle().Bold(true).Foreground(ColorAmber)
	StyleBranchName  = lipgloss.NewStyle().Foreground(ColorCyan)
	StyleTagName     = lipgloss.NewStyle().Foreground(ColorAccent)
	StyleNotes       = lipgloss.NewStyle().Foreground(ColorMuted).Italic(true)
	StyleTimestamp   = lipgloss.NewStyle().Foreground(ColorDim)

	// ── Commit expand rows ────────────────────────────────────────────────────

	StyleCommitSHA  = lipgloss.NewStyle().Foreground(lipgloss.Color("74"))  // steel blue
	StyleCommitDate = lipgloss.NewStyle().Foreground(lipgloss.Color("243")) // gray
	StyleCommitMsg  = lipgloss.NewStyle().Foreground(lipgloss.Color("247")) // light gray
	StyleCommitMore = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Italic(true)

	// ── Modal ─────────────────────────────────────────────────────────────────

	StyleModal = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorAccent).
			Background(ColorBgAlt).
			Padding(1, 2)

	StyleModalTitle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorAccent).
				MarginBottom(1)

	StyleModalLabel = lipgloss.NewStyle().Foreground(ColorText)

	StyleInput = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorSubtle).
			Padding(0, 1)

	StyleInputFocused = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorAccent).
				Padding(0, 1)
)

// ── Key Hints ─────────────────────────────────────────────────────────────────

func KeyHint(key, desc string) string {
	k := lipgloss.NewStyle().
		Foreground(ColorBg).
		Background(ColorMuted).
		Padding(0, 1).
		Render(key)
	d := lipgloss.NewStyle().
		Foreground(ColorMuted).
		Render(" " + desc + "  ")
	return k + d
}
