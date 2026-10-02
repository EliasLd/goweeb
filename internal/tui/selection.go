package tui

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Represents a generic selectable item
type SelectionItem struct {
	Label string
	Value string
}

// Implements list.Item interface
func (i SelectionItem) FilterValue() string { return i.Label }

// Custom item delegate for styling
type itemDelegate struct {
	Focused bool
}

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(SelectionItem)
	if !ok {
		return
	}

	str := fmt.Sprintf("%d. %s", index+1, i.Label)

	// Highlight selected item
	if index == m.Index() && d.Focused {
		s := lipgloss.NewStyle().
			Foreground(lipgloss.Color("226")).
			Bold(true).
			Render("> " + str)
		fmt.Fprint(w, s)
	} else {
		s := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Render(" " + str)
		fmt.Fprint(w, s)
	}
}

// Represents a generic selection screen
type SelectionModel struct {
	Title     string
	list      list.Model
	Cursor    int
	Selected  string
	Cancelled bool
	Width     int
	Height    int
}

func NewSelectionModel(title string, items []SelectionItem) SelectionModel {
	listItem := make([]list.Item, len(items))
	for i, item := range items {
		listItem[i] = item
	}

	// Create list
	delegate := itemDelegate{
		Focused: true,
	}
	l := list.New(listItem, delegate, 60, 14)
	l.Title = title
	l.SetShowStatusBar(false)
	l.SetShowPagination(true)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(true)

	l.Styles.Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("226")).
		Padding(0, 0, 1, 0)

	l.Styles.StatusBar = lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Padding(0, 0, 1, 0)

	return SelectionModel{
		Title:     title,
		list:      l,
		Selected:  "",
		Cancelled: false,
	}
}

func (m SelectionModel) Init() tea.Cmd {
	return nil
}

func (m SelectionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height

		minVisibleItems := 10
		maxVisibleItems := 20

		footerHeight := 2
		titleHeight := 3
		statusBarHeight := 2
		availableHeight := m.Height - footerHeight - titleHeight - statusBarHeight

		// Each item is 1 line
		visibleItems := availableHeight
		if visibleItems < minVisibleItems {
			visibleItems = minVisibleItems
		}
		if visibleItems > maxVisibleItems {
			visibleItems = maxVisibleItems
		}

		m.list.SetWidth(60)
		m.list.SetHeight(visibleItems + statusBarHeight)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "ctrl+c":
			m.Cancelled = true
			return m, nil

		case "enter":
			// Get selected item
			if i, ok := m.list.SelectedItem().(SelectionItem); ok {
				m.Selected = i.Value
			}
			return m, nil
		}
	}

	// Delegate to list
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m SelectionModel) View() string {
	listView := m.list.View()

	return renderViewWithFooter(
		m.Width,
		m.Height,
		listView,
		"↑/↓ navigate • Enter select • / filter • Ctrl+C/Esc cancel",
		lipgloss.Center,
		lipgloss.Center,
	)
}
