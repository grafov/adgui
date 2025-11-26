package ui

import (
	"regexp"

	"fyne.io/fyne/v2/widget"
)

var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func parseAnsi(text string) *widget.RichText {
	segments := []widget.RichTextSegment{}
	matches := ansiRegex.FindAllStringIndex(text, -1)

	lastIndex := 0
	for _, match := range matches {
		// Add text before the ANSI code
		if match[0] > lastIndex {
			segments = append(segments, &widget.TextSegment{Text: text[lastIndex:match[0]]})
		}

		// Handle the ANSI code (for now, we just strip it)
		// In a real implementation, you would parse the code and apply styles.
		lastIndex = match[1]
	}

	// Add any remaining text
	if lastIndex < len(text) {
		segments = append(segments, &widget.TextSegment{Text: text[lastIndex:]})
	}

	// A simple example of styling - you can expand this
	for i, segment := range segments {
		if textSegment, ok := segment.(*widget.TextSegment); ok {
			if i%2 == 1 { // Example: style odd segments
				textSegment.Style.ColorName = "primary"
			}
		}
	}

	return widget.NewRichText(segments...)
}
