package tui

import (
	"fmt"

	"github.com/EliasLd/goweeb/internal/app"
	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
)

var mouseZones = zone.New()

const (
	mouseZoneMainDestination = "main-destination"
	mouseZoneMainProvider    = "main-provider"
	mouseZoneMainOptions     = "main-options"
	mouseZoneMainSearch      = "main-search"

	mouseZoneRangeInput    = "range-input"
	mouseZoneRangeAll      = "range-all"
	mouseZoneRangeDownload = "range-download"

	mouseZoneProviderDomain  = "provider-domain"
	mouseZoneProviderConfirm = "provider-confirm"

	mouseZoneOptionalEbook = "optional-ebook"
	mouseZoneSearchInput   = "search-input"

	mouseZoneMainLogs = "main-logs"
	mouseZoneLogClose = "log-close"
)

func providerOptionZone(
	index int,
) string {
	return fmt.Sprintf(
		"provider-option-%d",
		index,
	)
}

func searchResultZone(
	index int,
) string {
	return fmt.Sprintf(
		"search-result-%d",
		index,
	)
}

func selectionItemZone(
	index int,
) string {
	return fmt.Sprintf(
		"selection-item-%d",
		index,
	)
}

func markMouseZone(
	id string,
	view string,
) string {
	return mouseZones.Mark(
		id,
		view,
	)
}

func mouseZoneHit(
	id string,
	msg tea.MouseMsg,
) bool {
	z := mouseZones.Get(id)

	return z != nil &&
		z.InBounds(msg)
}

func isLeftMouseRelease(
	msg tea.MouseMsg,
) bool {
	return msg.Action ==
		tea.MouseActionRelease &&
		msg.Button ==
			tea.MouseButtonLeft
}

func handleMouseUpdate(
	msg tea.MouseMsg,
	m Model,
) (Model, tea.Cmd, bool) {
	if msg.Action ==
		tea.MouseActionMotion {
		return handleMouseHover(
			msg,
			m,
		)
	}

	if !isLeftMouseRelease(msg) {
		return m, nil, false
	}

	if m.LogsVisible {
		return m, nil, false
	}

	switch m.State {
	case StateForm,
		StateDownloading:
		return handleMainMouse(
			msg,
			m,
		)

	case StateRangeSelection:
		return handleRangeMouse(
			msg,
			m,
		)

	case StateProviderSelection:
		return handleProviderMouse(
			msg,
			m,
		)

	case StateOptionalSettings:
		return handleOptionalMouse(
			msg,
			m,
		)

	case StateInteractiveSearch:
		return handleSearchMouse(
			msg,
			m,
		)
	}

	return m, nil, false
}

func handleMainMouse(
	msg tea.MouseMsg,
	m Model,
) (Model, tea.Cmd, bool) {
	switch {
	case mouseZoneHit(
		mouseZoneMainLogs,
		msg,
	):
		return toggleLogOverlay(m), nil, true

	case mouseZoneHit(
		mouseZoneMainDestination,
		msg,
	):
		if app.OutputDirLocked() {
			return m, nil, true
		}

		if m.Cursor != 0 {
			m.Cursor = 0
			m = updateFocus(m)

			return m, nil, true
		}

	case mouseZoneHit(
		mouseZoneMainProvider,
		msg,
	):
		if m.Cursor != 1 {
			m.Cursor = 1
			m = updateFocus(m)

			return m, nil, true
		}

	case mouseZoneHit(
		mouseZoneMainOptions,
		msg,
	):
		if m.Cursor != 2 {
			m.Cursor = 2
			m = updateFocus(m)

			return m, nil, true
		}

	case mouseZoneHit(
		mouseZoneMainSearch,
		msg,
	):
		if !m.SearchReady {
			return m, nil, true
		}

		if m.Cursor != 3 {
			m.Cursor = 3
			m = updateFocus(m)

			return m, nil, true
		}

	default:
		return m, nil, false
	}

	next, cmd :=
		handleFormUpdate(
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
			m,
		)

	return next, cmd, true
}

func handleRangeMouse(
	msg tea.MouseMsg,
	m Model,
) (Model, tea.Cmd, bool) {
	switch {
	case mouseZoneHit(
		mouseZoneRangeInput,
		msg,
	):
		if m.AllCheckbox.Checked {
			return m, nil, true
		}

		if m.Cursor != 0 {
			m.Cursor = 0
			m = updateRangeFocus(m)
		}

		return m, nil, true

	case mouseZoneHit(
		mouseZoneRangeAll,
		msg,
	):
		m.Cursor = 1
		m.AllCheckbox.Toggle()
		m = updateRangeFocus(m)

		return m, nil, true

	case mouseZoneHit(
		mouseZoneRangeDownload,
		msg,
	):
		if m.Cursor != 2 {
			m.Cursor = 2
			m = updateRangeFocus(m)

			return m, nil, true
		}

	default:
		return m, nil, false
	}

	next, cmd :=
		handleRangeUpdate(
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
			m,
		)

	return next, cmd, true
}

