package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"go.dalton.dog/bubbleup"

	"github.com/EliasLd/goweeb/internal/update"
	"github.com/EliasLd/goweeb/internal/version"
)

// updateNo is the cursor index of the decline button; No is the
// default of the yes/no choice.
const updateNo = 1

// fetchUpdateOffer runs the startup update check for the TUI; a
// develop build checks nothing. The check failing or finding
// nothing sends an empty offer: the surface is for newer versions
// only, and no failure may stand in the way of the TUI.
func fetchUpdateOffer() tea.Cmd {
	if version.IsDevelop() {
		return nil
	}

	return func() tea.Msg {
		release, err := update.Check(version.String(), update.FlavorTUI, nil)

		return updateCheckResultMsg{release: release, err: err}
	}
}

// installUpdate runs the download, verification and replacement in
// the background; the box stays open until the outcome lands.
func installUpdate(release *update.Release) tea.Cmd {
	return func() tea.Msg {
		return updateApplyDoneMsg{err: update.Apply(release, nil)}
	}
}

// handleUpdateCheckResult opens the offer box when the release is
// newer; anything else is dropped, the TUI having started as normal.
func handleUpdateCheckResult(
	msg updateCheckResultMsg,
	m Model,
) (Model, tea.Cmd) {
	if msg.err != nil || msg.release == nil {
		return m, nil
	}

	m.UpdateOffer = msg.release
	m.UpdatePromptVisible = true
	m.UpdatePromptCursor = updateNo
	m.HoveredAction = ""

	return m, nil
}

// closeUpdatePrompt dismisses the offer; the offer is spent either
// way, the footer version returns to its quiet tone.
func closeUpdatePrompt(m Model) Model {
	m.UpdateOffer = nil
	m.UpdatePromptVisible = false
	m.UpdatePromptCursor = updateNo
	m.UpdateRunning = false
	m.HoveredAction = ""

	return m
}

// handleUpdatePrompt is the topmost surface: while the offer box is
// open it consumes every message until the choice is made or the
// installation completes.
func handleUpdatePrompt(
	msg tea.Msg,
	m Model,
) (Model, tea.Cmd) {
	if msg, ok := msg.(updateApplyDoneMsg); ok {
		return handleUpdateApplyDone(msg, m)
	}

	key, ok := msg.(tea.KeyMsg)
	if !ok || m.UpdateRunning {
		return m, nil
	}

	switch key.String() {
	case "esc", "ctrl+c":
		return closeUpdatePrompt(m), nil

	case "up", "down", "left", "right", "tab":
		if m.UpdatePromptCursor == updateNo {
			m.UpdatePromptCursor = 0
		} else {
			m.UpdatePromptCursor = updateNo
		}

		return m, nil

	case "enter":
		if m.UpdatePromptCursor == updateNo {
			return closeUpdatePrompt(m), nil
		}

		return confirmUpdateInstall(m)
	}

	return m, nil
}

// confirmUpdateInstall locks the box and launches the installation
// in the background, announcing it as a running alert.
func confirmUpdateInstall(m Model) (Model, tea.Cmd) {
	m.UpdateRunning = true

	return m, tea.Batch(
		installUpdate(m.UpdateOffer),
		m.AlertModel.NewAlertCmd(
			bubbleup.InfoKey,
			fmt.Sprintf(
				"Updating to %s...",
				m.UpdateOffer.Tag,
			),
		),
	)
}

// handleUpdateApplyDone reports the installation outcome and frees
// the TUI: the new version starts on the next run either way.
func handleUpdateApplyDone(
	msg updateApplyDoneMsg,
	m Model,
) (Model, tea.Cmd) {
	tag := ""

	if m.UpdateOffer != nil {
		tag = m.UpdateOffer.Tag
	}

	closed := closeUpdatePrompt(m)

	alert := bubbleup.InfoKey
	text := "Updated: restart goweeb to use the new version."

	if msg.err != nil {
		alert = bubbleup.ErrorKey
		text = "The update failed; the current version stayed."

		closed.Logs = append(
			closed.Logs,
			fmt.Sprintf("The update to %s failed: %v", tag, msg.err),
		)
	}

	return closed, closed.AlertModel.NewAlertCmd(alert, text)
}
