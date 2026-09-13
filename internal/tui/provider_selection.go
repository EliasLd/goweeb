package tui

import (
	"strings"

	"github.com/EliasLd/goweeb/internal/source"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ProviderOption struct {
	ID       string
	Label    string
	Checkbox Checkbox
}

type ProviderSelectionModel struct {
	Options []ProviderOption

	Cursor int

	SelectedID    string
	SelectedLabel string

	DomainInput textinput.Model

	Confirmed bool
	Cancelled bool

	Width  int
	Height int
}

func NewProviderSelectionModel(
	providers []source.ProviderInfo,
	currentProvider string,
	currentDomain string,
) ProviderSelectionModel {
	options := make([]ProviderOption, 0, len(providers))

	for _, provider := range providers {
		options = append(options, ProviderOption{
			ID:    provider.ID,
			Label: provider.Label,
			Checkbox: Checkbox{
				Label:   provider.Label,
				Checked: provider.ID == currentProvider,
			},
		})
	}

	domain := textinput.New()
	domain.Placeholder = "Optional custom base URL"
	domain.Prompt = "> "
	domain.Width = 60
	domain.SetValue(currentDomain)
	domain.Blur()

	model := ProviderSelectionModel{
		Options:     options,
		SelectedID:  currentProvider,
		DomainInput: domain,
	}

	for _, option := range options {
		if option.ID == currentProvider {
			model.SelectedLabel = option.Label
			break
		}
	}

	model.updateFocus()

	return model
}

func (m ProviderSelectionModel) Init() tea.Cmd {
	return nil
}

func (m ProviderSelectionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit

		case "esc":
			m.Cancelled = true
			return m, nil

		case "up":
			if m.Cursor > 0 {
				m.Cursor--
			}
			m.updateFocus()
			return m, nil

		case "down", "tab":
			if m.Cursor < m.maxCursor() {
				m.Cursor++
			}
			m.updateFocus()
			return m, nil

		case "shift+tab":
			if m.Cursor > 0 {
				m.Cursor--
			}
			m.updateFocus()
			return m, nil

		case " ":
			if m.isProviderCursor() {
				m.selectCurrentProvider()
				return m, nil
			}

		case "enter":
			switch {
			case m.isProviderCursor():
				m.selectCurrentProvider()
				return m, nil

			case m.Cursor == m.confirmCursor():
				if m.SelectedID != "" {
					m.Confirmed = true
				}
				return m, nil
			}
		}

		if m.Cursor == m.domainCursor() {
			var cmd tea.Cmd
			m.DomainInput, cmd = m.DomainInput.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

func (m ProviderSelectionModel) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Select provider"))
	b.WriteString("\n\n")

	b.WriteString(
		lipgloss.NewStyle().
			Faint(true).
			Render("Choose the provider used to search and download manga."),
	)
	b.WriteString("\n\n")

	for i, option := range m.Options {
		b.WriteString(option.Checkbox.View(m.Cursor == i))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(labelStyle.Render("Custom domain (optional)"))
	b.WriteString("\n\n")

	if m.Cursor == m.domainCursor() {
		b.WriteString(m.DomainInput.View())
	} else {
		b.WriteString(
			lipgloss.NewStyle().
				Faint(true).
				Render(m.DomainInput.View()),
		)
	}

	b.WriteString("\n\n")

	button := "[ Confirm ]"

	if m.SelectedID == "" {
		button = lipgloss.NewStyle().
			Faint(true).
			Render(button)
	} else if m.Cursor == m.confirmCursor() {
		button = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("226")).
			Render(button)
	} else {
		button = lipgloss.NewStyle().
			Faint(true).
			Render(button)
	}

	b.WriteString(button)
	b.WriteString("\n\n")

	footer := lipgloss.NewStyle().
		Faint(true).
		Render("↑/↓ navigate • Space/Enter select • Esc cancel")

	b.WriteString(footer)

	content := b.String()

	boxWidth := lipgloss.Width(content)
	boxHeight := lipgloss.Height(content)

	horizontalMargin := max(0, (m.Width-boxWidth)/2)
	verticalMargin := max(0, (m.Height-boxHeight)/2)

	return lipgloss.NewStyle().
		MarginTop(verticalMargin).
		MarginLeft(horizontalMargin).
		Render(content)
}

func (m *ProviderSelectionModel) selectCurrentProvider() {
	if !m.isProviderCursor() {
		return
	}

	selected := &m.Options[m.Cursor]

	// Avoid accidentally keeping a custom domain belonging
	// to another provider.
	if m.SelectedID != "" && m.SelectedID != selected.ID {
		m.DomainInput.SetValue("")
	}

	for i := range m.Options {
		m.Options[i].Checkbox.Checked = false
	}

	selected.Checkbox.Checked = true

	m.SelectedID = selected.ID
	m.SelectedLabel = selected.Label
}

func (m *ProviderSelectionModel) updateFocus() {
	m.DomainInput.Blur()

	if m.Cursor == m.domainCursor() {
		m.DomainInput.Focus()
	}
}

func (m ProviderSelectionModel) isProviderCursor() bool {
	return m.Cursor >= 0 && m.Cursor < len(m.Options)
}

func (m ProviderSelectionModel) domainCursor() int {
	return len(m.Options)
}

func (m ProviderSelectionModel) confirmCursor() int {
	return len(m.Options) + 1
}

func (m ProviderSelectionModel) maxCursor() int {
	return m.confirmCursor()
}
