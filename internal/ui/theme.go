package ui

import "github.com/charmbracelet/lipgloss"

// HN orange and depth colors for comment nesting.
var (
	hnOrange = lipgloss.Color("#FF6600")

	// DepthColors cycles through these for nested comment bars.
	DepthColors = []lipgloss.Color{
		"#FF6600", // orange
		"#828282", // gray
		"#00BFFF", // deep sky blue
		"#32CD32", // lime green
		"#FFD700", // gold
		"#FF69B4", // hot pink
		"#9370DB", // medium purple
		"#20B2AA", // light sea green
	}

	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF"))

	URLStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#828282"))

	MetaStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#828282"))

	ScoreStyle = lipgloss.NewStyle().
			Foreground(hnOrange)

	AuthorStyle = lipgloss.NewStyle().
			Foreground(hnOrange).
			Bold(true)

	OPBadgeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#000000")).
			Background(hnOrange).
			Bold(true).
			Padding(0, 1)

	SelectedStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(hnOrange).
			PaddingLeft(1)

	StatusBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#333333")).
			Foreground(lipgloss.Color("#FFFFFF")).
			Padding(0, 1)

	StatusBarActiveTab = lipgloss.NewStyle().
				Background(hnOrange).
				Foreground(lipgloss.Color("#FFFFFF")).
				Bold(true).
				Padding(0, 1)

	StatusBarTab = lipgloss.NewStyle().
			Background(lipgloss.Color("#555555")).
			Foreground(lipgloss.Color("#CCCCCC")).
			Padding(0, 1)

	DimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666"))

	CodeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A0D0A0"))

	NotificationBadge = lipgloss.NewStyle().
				Background(lipgloss.Color("#FF0000")).
				Foreground(lipgloss.Color("#FFFFFF")).
				Bold(true).
				Padding(0, 1)
)
