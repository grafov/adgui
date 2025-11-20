package ui

import (
	"regexp"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// parseAnsi converts a string with ANSI escape codes into a RichText widget
// with properly formatted segments. Currently handles:
// - \033[1m or \x1b[1m or [1m: Bold text
// - \033[0m or \x1b[0m or [0m: Reset formatting
func parseAnsi(text string) *widget.RichText {
	if text == "" {
		return widget.NewRichTextWithText("")
	}

	var segments []widget.RichTextSegment
	
	// Pattern to match ANSI escape sequences: ESC[1m (bold) or ESC[0m (reset)
	// Also handles raw [1m and [0m sequences
	ansiPattern := regexp.MustCompile(`(\x1b\[1m|\033\[1m|\[1m|\x1b\[0m|\033\[0m|\[0m)`)
	
	matches := ansiPattern.FindAllStringIndex(text, -1)
	if len(matches) == 0 {
		// No ANSI codes found, return plain text
		return widget.NewRichTextWithText(text)
	}
	
	bold := false
	lastPos := 0
	
	for _, match := range matches {
		start, end := match[0], match[1]
		code := text[start:end]
		
		// Add text segment before this ANSI code
		if start > lastPos {
			textPart := text[lastPos:start]
			if textPart != "" {
				style := widget.RichTextStyleInline
				if bold {
					style = widget.RichTextStyle{
						ColorName: widget.RichTextStyleInline.ColorName,
						Inline:    true,
						SizeName:  widget.RichTextStyleInline.SizeName,
						TextStyle: widget.RichTextStyleInline.TextStyle,
					}
					style.TextStyle.Bold = true
				}
				segments = append(segments, &widget.TextSegment{
					Style: style,
					Text:  textPart,
				})
			}
		}
		
		// Update bold state based on ANSI code
		if strings.Contains(code, "1m") {
			bold = true
		} else if strings.Contains(code, "0m") {
			bold = false
		}
		
		lastPos = end
	}
	
	// Add remaining text after last ANSI code
	if lastPos < len(text) {
		textPart := text[lastPos:]
		if textPart != "" {
			style := widget.RichTextStyleInline
			if bold {
				style = widget.RichTextStyle{
					ColorName: widget.RichTextStyleInline.ColorName,
					Inline:    true,
					SizeName:  widget.RichTextStyleInline.SizeName,
					TextStyle: widget.RichTextStyleInline.TextStyle,
				}
				style.TextStyle.Bold = true
			}
			segments = append(segments, &widget.TextSegment{
				Style: style,
				Text:  textPart,
			})
		}
	}
	
	if len(segments) == 0 {
		return widget.NewRichTextWithText("")
	}
	
	richText := widget.NewRichText(segments...)
	richText.Wrapping = fyne.TextWrapWord
	return richText
}

