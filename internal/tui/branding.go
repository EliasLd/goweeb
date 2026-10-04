package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const asciiArt = `
                                                            
                                                   ▄▄       
                                                   ██       
  ▄███▄██   ▄████▄  ██      ██  ▄████▄    ▄████▄   ██▄███▄  
 ██▀  ▀██  ██▀  ▀██ ▀█  ██  █▀ ██▄▄▄▄██  ██▄▄▄▄██  ██▀  ▀██ 
 ██    ██  ██    ██  ██▄██▄██  ██▀▀▀▀▀▀  ██▀▀▀▀▀▀  ██    ██ 
 ▀██▄▄███  ▀██▄▄██▀  ▀██  ██▀  ▀██▄▄▄▄█  ▀██▄▄▄▄█  ███▄▄██▀ 
  ▄▀▀▀ ██    ▀▀▀▀     ▀▀  ▀▀     ▀▀▀▀▀     ▀▀▀▀▀   ▀▀ ▀▀▀   
  ▀████▀▀                                                   
                                                            
`

const (
	mainBrandKatakana = "ゴウィーブ"

	mainBrandTagline = "manga downloads, straight from your terminal"

	mainGradientStart = "#5B7CFF"
	mainGradientEnd   = "#5EE7F7"
)

func gradientText(
	text string,
	startHex string,
	endHex string,
) string {
	lines := strings.Split(
		text,
		"\n",
	)

	maxWidth := 0

	for _, line := range lines {
		width := len(
			[]rune(line),
		)

		if width > maxWidth {
			maxWidth = width
		}
	}

	if maxWidth <= 1 {
		return text
	}

	startR, startG, startB :=
		parseHexColor(startHex)

	endR, endG, endB :=
		parseHexColor(endHex)

	var result strings.Builder

	for lineIndex, line := range lines {
		runes := []rune(line)

		for column, r := range runes {
			if r == ' ' {
				result.WriteRune(r)

				continue
			}

			progress :=
				float64(column) /
					float64(maxWidth-1)

			red := interpolateChannel(
				startR,
				endR,
				progress,
			)

			green := interpolateChannel(
				startG,
				endG,
				progress,
			)

			blue := interpolateChannel(
				startB,
				endB,
				progress,
			)

			color := fmt.Sprintf(
				"#%02X%02X%02X",
				red,
				green,
				blue,
			)

			result.WriteString(
				lipgloss.NewStyle().
					Foreground(
						lipgloss.Color(color),
					).
					Bold(true).
					Render(
						string(r),
					),
			)
		}

		if lineIndex <
			len(lines)-1 {
			result.WriteString("\n")
		}
	}

	return result.String()
}

func parseHexColor(
	value string,
) (int, int, int) {
	value = strings.TrimPrefix(
		value,
		"#",
	)

	red, _ := strconv.ParseInt(
		value[0:2],
		16,
		64,
	)

	green, _ := strconv.ParseInt(
		value[2:4],
		16,
		64,
	)

	blue, _ := strconv.ParseInt(
		value[4:6],
		16,
		64,
	)

	return int(red),
		int(green),
		int(blue)
}

func interpolateChannel(
	start int,
	end int,
	progress float64,
) int {
	return int(
		float64(start) +
			float64(end-start)*progress,
	)
}
