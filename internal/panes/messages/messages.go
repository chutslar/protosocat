package messages

import (
	"errors"
	"protosocat/internal/colors"
	"protosocat/internal/panes"
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

type MessageRenderer struct {
	gap           string
	halfgap       string
	ownStyle      lipgloss.Style
	receivedStyle lipgloss.Style
	infoStyle     lipgloss.Style
	errorStyle    lipgloss.Style
}

func NewMessageRenderer() MessageRenderer {
	return MessageRenderer{
		ownStyle: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(colors.PrimaryColor).
			AlignHorizontal(lipgloss.Left),
		receivedStyle: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(colors.SecondaryColor).
			AlignHorizontal(lipgloss.Left),
		infoStyle: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(colors.InfoColor).
			AlignHorizontal(lipgloss.Center),
		errorStyle: lipgloss.NewStyle().
			AlignHorizontal(lipgloss.Center).
			Border(lipgloss.NormalBorder()).
			BorderForeground(colors.ErrorBorderColor).
			Foreground(colors.ErrorColor),
	}
}

func (mr *MessageRenderer) SetViewportWidth(viewportWidth int) {
	messageWidth := viewportWidth * 2 / 3
	remWidth := viewportWidth - messageWidth
	mr.gap = lipgloss.NewStyle().
		Width(remWidth).
		Render("")
	mr.halfgap = lipgloss.NewStyle().
		Width(remWidth / 2).
		Render("")
	mr.ownStyle = mr.ownStyle.Width(messageWidth)
	mr.errorStyle = mr.errorStyle.Width(messageWidth)
	mr.infoStyle = mr.infoStyle.Width(messageWidth)
}

func (mr MessageRenderer) ViewOwnMessage(msg string) string {
	return lipgloss.JoinHorizontal(
		lipgloss.Center,
		mr.ownStyle.Render(msg),
		mr.gap,
	)
}

func (mr MessageRenderer) ViewReceivedMessage(msg string) string {
	return lipgloss.JoinHorizontal(
		lipgloss.Center,
		mr.gap,
		mr.receivedStyle.Render(msg),
	)
}

func (mr MessageRenderer) ViewInfoMessage(msg string) string {
	return lipgloss.JoinHorizontal(
		lipgloss.Center,
		mr.halfgap,
		mr.infoStyle.Render(msg),
		mr.halfgap,
	)
}

func (mr MessageRenderer) ViewErrorMessage(msg string) string {
	return lipgloss.JoinHorizontal(
		lipgloss.Center,
		mr.halfgap,
		mr.errorStyle.Render(msg),
		mr.halfgap,
	)
}

type MessagePane struct {
	style           panes.ToggleStyle
	ReceiveProto    protos.Message
	SavedSent       chan string
	Receive         chan []byte
	Errors          chan error
	Info            chan string
	Messages        []MessageDisplay
	viewport        viewport.Model
	messageRenderer MessageRenderer
	IsActiveTab     bool
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
		style: panes.ToggleStyle{
			ActiveStyle: lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(colors.BorderColor).
				Padding(1).
				Margin(1),
			InactiveStyle: lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(colors.InactiveColor).
				Padding(1).
				Margin(1),
		},
		ReceiveProto:    receiveProto,
		Receive:         receiveChan,
		Errors:          errorChan,
		SavedSent:       savedSentChan,
		Info:            infoChan,
		viewport:        viewport.New(),
		messageRenderer: NewMessageRenderer(),
	}
}

func (mp *MessagePane) SetActive(active bool) {
	mp.IsActiveTab = active
}

func (mp *MessagePane) UpdateSize(width int, height int) {
	mp.style = mp.style.Width(width - 2).Height(height - 2)
	verticalOverhead := mp.style.GetVerticalFrameSize()
	horizontalOverhead := mp.style.GetHorizontalFrameSize()
	viewportWidth := width - 2 - horizontalOverhead
	mp.viewport.SetHeight(height - 2 - verticalOverhead)
	mp.viewport.SetWidth(viewportWidth)
	mp.messageRenderer.SetViewportWidth(viewportWidth)
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
	var contentUpdate string
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case DataReceived:
		cmd = ParseData(msg.data, mp.ReceiveProto)
	case ErrorReceived:
		contentUpdate = mp.messageRenderer.ViewErrorMessage(msg.View())
		mp.Messages = append(mp.Messages, msg)
		cmd = ReceiveError(mp.Errors)
	case ReceivedMessage:
		contentUpdate = mp.messageRenderer.ViewReceivedMessage(msg.View())
		mp.Messages = append(mp.Messages, msg)
		cmd = ReceiveData(mp.Receive)
	case OwnMessage:
		contentUpdate = mp.messageRenderer.ViewOwnMessage(msg.View())
		mp.Messages = append(mp.Messages, msg)
		cmd = ReceiveSent(mp.SavedSent)
	case InfoMessage:
		contentUpdate = mp.messageRenderer.ViewInfoMessage(msg.View())
		mp.Messages = append(mp.Messages, msg)
		cmd = ReceiveInfo(mp.Info)
	}

	if contentUpdate != "" {
		newContent := lipgloss.JoinVertical(
			lipgloss.Top,
			mp.viewport.GetContent(),
			contentUpdate,
		)
		mp.viewport.SetContent(newContent)
		mp.viewport.GotoBottom()
	} else if mp.IsActiveTab {
		mp.viewport, cmd = mp.viewport.Update(msg)
	}
	return mp, cmd
}

func (mp MessagePane) View() string {
	return mp.style.GetStyle(mp.IsActiveTab).Render(mp.viewport.View())
}
