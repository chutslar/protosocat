package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"protosocat/internal/panes"
	"protosocat/internal/panes/messages"
	"protosocat/internal/panes/protodetails"
	"protosocat/internal/panes/protolist"
	"protosocat/internal/protos"
	"protosocat/internal/ws"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	flag "github.com/spf13/pflag"
)

type Model struct {
	protoListPane    protolist.ProtoListPane
	protoDetailsPane protodetails.ProtoDetailsPane
	showDetails      bool
	messagePane      messages.MessagePane
	messagesActive   bool
	width            int
	height           int
}

func NewModel(
	directory *string,
	receiveType string,
	sendChan chan []byte,
	receiveChan chan []byte,
	errorChan chan error,
	infoChan chan string,
) (*Model, error) {
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

	protobufs, err := parser.Parse()
	if err != nil {
		return nil, err
	}

	var receiveProtobuf *protos.Message
	for _, protobuf := range protobufs {
		if string(protobuf.Descriptor.FullName()) == receiveType {
			receiveProtobuf = &protobuf
		}
	}
	if receiveProtobuf == nil {
		return nil, fmt.Errorf("receive-type protobuf not found: %s", receiveType)
	}

	savedSent := make(chan string, 8)

	protoListPane := protolist.NewProtoListPane(protobufs, wd)
	protoDetailsPane := protodetails.NewProtoDetailsPane(sendChan, savedSent)
	messagePane := messages.NewMessagePane(sendChan, receiveChan, errorChan, savedSent, infoChan, *receiveProtobuf)

	protoListPane.SetActive(true)
	protoDetailsPane.SetActive(true)

	return &Model{
		protoListPane:    protoListPane,
		protoDetailsPane: protoDetailsPane,
		showDetails:      false,
		messagePane:      messagePane,
	}, nil
}

func (m Model) Init() tea.Cmd {
	return m.messagePane.Init()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		halfWidth := m.width / 2
		m.protoListPane.UpdateSize(halfWidth, m.height)
		m.protoDetailsPane.UpdateSize(halfWidth, m.height)
		m.messagePane.UpdateSize(halfWidth, m.height)
		m.messagePane.UpdateSize(halfWidth, m.height)
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "tab":
			m.messagesActive = !m.messagesActive
			m.messagePane.SetActive(m.messagesActive)
			m.protoListPane.SetActive(!m.messagesActive)
			m.protoDetailsPane.SetActive(!m.messagesActive)
		}
	case panes.SwitchToListPane:
		m.protoDetailsPane.SetMessage(nil)
		m.showDetails = false
	case panes.SwitchToDetailsPane:
		m.protoDetailsPane.SetMessage(&msg.Message)
		m.showDetails = true
	}

	// Only the current active proto pane gets updates since their
	// only updates are from user input. Messages pane gets updates
	// even if inactive because it should be updated when messages are received.
	var protoCmd tea.Cmd
	var messageCmd tea.Cmd
	if !m.messagesActive {
		if m.showDetails {
			m.protoDetailsPane, protoCmd = m.protoDetailsPane.Update(msg)
		} else {
			m.protoListPane, protoCmd = m.protoListPane.Update(msg)
		}
	}
	m.messagePane, messageCmd = m.messagePane.Update(msg)
	return m, tea.Batch(protoCmd, messageCmd)
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

	var port int
	var connectionURL string
	var receiveType string
	var basePath string
	flag.IntVarP(&port, "listen", "l", -1, "port to listen on")
	flag.StringVarP(&connectionURL, "connect", "c", "", "websocket server connection URL")
	flag.StringVar(&receiveType, "receive-type", "", "full name of the type of messages to be received")
	flag.StringVarP(&basePath, "base-path", "b", "", "path the websocket will be served on")

	flag.Parse()

	if port >= 0 && connectionURL != "" {
		log.Fatalln("error: -l and -c are mutually exclusive")
	}
	if port < 0 && connectionURL == "" {
		log.Fatalln("error: must provide either -l or -c")
	}

	if receiveType == "" {
		log.Fatalln("error: --receive-type must be provided")
	}

	directory := flag.Arg(0)

	sendChan := make(chan []byte, 8)
	receiveChan := make(chan []byte, 8)
	errorChan := make(chan error, 8)
	infoChan := make(chan string, 8)

	m, err := NewModel(
		emptyToNil(directory),
		receiveType,
		sendChan,
		receiveChan,
		errorChan,
		infoChan)
	if err != nil {
		log.Fatalln("Failed to build model", err)
	}

	if port < 0 {
		if !strings.HasPrefix(connectionURL, "ws://") {
			connectionURL = "ws://" + connectionURL
		}

		client := ws.WSClient{
			URL:     connectionURL,
			Send:    sendChan,
			Receive: receiveChan,
			Error:   errorChan,
			Info:    infoChan,
		}
		client.Run()
	} else {
		server := ws.WSServer{
			Port:     port,
			BasePath: basePath,
			Send:     sendChan,
			Receive:  receiveChan,
			Errors:   errorChan,
			Info:     infoChan,
		}
		server.Run()
	}

	p := tea.NewProgram(m)
	if _, err = p.Run(); err != nil {
		log.Fatalln("Failed while running", err)
	}
}

func emptyToNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
