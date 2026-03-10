package core

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

// MarkdownToTelegramHTML converts common Markdown to Telegram-compatible HTML.
// Telegram supports: <b>, <i>, <s>, <code>, <pre>, <a href="">, <blockquote>.
// Markdown tables are converted to <pre> blocks so they display correctly
// (Telegram does not support HTML or Markdown table syntax).
func MarkdownToTelegramHTML(md string) string {
	var b strings.Builder
	b.Grow(len(md) + len(md)/4)

	lines := strings.Split(md, "\n")
	inCodeBlock := false
	codeLang := ""
	var codeLines []string

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "```") {
			if !inCodeBlock {
				inCodeBlock = true
				codeLang = strings.TrimPrefix(trimmed, "```")
				codeLines = nil
			} else {
				inCodeBlock = false
				if codeLang != "" {
					b.WriteString("<pre><code class=\"language-" + escapeHTML(codeLang) + "\">")
				} else {
					b.WriteString("<pre><code>")
				}
				b.WriteString(escapeHTML(strings.Join(codeLines, "\n")))
				b.WriteString("</code></pre>")
				if i < len(lines)-1 {
					b.WriteByte('\n')
				}
			}
			continue
		}

		if inCodeBlock {
			codeLines = append(codeLines, line)
			continue
		}

		// Markdown table: consecutive lines starting with | and containing |
		if isTableLine(trimmed) {
			tableLines := []string{line}
			j := i + 1
			for j < len(lines) {
				t := strings.TrimSpace(lines[j])
				if !isTableLine(t) {
					break
				}
				tableLines = append(tableLines, lines[j])
				j++
			}
			formatted := formatTableForTelegram(tableLines)
			b.WriteString("<pre>")
			b.WriteString(escapeHTML(formatted))
			b.WriteString("</pre>")
			if j < len(lines) {
				b.WriteByte('\n')
			}
			i = j - 1
			continue
		}

		// Headings → bold
		if heading := reHeading.FindString(line); heading != "" {
			rest := strings.TrimPrefix(line, heading)
			b.WriteString("<b>")
			b.WriteString(convertInlineHTML(rest))
			b.WriteString("</b>")
		} else if strings.HasPrefix(trimmed, "> ") || trimmed == ">" {
			quote := strings.TrimPrefix(line, "> ")
			if quote == ">" {
				quote = ""
			}
			b.WriteString("<blockquote>")
			b.WriteString(convertInlineHTML(quote))
			b.WriteString("</blockquote>")
		} else if reHorizontal.MatchString(trimmed) {
			b.WriteString("———")
		} else {
			b.WriteString(convertInlineHTML(line))
		}

		if i < len(lines)-1 {
			b.WriteByte('\n')
		}
	}

	// Handle unclosed code block
	if inCodeBlock && len(codeLines) > 0 {
		b.WriteString("<pre><code>")
		b.WriteString(escapeHTML(strings.Join(codeLines, "\n")))
		b.WriteString("</code></pre>")
	}

	return b.String()
}

// isTableLine returns true if the line looks like a Markdown table row (| cell | cell |).
func isTableLine(trimmed string) bool {
	if trimmed == "" {
		return false
	}
	if !strings.HasPrefix(trimmed, "|") {
		return false
	}
	return strings.Contains(trimmed[1:], "|")
}

// formatTableForTelegram converts Markdown table lines to a plain-text table
// with aligned columns, suitable for display inside <pre> in Telegram.
func formatTableForTelegram(tableLines []string) string {
	if len(tableLines) == 0 {
		return ""
	}
	var rows [][]string
	for _, line := range tableLines {
		cells := parseTableRow(line)
		if len(cells) == 0 {
			continue
		}
		rows = append(rows, cells)
	}
	if len(rows) == 0 {
		return ""
	}
	// Check if second row is separator (|---||---|)
	sepIdx := -1
	if len(rows) >= 2 {
		allSep := true
		for _, c := range rows[1] {
			s := strings.TrimSpace(strings.ReplaceAll(c, " ", ""))
			if s != "" && s != "-" && !strings.HasPrefix(s, ":") && !strings.HasSuffix(s, ":") {
				// allow :-: or :--- etc
				onlyDash := true
				for _, r := range s {
					if r != '-' && r != ':' {
						onlyDash = false
						break
					}
				}
				if !onlyDash {
					allSep = false
					break
				}
			}
		}
		if allSep && len(rows[1]) == len(rows[0]) {
			sepIdx = 1
		}
	}
	// Column count from first row
	nc := len(rows[0])
	widths := make([]int, nc)
	for r, row := range rows {
		if r == sepIdx {
			continue
		}
		for c, cell := range row {
			if c < nc {
				w := utf8.RuneCountInString(strings.TrimSpace(cell))
				if w > widths[c] {
					widths[c] = w
				}
			}
		}
	}
	// Build output
	var out strings.Builder
	for r, row := range rows {
		if r == sepIdx {
			// Separator row: output a line of dashes
			for c := 0; c < nc; c++ {
				if c > 0 {
					out.WriteString(" ")
				}
				for i := 0; i < widths[c]; i++ {
					out.WriteByte('-')
				}
			}
			out.WriteByte('\n')
			continue
		}
		for c := 0; c < nc && c < len(row); c++ {
			cell := strings.TrimSpace(row[c])
			if c > 0 {
				out.WriteString(" ")
			}
			out.WriteString(cell)
			for i := utf8.RuneCountInString(cell); i < widths[c]; i++ {
				out.WriteByte(' ')
			}
		}
		out.WriteByte('\n')
	}
	return strings.TrimRight(out.String(), "\n")
}

