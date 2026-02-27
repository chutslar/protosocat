package messages

import (
	"protosocat/internal/colors"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type MessagePane struct {
	style lipgloss.Style
}

func NewMessagePane() MessagePane {
	return MessagePane{
		style: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(colors.BorderColor).
			Padding(1).
			Margin(1),
	}
}

func (mp MessagePane) Init() tea.Cmd {
	return nil
}

func (mp MessagePane) Update(_ tea.Msg) (MessagePane, tea.Cmd) {
	return mp, nil
}

func (mp MessagePane) View() string {
	return "Messages go here"
}
