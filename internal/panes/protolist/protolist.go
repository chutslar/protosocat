package protolist

import (
	"fmt"
	"protosocat/internal/colors"
	"protosocat/internal/panes"
	"protosocat/internal/protos"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type ProtoListPane struct {
	list          []protos.Message
	favorites     []bool
	onlyFavorites bool
	selectedIndex int
	style         panes.ToggleStyle
	headerStyle   lipgloss.Style
	itemStyle     lipgloss.Style
	directory     string
	IsActiveTab   bool
}

func NewProtoListPane(pl []protos.Message, directory string) ProtoListPane {
	return ProtoListPane{
		list:          pl,
		favorites:     make([]bool, len(pl)),
		onlyFavorites: false,
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
		headerStyle: lipgloss.NewStyle().Underline(true),
		itemStyle:   lipgloss.NewStyle().PaddingLeft(1).MarginTop(1),
		directory:   directory,
	}
}

func (p *ProtoListPane) SetActive(active bool) {
	p.IsActiveTab = active
}

func (p *ProtoListPane) UpdateSize(width int, height int) {
	p.style = p.style.Width(width - 2).Height(height - 2)
}

func (p *ProtoListPane) Up() {
	if p.selectedIndex > 0 {
		p.selectedIndex--
	} else {
		p.selectedIndex = len(p.list) - 1
	}
	if p.onlyFavorites && !p.favorites[p.selectedIndex] {
		p.Up()
	}
}

func (p *ProtoListPane) Down() {
	if p.selectedIndex < len(p.list)-1 {
		p.selectedIndex++
	} else {
		p.selectedIndex = 0
	}
	if p.onlyFavorites && !p.favorites[p.selectedIndex] {
		p.Down()
	}
}

func (p ProtoListPane) Init() tea.Cmd {
	return nil
}

func (p ProtoListPane) Update(msg tea.Msg) (ProtoListPane, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "up", "j":
			p.Up()
		case "down", "k":
			p.Down()
		case "f":
			p.favorites[p.selectedIndex] = !p.favorites[p.selectedIndex]
		case "*":
			p.onlyFavorites = !p.onlyFavorites
		case "d":
			return p, panes.SwitchToDetails(p.list[p.selectedIndex])
		}
	}

	return p, nil
}

func (p ProtoListPane) ViewProto(index int) string {
	prefix := ""
	if index == p.selectedIndex {
		prefix = "> "
	}

	star := ""
	if p.favorites[index] {
		star = lipgloss.NewStyle().Bold(true).Render("◆")
	}

	proto := p.list[index]
	desc := lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(250)).
		Render(fmt.Sprintf("(%s)", proto.ShortPath))
	return p.itemStyle.Render(
		fmt.Sprintf("%s%s %s %s", prefix, proto.Descriptor.Name(), desc, star),
	)
}

func (p ProtoListPane) View() string {
	var strs []string
	header := p.headerStyle.Render(fmt.Sprintf("Protobufs in %s", p.directory))
	strs = append(strs, header)
	for i := range len(p.list) {
		if p.onlyFavorites && !p.favorites[i] {
			continue
		}
		strs = append(strs, p.ViewProto(i))
	}

	return p.style.GetStyle(p.IsActiveTab).Render(
		lipgloss.JoinVertical(
			lipgloss.Top,
			strs...,
		),
	)
}
