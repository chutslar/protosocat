package protodetails

import (
	"fmt"
	"protosocat/internal/colors"
	"protosocat/internal/panes"
	"protosocat/internal/protos"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type ProtoDetailsPane struct {
	message *protos.Message
	style   lipgloss.Style
}

func NewProtoDetailsPane() ProtoDetailsPane {
	return ProtoDetailsPane{
		style: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(colors.BorderColor).
			Padding(1).
			Margin(1),
	}
}

func (pd *ProtoDetailsPane) SetMessage(message *protos.Message) {
	pd.message = message
}

func (pd *ProtoDetailsPane) UpdateSize(width int, height int) {
	pd.style = pd.style.Width(width - 2).Height(height - 2)
}

func (pd ProtoDetailsPane) Init() tea.Cmd {
	return nil
}

func (pd ProtoDetailsPane) Update(msg tea.Msg) (ProtoDetailsPane, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "b":
			return pd, panes.SwitchToList()
		}
	}
	return pd, nil
}

func (pd ProtoDetailsPane) View() string {
	return pd.style.Render(fmt.Sprintf("Details for %s", pd.message.Descriptor.FullName()))
}
