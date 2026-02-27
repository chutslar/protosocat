package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"protosocat/internal/panes"
	"protosocat/internal/panes/messages"
	"protosocat/internal/panes/protodetails"
	"protosocat/internal/panes/protolist"
	"protosocat/internal/protos"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type Model struct {
	protoListPane    protolist.ProtoListPane
	protoDetailsPane protodetails.ProtoDetailsPane
	showDetails      bool
	messagePane      messages.MessagePane
	width            int
	height           int
}

func NewModel(directory *string) (*Model, error) {
	var wd string
	if directory != nil {
		wd = *directory
	} else {
		wd = "."
	}

	parser := protos.NewParser(wd)
	err := filepath.WalkDir(wd, func(path string, d os.DirEntry, err error) error {
		if err == nil && strings.HasSuffix(path, ".proto") {
			shortPath := strings.TrimPrefix(path, wd)
			parser.AddSource(path, shortPath)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	protos, err := parser.Parse()
	if err != nil {
		return nil, err
	}

	protoListPane := protolist.NewProtoListPane(protos, wd)

	messagePane := messages.NewMessagePane()

	return &Model{
		protoListPane:    protoListPane,
		protoDetailsPane: protodetails.NewProtoDetailsPane(),
		showDetails:      false,
		messagePane:      messagePane,
	}, nil
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	log.Printf("Got msg %v\n", msg)
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.protoListPane.UpdateSize(m.width/2, m.height)
		m.protoDetailsPane.UpdateSize(m.width/2, m.height)
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		}
	case panes.SwitchToListPane:
		m.protoDetailsPane.SetMessage(nil)
		m.showDetails = false
	case panes.SwitchToDetailsPane:
		m.protoDetailsPane.SetMessage(&msg.Message)
		m.showDetails = true
	}

	var cmd tea.Cmd
	if m.showDetails {
		m.protoDetailsPane, cmd = m.protoDetailsPane.Update(msg)
	} else {
		m.protoListPane, cmd = m.protoListPane.Update(msg)
	}
	return m, cmd
}

func (m Model) View() tea.View {
	if m.width == 0 {
		return tea.NewView("")
	}

	var protoView string
	if m.showDetails {
		protoView = m.protoDetailsPane.View()
	} else {
		protoView = m.protoListPane.View()
	}

	s := lipgloss.JoinHorizontal(
		lipgloss.Center,
		protoView,
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

	var directory *string
	if len(os.Args) > 1 {
		directory = &os.Args[1]
	} else {
		directory = nil
	}

	m, err := NewModel(directory)
	if err != nil {
		log.Fatalln("Failed to build model", err)
	}

	p := tea.NewProgram(m)
	if _, err = p.Run(); err != nil {
		log.Fatalln("Failed while running", err)
	}
}
