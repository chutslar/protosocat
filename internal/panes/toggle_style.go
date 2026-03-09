package panes

import "charm.land/lipgloss/v2"

type ToggleStyle struct {
	ActiveStyle   lipgloss.Style
	InactiveStyle lipgloss.Style
}

func (s ToggleStyle) Width(width int) ToggleStyle {
	return ToggleStyle{
		ActiveStyle:   s.ActiveStyle.Width(width),
		InactiveStyle: s.InactiveStyle.Width(width),
	}
}

func (s ToggleStyle) Height(height int) ToggleStyle {
	return ToggleStyle{
		ActiveStyle:   s.ActiveStyle.Height(height),
		InactiveStyle: s.InactiveStyle.Height(height),
	}
}

func (s ToggleStyle) GetVerticalFrameSize() int {
	return s.ActiveStyle.GetVerticalFrameSize()
}

func (s ToggleStyle) GetHorizontalFrameSize() int {
	return s.ActiveStyle.GetHorizontalFrameSize()
}

func (s ToggleStyle) GetStyle(active bool) lipgloss.Style {
	if active {
		return s.ActiveStyle
	}
	return s.InactiveStyle
}
