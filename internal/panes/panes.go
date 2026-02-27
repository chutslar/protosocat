package panes

import (
	"protosocat/internal/protos"

	tea "charm.land/bubbletea/v2"
)

type PaneType int

const (
	ProtoListPaneType      PaneType = 1
	ProtoDetailsPaneType   PaneType = 2
	MessageHistoryPaneType PaneType = 3
)

type SwitchToDetailsPane struct {
	Message protos.Message
}

type SwitchToListPane struct{}

func SwitchToDetails(message protos.Message) tea.Cmd {
	return func() tea.Msg {
		return SwitchToDetailsPane{
			Message: message,
		}
	}
}

func SwitchToList() tea.Cmd {
	return func() tea.Msg {
		return SwitchToListPane{}
	}
}