// parseTableRow splits a table row by | and returns trimmed cells (no leading/trailing empty).
func parseTableRow(line string) []string {
	line = strings.TrimSpace(line)
	if strings.HasPrefix(line, "|") {
		line = line[1:]
	}
	if strings.HasSuffix(line, "|") {
		line = line[:len(line)-1]
	}
	parts := strings.Split(line, "|")
	cells := make([]string, 0, len(parts))
	for _, p := range parts {
		cells = append(cells, strings.TrimSpace(p))
	}
	return cells
}

var (
	reInlineCodeHTML = regexp.MustCompile("`([^`]+)`")
	reBoldAstHTML    = regexp.MustCompile(`\*\*(.+?)\*\*`)
	reBoldUndHTML    = regexp.MustCompile(`__(.+?)__`)
	reItalicAstHTML  = regexp.MustCompile(`(?:^|[^*])\*([^*]+?)\*(?:[^*]|$)`)
	reStrikeHTML     = regexp.MustCompile(`~~(.+?)~~`)
	reLinkHTML       = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
)

// convertInlineHTML converts inline Markdown formatting to Telegram-compatible HTML.
//
// Each formatting pass (bold, strikethrough) protects its output as placeholders
// so that subsequent passes (italic) cannot match across HTML tag boundaries.
// Without this, input like `**bold *text***` would produce crossed tags
// `<b>bold <i>text</b></i>` which Telegram rejects.
func convertInlineHTML(s string) string {
	type placeholder struct {
		key  string
		html string
	}
	var phs []placeholder
	phIdx := 0

	nextPH := func(html string) string {
		key := "\x00PH" + string(rune('0'+phIdx)) + "\x00"
		phs = append(phs, placeholder{key: key, html: html})
		phIdx++
		return key
	}

	// 1. Extract inline code → placeholder (content escaped)
	s = reInlineCodeHTML.ReplaceAllStringFunc(s, func(m string) string {
		inner := m[1 : len(m)-1]
		return nextPH("<code>" + escapeHTML(inner) + "</code>")
	})

	// 2. Extract links → placeholder (text & URL escaped)
	s = reLinkHTML.ReplaceAllStringFunc(s, func(m string) string {
		sm := reLinkHTML.FindStringSubmatch(m)
		if len(sm) < 3 {
			return m
		}
		return nextPH(`<a href="` + escapeHTML(sm[2]) + `">` + escapeHTML(sm[1]) + `</a>`)
	})

	// 3. HTML-escape the entire remaining text.
	s = escapeHTML(s)

	// 4. Bold → placeholder (so italic regex can't cross bold boundaries)
	s = reBoldAstHTML.ReplaceAllStringFunc(s, func(m string) string {
		inner := m[2 : len(m)-2]
		return nextPH("<b>" + inner + "</b>")
	})
	s = reBoldUndHTML.ReplaceAllStringFunc(s, func(m string) string {
		inner := m[2 : len(m)-2]
		return nextPH("<b>" + inner + "</b>")
	})

	// 5. Strikethrough → placeholder
	s = reStrikeHTML.ReplaceAllStringFunc(s, func(m string) string {
		inner := m[2 : len(m)-2]
		return nextPH("<s>" + inner + "</s>")
	})

	// 6. Italic (applied last, on text with bold/strike already protected)
	s = reItalicAstHTML.ReplaceAllStringFunc(s, func(m string) string {
		idx := strings.Index(m, "*")
		if idx < 0 {
			return m
		}
		lastIdx := strings.LastIndex(m, "*")
		if lastIdx <= idx {
			return m
		}
		return m[:idx] + "<i>" + m[idx+1:lastIdx] + "</i>" + m[lastIdx+1:]
	})

	// 7. Restore all placeholders (may be nested, so iterate until stable)
	for i := 0; i < 3; i++ {
		changed := false
		for _, ph := range phs {
			if strings.Contains(s, ph.key) {
				s = strings.Replace(s, ph.key, ph.html, 1)
				changed = true
			}
		}
		if !changed {
			break
		}
	}

	return s
}

func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}

// SplitMessageCodeFenceAware splits text into chunks respecting code fence boundaries.
// When a chunk boundary falls inside a code block, the fence is closed at the end of
// the chunk and re-opened at the start of the next chunk.
func SplitMessageCodeFenceAware(text string, maxLen int) []string {
	if len(text) <= maxLen {
		return []string{text}
	}

	lines := strings.Split(text, "\n")
	var chunks []string
	var current []string
	currentLen := 0
	openFence := "" // the ``` opening line, or "" if outside code block

	for _, line := range lines {
		lineLen := len(line) + 1 // +1 for newline

		if currentLen+lineLen > maxLen && len(current) > 0 {
			chunk := strings.Join(current, "\n")
			if openFence != "" {
				chunk += "\n```"
			}
			chunks = append(chunks, chunk)

			current = nil
			currentLen = 0
			if openFence != "" {
				current = append(current, openFence)
				currentLen = len(openFence) + 1
			}
		}

		current = append(current, line)
		currentLen += lineLen

		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			if openFence != "" {
				openFence = ""
			} else {
				openFence = trimmed
			}
		}
	}

	if len(current) > 0 {
		chunk := strings.Join(current, "\n")
		if openFence != "" {
			chunk += "\n```"
		}
		chunks = append(chunks, chunk)
	}

	return chunks
}
