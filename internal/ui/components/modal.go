package components

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/adhaniscuber/reprac/internal/ui/styles"
)

// AddRepoResult is returned when the modal is submitted.
type AddRepoResult struct {
	Owner string
	Repo  string
	Notes string
}

// AddRepoModal is a Bubble Tea model for the "add repo" overlay.
type AddRepoModal struct {
	inputs  []textinput.Model
	focused int
	width   int
	height  int
}

const (
	fieldOwner = 0
	fieldRepo  = 1
	fieldNotes = 2
)

func NewAddRepoModal(width, height int) AddRepoModal {
	labels := []string{"owner", "repo", "notes (optional)"}
	placeholders := []string{"e.g. your-org", "e.g. your-app", "e.g. Production API"}

	inputs := make([]textinput.Model, 3)
	for i := range inputs {
		t := textinput.New()
		t.Placeholder = placeholders[i]
		t.CharLimit = 100
		t.Prompt = "  "
		if i == 0 {
			t.Focus()
		}
		_ = labels[i]
		inputs[i] = t
	}

	return AddRepoModal{
		inputs:  inputs,
		focused: 0,
		width:   width,
		height:  height,
	}
}

type ModalSubmitMsg struct{ Result AddRepoResult }
type ModalCancelMsg struct{}

func (m AddRepoModal) Update(msg tea.Msg) (AddRepoModal, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg { return ModalCancelMsg{} }

		case "tab", "down":
			m.inputs[m.focused].Blur()
			m.focused = (m.focused + 1) % len(m.inputs)
			m.inputs[m.focused].Focus()

		case "shift+tab", "up":
			m.inputs[m.focused].Blur()
			m.focused = (m.focused - 1 + len(m.inputs)) % len(m.inputs)
			m.inputs[m.focused].Focus()

		case "enter":
			owner := strings.TrimSpace(m.inputs[fieldOwner].Value())
			repo := strings.TrimSpace(m.inputs[fieldRepo].Value())
			notes := strings.TrimSpace(m.inputs[fieldNotes].Value())
			if owner != "" && repo != "" {
				return m, func() tea.Msg {
					return ModalSubmitMsg{Result: AddRepoResult{
						Owner: owner, Repo: repo, Notes: notes,
					}}
				}
			}
			// Highlight empty required fields
			if owner == "" {
				m.inputs[m.focused].Blur()
				m.focused = fieldOwner
				m.inputs[m.focused].Focus()
			} else if repo == "" {
				m.inputs[m.focused].Blur()
				m.focused = fieldRepo
				m.inputs[m.focused].Focus()
			}
		}
	}

	var cmds []tea.Cmd
	for i := range m.inputs {
		var cmd tea.Cmd
		m.inputs[i], cmd = m.inputs[i].Update(msg)
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}

func (m AddRepoModal) View() string {
	labels := []string{"Owner", "Repo", "Notes (optional)"}

	var sb strings.Builder
	sb.WriteString(styles.ModalTitle.Render("➕  Add Repository"))
	sb.WriteString("\n\n")

	for i, inp := range m.inputs {
		label := styles.ModalLabel.Render(labels[i])
		sb.WriteString(label + "\n")

		var inputBox string
		if m.focused == i {
			inputBox = styles.InputFocused.Width(38).Render(inp.View())
		} else {
			inputBox = styles.InputStyle.Width(38).Render(inp.View())
		}
		sb.WriteString(inputBox + "\n\n")
	}

	enterHint := styles.KeyHint("enter", "confirm")
	tabHint := styles.KeyHint("tab", "next")
	escHint := styles.KeyHint("esc", "cancel")
	hints := lipgloss.JoinHorizontal(lipgloss.Top, enterHint, tabHint, escHint)
	sb.WriteString(hints)

	dialog := styles.Modal.Render(sb.String())

	// Centre the dialog
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, dialog,
		lipgloss.WithWhitespaceForeground(styles.ColorMuted),
	)
}
