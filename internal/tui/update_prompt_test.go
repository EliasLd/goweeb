package tui

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/EliasLd/goweeb/internal/update"
)

func TestUpdateCheckResultOpensTheOfferBox(t *testing.T) {
	m, _ := handleUpdateCheckResult(
		updateCheckResultMsg{
			release: &update.Release{Tag: "v4.7.0"},
		},
		Model{},
	)

	if !m.UpdatePromptVisible {
		t.Error("expected the offer box to open")
	}

	if m.UpdateOffer == nil || m.UpdateOffer.Tag != "v4.7.0" {
		t.Errorf("unexpected offer: %+v", m.UpdateOffer)
	}

	// No is the default of the choice.
	if m.UpdatePromptCursor != updateNo {
		t.Errorf("cursor = %d, want the No button", m.UpdatePromptCursor)
	}
}

func TestUpdateCheckResultDropsAFailedLookup(t *testing.T) {
	m, _ := handleUpdateCheckResult(
		updateCheckResultMsg{err: fmt.Errorf("offline")},
		Model{},
	)

	if m.UpdatePromptVisible {
		t.Error("a failed check must not open the offer box")
	}

	m, _ = handleUpdateCheckResult(
		updateCheckResultMsg{},
		Model{},
	)

	if m.UpdatePromptVisible {
		t.Error("an absent offer must not open the box")
	}
}

func TestUpdatePromptDecline(t *testing.T) {
	m := Model{
		UpdateOffer:         &update.Release{Tag: "v4.7.0"},
		UpdatePromptVisible: true,
		UpdatePromptCursor:  updateNo,
	}

	next, _ := handleUpdatePrompt(
		tea.KeyMsg{Type: tea.KeyEnter},
		m,
	)

	if next.UpdatePromptVisible {
		t.Error("the decline should close the box")
	}

	if next.UpdateOffer != nil {
		t.Error("the offer should be spent after the decision")
	}
}

func TestUpdatePromptEscAlsoDeclines(t *testing.T) {
	m := Model{
		UpdateOffer:         &update.Release{Tag: "v4.7.0"},
		UpdatePromptVisible: true,
		UpdatePromptCursor:  0,
	}

	next, _ := handleUpdatePrompt(
		tea.KeyMsg{Type: tea.KeyEsc},
		m,
	)

	if next.UpdatePromptVisible {
		t.Error("esc should close the box")
	}
}

func TestUpdatePromptConfirmRunsTheInstall(t *testing.T) {
	m := Model{
		UpdateOffer:         &update.Release{Tag: "v4.7.0"},
		UpdatePromptVisible: true,
		UpdatePromptCursor:  0,
	}

	next, cmd := handleUpdatePrompt(
		tea.KeyMsg{Type: tea.KeyEnter},
		m,
	)

	if !next.UpdateRunning {
		t.Error("the confirmation should lock the box")
	}

	if cmd == nil {
		t.Error("expected the installation to launch")
	}
}

func TestUpdateApplyDoneClosesTheBox(t *testing.T) {
	m := Model{
		UpdateOffer:         &update.Release{Tag: "v4.7.0"},
		UpdatePromptVisible: true,
		UpdatePromptCursor:  0,
		UpdateRunning:       true,
	}

	next, cmd := handleUpdatePrompt(
		updateApplyDoneMsg{},
		m,
	)

	if next.UpdatePromptVisible || next.UpdateRunning {
		t.Errorf("the outcome should free the TUI: %+v", next)
	}

	if cmd == nil {
		t.Error("expected the outcome alert")
	}
}