func handleProviderMouse(
	msg tea.MouseMsg,
	m Model,
) (Model, tea.Cmd, bool) {
	pm :=
		&m.ProviderSelectionModel

	for i := range pm.Options {
		if !mouseZoneHit(
			providerOptionZone(i),
			msg,
		) {
			continue
		}

		pm.Cursor = i
		pm.updateFocus()

		// Reuse the existing Enter behavior to select the provider.
		next, cmd := handleProviderSelectionUpdate(
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
			m,
		)

		return next, cmd, true
	}

	if mouseZoneHit(
		mouseZoneProviderDomain,
		msg,
	) {
		if pm.Cursor !=
			pm.domainCursor() {

			pm.Cursor =
				pm.domainCursor()

			pm.updateFocus()
		}

		return m, nil, true
	}

	if mouseZoneHit(
		mouseZoneProviderConfirm,
		msg,
	) {
		if pm.SelectedID == "" {
			return m, nil, true
		}

		if pm.Cursor !=
			pm.confirmCursor() {

			pm.Cursor =
				pm.confirmCursor()

			pm.updateFocus()

			return m, nil, true
		}

		next, cmd :=
			handleProviderSelectionUpdate(
				tea.KeyMsg{
					Type: tea.KeyEnter,
				},
				m,
			)

		return next, cmd, true
	}

	return m, nil, false
}

func handleSearchMouse(
	msg tea.MouseMsg,
	m Model,
) (Model, tea.Cmd, bool) {
	if mouseZoneHit(
		mouseZoneSearchInput,
		msg,
	) {
		if m.SearchFocus !=
			searchFocusInput {

			m.SearchFocus =
				searchFocusInput

			m.SearchInput.Focus()

			m.SearchList.SetDelegate(
				itemDelegate{
					Focused:    false,
					ZonePrefix: "search-result-",
				},
			)
		}

		return m, nil, true
	}

	for i := range m.SearchList.Items() {

		if !mouseZoneHit(
			searchResultZone(i),
			msg,
		) {
			continue
		}

		alreadyFocused :=
			m.SearchFocus ==
				searchFocusResults &&
				m.SearchList.Index() == i

		m.SearchList.Select(i)

		m.SearchFocus =
			searchFocusResults

		m.SearchInput.Blur()

		m.SearchList.SetDelegate(
			itemDelegate{
				Focused:    true,
				ZonePrefix: "search-result-",
			},
		)

		if !alreadyFocused {
			return m, nil, true
		}

		next, cmd :=
			handleInteractiveSearchUpdate(
				tea.KeyMsg{
					Type: tea.KeyEnter,
				},
				m,
			)

		return next, cmd, true
	}

	return m, nil, false
}

func handleOptionalMouse(
	msg tea.MouseMsg,
	m Model,
) (Model, tea.Cmd, bool) {
	if !mouseZoneHit(
		mouseZoneOptionalEbook,
		msg,
	) {
		return m, nil, false
	}

	m.OptionalCursor = 0

	next, cmd :=
		handleOptionalSettingsUpdate(
			tea.KeyMsg{
				Type: tea.KeySpace,
			},
			m,
		)

	return next, cmd, true
}

func handleMouseHover(
	msg tea.MouseMsg,
	m Model,
) (Model, tea.Cmd, bool) {
	hoveredAction := ""

	switch {
	case m.LogsVisible &&
		mouseZoneHit(
			mouseZoneLogClose,
			msg,
		):
		hoveredAction =
			mouseZoneLogClose

	case !m.LogsVisible &&
		isMainViewState(m.State) &&
		mouseZoneHit(
			mouseZoneMainLogs,
			msg,
		):
		hoveredAction =
			mouseZoneMainLogs
	}

	if m.HoveredAction ==
		hoveredAction {
		return m, nil, false
	}

	m.HoveredAction =
		hoveredAction

	if hoveredAction == "" {
		return m,
			setMousePointer(""),
			true
	}

	return m,
		setMousePointer("pointer"),
		true
}

func setMousePointer(
	shape string,
) tea.Cmd {
	return func() tea.Msg {
		if shape == "" {
			fmt.Print(
				"\x1b]22;\x1b\\",
			)

			return nil
		}

		fmt.Printf(
			"\x1b]22;%s\x1b\\",
			shape,
		)

		return nil
	}
}
