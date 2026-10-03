package tui

import (
	"github.com/EliasLd/goweeb/internal/app"
	"github.com/charmbracelet/lipgloss"
	"strings"
)

func viewForm(m Model) string {
	var form strings.Builder

	form.WriteString(
		titleStyle.Render(m.Title),
	)
	form.WriteString("\n\n")

	form.WriteString(labelStyle.Render("Destination folder"))
	form.WriteString("\n\n")

	if app.OutputDirLocked() {
		form.WriteString(
			lipgloss.NewStyle().
				Faint(true).
				Render(m.ScanDirInput.Value()),
		)

		form.WriteString("\n")

		form.WriteString(
			lipgloss.NewStyle().
				Faint(true).
				Italic(true).
				Render(
					"Managed by the container volume.",
				),
		)
	} else if m.Cursor == 0 {
		form.WriteString(m.ScanDirInput.View())
	} else {
		form.WriteString(
			lipgloss.NewStyle().
				Faint(true).
				Render(m.ScanDirInput.View()),
		)
	}

	form.WriteString("\n\n")

	providerLabel := "[ Select provider ]"

	if m.SelectedProviderLabel != "" {
		providerLabel = "Provider: " + m.SelectedProviderLabel
	}

	form.WriteString(
		renderButton(
			providerLabel,
			m.Cursor == 1,
			buttonSecondary,
		),
	)

	form.WriteString("\n\n")

	// Optional settings
	form.WriteString(
		renderButton(
			"[ Optional settings ]",
			m.Cursor == 2,
			buttonSecondary,
		),
	)
	form.WriteString("\n\n")

	if m.SearchReady {
		form.WriteString(
			renderButton(
				"[ Search manga ]",
				m.Cursor == 3,
				buttonPrimary,
			),
		)
	} else if m.IsDownloading {
		form.WriteString(
			disabledButtonStyle.Render(
				"Search manga (download in progress)",
			),
		)
	} else {
		form.WriteString(
			disabledButtonStyle.Render(
				"Search manga",
			),
		)
	}

	form.WriteString("\n\n")

	return renderViewWithFooter(
		m.Width,
		m.Height,
		form.String(),
		"↑/↓ navigate • Space/Enter select • Ctrl+L logs • Ctrl+C/Esc quit",
		lipgloss.Center,
		lipgloss.Center,
	)
}
