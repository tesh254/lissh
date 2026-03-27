package style

import "github.com/charmbracelet/lipgloss"

var (
	Header  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86"))
	Subtle  = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	OK      = lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Bold(true)
	Warning = lipgloss.NewStyle().Foreground(lipgloss.Color("226"))
	Error   = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	Info    = lipgloss.NewStyle().Foreground(lipgloss.Color("75"))
	Bold    = lipgloss.NewStyle().Bold(true)
	Dim     = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))

	Success  = lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
	Inactive = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	IDStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("75")).Width(5)
)

func StatusColor(inactive bool) lipgloss.Style {
	if inactive {
		return Inactive
	}
	return Success
}

func UserPortColor(user string) lipgloss.Style {
	if user == "" {
		return Subtle
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color("228"))
}
