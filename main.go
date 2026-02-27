package main

import (
	"log"
	"protosocat/internal/panes/messages"
	"protosocat/internal/panes/protolist"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

var borderColor = lipgloss.ANSIColor(36)

type Styles struct {
	protoList lipgloss.Style
}

func NewStyles() *Styles {
	return &Styles{}
}

type Model struct {
	protoListPane protolist.ProtoListPane
	messagePane   messages.MessagePane
	width         int
	height        int
	styles        *Styles
}

func NewModel() *Model {
	protoList := protolist.ProtoList{}
	protoList.AddProto(protolist.NewProto("ClientMessage", "cordially.proto"))
	protoListPane := protolist.NewProtoListPane(protoList)

	messagePane := messages.NewMessagePane()

	return &Model{
		protoListPane: protoListPane,
		messagePane:   messagePane,
		styles:        NewStyles(),
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.protoListPane.UpdateSize(m.width/2, m.height)
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.protoListPane, cmd = m.protoListPane.Update(msg)
	return m, cmd
}

func (m Model) View() tea.View {
	if m.width == 0 {
		return tea.NewView("")
	}

	s := lipgloss.JoinHorizontal(
		lipgloss.Center,
		m.protoListPane.View(),
		m.messagePane.View(),
	)
	v := tea.NewView(s)
	v.AltScreen = true
	return v
}

func main() {
	f, err := tea.LogToFile("debug.log", "debug")
	if err != nil {
		log.Fatalln(err)
	}
	defer func() {
		_ = f.Close()
	}()

	m := NewModel()
	p := tea.NewProgram(m)
	if _, err = p.Run(); err != nil {
		log.Fatalln(err)
	}
}
