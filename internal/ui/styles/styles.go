package styles

import "github.com/charmbracelet/lipgloss"

// Palette
var (
	ColorPrimary   = lipgloss.Color("#CBA6F7") // mauve
	ColorSecondary = lipgloss.Color("#89B4FA") // blue
	ColorGreen     = lipgloss.Color("#A6E3A1") // green
	ColorYellow    = lipgloss.Color("#F9E2AF") // yellow
	ColorRed       = lipgloss.Color("#F38BA8") // red
	ColorCyan      = lipgloss.Color("#89DCEB") // sky
	ColorGray      = lipgloss.Color("#6C7086") // overlay0
	ColorSubtle    = lipgloss.Color("#45475A") // surface1
	ColorBg        = lipgloss.Color("#1E1E2E") // base
	ColorBgAlt     = lipgloss.Color("#181825") // mantle
	ColorSurface   = lipgloss.Color("#313244") // surface0
	ColorText      = lipgloss.Color("#CDD6F4") // text
	ColorMuted     = lipgloss.Color("#585B70") // surface2
)

// ── Layout ────────────────────────────────────────────────────────────────────

var (
	App = lipgloss.NewStyle().
		Background(ColorBg).
		Foreground(ColorText)

	Header = lipgloss.NewStyle().
		Background(ColorBgAlt).
		Padding(1, 3).
		BorderStyle(lipgloss.ThickBorder()).
		BorderBottom(true).
		BorderForeground(ColorPrimary)

	HeaderTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			Background(ColorBgAlt)

	HeaderSub = lipgloss.NewStyle().
			Foreground(ColorText).
			Background(ColorBgAlt).
			Faint(true)

	Footer = lipgloss.NewStyle().
		Foreground(ColorGray).
		Background(ColorBgAlt).
		Padding(0, 1)

	SummaryBar = lipgloss.NewStyle().
			Foreground(ColorText).
			Background(ColorSurface).
			Padding(0, 2)

	SummaryBarPending = lipgloss.NewStyle().
				Foreground(ColorYellow).
				Background(ColorSurface).
				Bold(true).
				Padding(0, 2)
)

// ── Table ─────────────────────────────────────────────────────────────────────

var (
	TableHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorSecondary).
			Background(ColorBgAlt).
			Padding(0, 1).
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			BorderForeground(ColorSubtle)

	RowSelected = lipgloss.NewStyle().
			Background(ColorSurface).
			Foreground(ColorText)

	RowNormal = lipgloss.NewStyle().
			Foreground(ColorText)

	RowAlt = lipgloss.NewStyle().
		Foreground(ColorText).
		Background(lipgloss.Color("#1a1a2e"))

	Cell = lipgloss.NewStyle().Padding(0, 1)
)

// ── Status Badges ─────────────────────────────────────────────────────────────

var (
	BadgeDeploy = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorBg).
			Background(ColorYellow).
			Padding(0, 1)

	BadgeClean = lipgloss.NewStyle().
			Foreground(ColorGreen)

	BadgeNoRelease = lipgloss.NewStyle().
			Foreground(ColorCyan)

	BadgeError = lipgloss.NewStyle().
			Foreground(ColorRed)

	BadgeLoading = lipgloss.NewStyle().
			Foreground(ColorGray)
)

// ── Text Helpers ──────────────────────────────────────────────────────────────

var (
	Bold = lipgloss.NewStyle().Bold(true)

	Faint = lipgloss.NewStyle().Foreground(ColorGray)

	RepoName = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorSecondary)

	CommitsAhead = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorYellow)

	BranchName = lipgloss.NewStyle().
			Foreground(ColorCyan)

	TagName = lipgloss.NewStyle().
		Foreground(ColorPrimary)

	Notes = lipgloss.NewStyle().
		Foreground(ColorGray).
		Italic(true)

	Timestamp = lipgloss.NewStyle().
			Foreground(ColorMuted)
)

// ── Modal / Overlay ───────────────────────────────────────────────────────────

var (
	Modal = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary).
		Background(ColorBgAlt).
		Padding(1, 2)

	ModalTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			MarginBottom(1)

	ModalLabel = lipgloss.NewStyle().
			Foreground(ColorText)

	InputStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorSubtle).
			Padding(0, 1)

	InputFocused = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			Padding(0, 1)
)

// ── Key Hints ─────────────────────────────────────────────────────────────────

func KeyHint(key, desc string) string {
	k := lipgloss.NewStyle().
		Foreground(ColorBg).
		Background(ColorGray).
		Padding(0, 1).
		Render(key)
	d := lipgloss.NewStyle().
		Foreground(ColorGray).
		Render(" " + desc + "  ")
	return k + d
}
