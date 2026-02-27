package protolist

import (
	"fmt"
	"io"
	"protosocat/internal/colors"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type ItemStyles struct {
	item lipgloss.Style
}

type Proto struct {
	Name string
	File string
}

func NewProto(name string, file string) Proto {
	return Proto{
		Name: name,
		File: file,
	}
}

func (p Proto) FilterValue() string {
	return p.Name
}

func (p Proto) Title() string {
	return p.Name
}

func (p Proto) Description() string {
	return p.File
}

type itemDelegate struct {
	styles ItemStyles
}

func (d itemDelegate) Height() int {
	return 1
}

func (d itemDelegate) Spacing() int {
	return 0
}

func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd {
	return nil
}

func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(Proto)
	if !ok {
		return
	}

	desc := lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(250)).
		Render(fmt.Sprintf("(%s)", i.Description()))

	str := fmt.Sprintf("%s %s", i.Name, desc)

	fn := d.styles.item.Render
	// if index == m.Index() {
	// 	fn = func(s ...string) string {
	// 		return i.styles.selectedItem.Render("> " + strings.Join(s, " "))
	// 	}
	// }

	_, _ = fmt.Fprint(w, fn(str))
}

type ProtoList struct {
	Protos []list.Item
}

func (pl *ProtoList) AddProto(p Proto) {
	pl.Protos = append(pl.Protos, p)
}

type ProtoListPane struct {
	list  list.Model
	style lipgloss.Style
}

func NewProtoListPane(pl ProtoList) ProtoListPane {
	list := list.New(pl.Protos, itemDelegate{}, 20, 20)
	list.SetShowHelp(false)
	list.Title = "Protobufs"
	return ProtoListPane{
		list: list,
		style: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(colors.BorderColor).
			Padding(1).
			Margin(1),
	}
}

func (p *ProtoListPane) UpdateSize(width int, height int) {
	p.style = p.style.Width(width - 2).Height(height - 2)
}

func (p ProtoListPane) Init() tea.Cmd {
	return nil
}

func (p ProtoListPane) Update(msg tea.Msg) (ProtoListPane, tea.Cmd) {
	var cmd tea.Cmd
	p.list, cmd = p.list.Update(msg)
	return p, cmd
}

func (p ProtoListPane) View() string {
	return p.style.Render(p.list.View())
}
