package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

func Start() error {
	p := tea.NewProgram(
		InitialModel(),
		tea.WithAltScreen(),
		tea.WithMouseAllMotion(),
	)
	if _, err := p.Run(); err != nil {
		return err
	}
	return nil
}
