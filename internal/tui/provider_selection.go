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
	options := make(
		[]ProviderOption,
		0,
		len(providers)+1,
	)

	// The search screen queries every provider at once by default,
	// so the selection screen always offers an all-providers choice.
	options = append(
		options,
		ProviderOption{
			ID:       "",
			Label:    "All providers",
			Checkbox: Checkbox{Label: "All providers", Checked: currentProvider == ""},
		},
	)

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

	// A custom domain is inherently tied to one provider; it must
	// not leak into the all-providers search.
	if currentProvider == "" {
		currentDomain = ""
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

	if currentProvider == "" {
		model.SelectedLabel = "All providers"
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
		case "esc", "ctrl+c":
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

		case "enter", " ":
			switch {
			case m.isProviderCursor():
				m.selectCurrentProvider()
				m.Cursor = m.confirmCursor()
				return m, nil

			case m.Cursor == m.confirmCursor():
				// The All providers option is always available,
				// so confirming is always valid.
				m.Confirmed = true
				return m, nil
			}
		}

		if m.Cursor == m.domainCursor() &&
			m.SelectedID != "" {
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
		optionView :=
			option.Checkbox.View(
				m.Cursor == i,
			)

		optionView =
			markMouseZone(
				providerOptionZone(i),
				optionView,
			)

		b.WriteString(
			optionView,
		)

		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(labelStyle.Render("Custom domain (optional)"))
	b.WriteString("\n\n")

	domainView :=
		m.DomainInput.View()

	// The domain is dimmed when unfocused and when it cannot be
	// used (the all-providers search has no single provider).
	if m.Cursor !=
		m.domainCursor() ||
		m.SelectedID == "" {

		domainView =
			lipgloss.NewStyle().
				Faint(true).
				Render(
					domainView,
				)
	}

	domainView =
		markMouseZone(
			mouseZoneProviderDomain,
			domainView,
		)

	b.WriteString(
		domainView,
	)
	b.WriteString("\n\n")

	// The All providers option is a first-class choice, so the
	// confirm button stays selectable in every state.
	button :=
		renderButton(
			"Confirm",
			m.Cursor ==
				m.confirmCursor(),
			buttonPrimary,
		)

	button =
		markMouseZone(
			mouseZoneProviderConfirm,
			button,
		)

	b.WriteString(button)

	return renderViewWithFooter(
		m.Width,
		m.Height,
		b.String(),
		"↑/↓ navigate • Space/Enter select • Ctrl+C/Esc cancel",
		lipgloss.Center,
		lipgloss.Center,
	)
}

func (m *ProviderSelectionModel) selectCurrentProvider() {
	if !m.isProviderCursor() {
		return
	}

	selected := &m.Options[m.Cursor]

	// Avoid accidentally keeping a custom domain that belongs to
	// another selection: a domain only applies to a single provider.
	if selected.ID != m.SelectedID {
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

	// The domain is only meaningful when a single provider is
	// selected.
	if m.Cursor == m.domainCursor() &&
		m.SelectedID != "" {
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
