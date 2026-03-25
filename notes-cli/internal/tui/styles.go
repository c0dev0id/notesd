package tui

import "github.com/charmbracelet/lipgloss"

var (
	colorPrimary  = lipgloss.Color("12")  // bright blue
	colorMuted    = lipgloss.Color("8")   // dark grey
	colorAccent   = lipgloss.Color("10")  // bright green
	colorWarn     = lipgloss.Color("11")  // yellow
	colorErr      = lipgloss.Color("9")   // red
	colorSelected = lipgloss.Color("4")   // blue

	styleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary)

	styleSubtle = lipgloss.NewStyle().
			Foreground(colorMuted)

	styleSelected = lipgloss.NewStyle().
			Foreground(lipgloss.Color("0")).
			Background(colorSelected).
			Bold(true)

	styleStatusBar = lipgloss.NewStyle().
			Foreground(colorMuted)

	styleStatusBarKey = lipgloss.NewStyle().
				Foreground(colorAccent)

	styleErr = lipgloss.NewStyle().
			Foreground(colorErr)

	styleBorder = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(colorMuted).
			Padding(0, 1)

	styleHelp = lipgloss.NewStyle().
			Foreground(colorMuted).
			Italic(true)
)
