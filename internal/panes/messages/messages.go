package messages

import (
	"errors"
	"protosocat/internal/colors"
	"protosocat/internal/protos"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/dynamicpb"
)

type MessageDisplay interface {
	View() string
}

type DataReceived struct {
	data []byte
}

type ErrorReceived struct {
	err error
}

func (e ErrorReceived) View() string {
	return e.err.Error()
}

type InfoMessage struct {
	text string
}

func (i InfoMessage) View() string {
	return i.text
}

type ReceivedMessage struct {
	Text string
	Type protos.Message
}

func (m ReceivedMessage) View() string {
	return m.Text
}

type OwnMessage struct {
	Text string
	Type protos.Message
}

func (m OwnMessage) View() string {
	return m.Text
}

type MessagePane struct {
	style        lipgloss.Style
	ReceiveProto protos.Message
	SavedSent    chan string
	Receive      chan []byte
	Errors       chan error
	Info         chan string
	Messages     []MessageDisplay
	viewport     viewport.Model
}

func NewMessagePane(
	sendChan chan []byte,
	receiveChan chan []byte,
	errorChan chan error,
	savedSentChan chan string,
	infoChan chan string,
	receiveProto protos.Message,
) MessagePane {
	return MessagePane{
		style: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(colors.BorderColor).
			Padding(1).
			Margin(1),
		ReceiveProto: receiveProto,
		Receive:      receiveChan,
		Errors:       errorChan,
		SavedSent:    savedSentChan,
		Info:         infoChan,
		viewport:     viewport.New(),
	}
}

func (mp *MessagePane) UpdateSize(width int, height int) {
	mp.style = mp.style.Width(width - 2).Height(height - 2)
	verticalOverhead := mp.style.GetVerticalFrameSize()
	horizontalOverhead := mp.style.GetHorizontalFrameSize()
	mp.viewport.SetHeight(height - 2 - verticalOverhead)
	mp.viewport.SetWidth(width - 2 - horizontalOverhead)
}

func ReceiveData(ch chan []byte) tea.Cmd {
	return func() tea.Msg {
		data := <-ch
		return DataReceived{data}
	}
}

func ReceiveError(ch chan error) tea.Cmd {
	return func() tea.Msg {
		err := <-ch
		return ErrorReceived{err}
	}
}

func ReceiveSent(ch chan string) tea.Cmd {
	return func() tea.Msg {
		msg := <-ch
		return OwnMessage{
			Text: msg,
		}
	}
}

func ReceiveInfo(ch chan string) tea.Cmd {
	return func() tea.Msg {
		info := <-ch
		return InfoMessage{
			text: info,
		}
	}
}

func ParseData(data []byte, protobuf protos.Message) tea.Cmd {
	return func() tea.Msg {
		msg := dynamicpb.NewMessage(protobuf.Descriptor)
		err := proto.Unmarshal(data, msg)
		if err != nil {
			return ErrorReceived{
				err: errors.New("could not parse message"),
			}
		}
		opts := protojson.MarshalOptions{
			Indent:          "  ",
			UseProtoNames:   true,
			EmitUnpopulated: true,
		}
		text, err := opts.Marshal(msg)
		if err != nil {
			return ErrorReceived{err}
		}
		return ReceivedMessage{
			Text: string(text),
			Type: protobuf,
		}
	}
}

func (mp MessagePane) Init() tea.Cmd {
	return tea.Batch(
		ReceiveData(mp.Receive),
		ReceiveError(mp.Errors),
		ReceiveSent(mp.SavedSent),
		ReceiveInfo(mp.Info),
	)
}

func (mp MessagePane) Update(msg tea.Msg) (MessagePane, tea.Cmd) {
	switch msg := msg.(type) {
	case DataReceived:
		return mp, ParseData(msg.data, mp.ReceiveProto)
	case ErrorReceived:
		mp.Messages = append(mp.Messages, msg)
		return mp, ReceiveError(mp.Errors)
	case ReceivedMessage:
		mp.Messages = append(mp.Messages, msg)
		return mp, ReceiveData(mp.Receive)
	case OwnMessage:
		mp.Messages = append(mp.Messages, msg)
		return mp, ReceiveSent(mp.SavedSent)
	case InfoMessage:
		mp.Messages = append(mp.Messages, msg)
		return mp, ReceiveInfo(mp.Info)
	}
	return mp, nil
}

func (mp MessagePane) View() string {
	messageWidth := mp.viewport.Width() * 2 / 3
	remWidth := mp.viewport.Width() - messageWidth
	gap := lipgloss.NewStyle().
		Width(remWidth).
		Render("")
	halfgap := lipgloss.NewStyle().
		Width(remWidth / 2).
		Render("")
	strs := make([]string, len(mp.Messages))
	for i, msg := range mp.Messages {
		switch msg := msg.(type) {
		case ErrorReceived:
			strs[i] = lipgloss.JoinHorizontal(
				lipgloss.Center,
				halfgap,
				lipgloss.NewStyle().
					Width(messageWidth).
					AlignHorizontal(lipgloss.Center).
					Border(lipgloss.NormalBorder()).
					BorderForeground(colors.BorderColor).
					Foreground(colors.ErrorColor).
					Render(msg.View()),
				halfgap,
			)
		case ReceivedMessage:
			strs[i] = lipgloss.JoinHorizontal(
				lipgloss.Center,
				gap,
				lipgloss.NewStyle().
					Width(messageWidth).
					Border(lipgloss.NormalBorder()).
					BorderForeground(colors.BorderColor).
					AlignHorizontal(lipgloss.Left).
					Render(msg.View()),
			)
		case OwnMessage:
			strs[i] = lipgloss.JoinHorizontal(
				lipgloss.Center,
				lipgloss.NewStyle().
					Width(messageWidth).
					Border(lipgloss.NormalBorder()).
					BorderForeground(colors.BorderColor).
					AlignHorizontal(lipgloss.Left).
					Render(msg.View()),
				gap,
			)
		case InfoMessage:
			strs[i] = lipgloss.JoinHorizontal(
				lipgloss.Center,
				halfgap,
				lipgloss.NewStyle().
					Width(messageWidth).
					Border(lipgloss.NormalBorder()).
					BorderForeground(colors.BorderColor).
					AlignHorizontal(lipgloss.Center).
					Render(msg.View()),
				halfgap,
			)
		}
	}
	mp.viewport.SetContent(lipgloss.JoinVertical(lipgloss.Top, strs...))
	return mp.style.Render(mp.viewport.View())
}
