package views

import (
	"bytes"
	"strings"
	"sync"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

var (
	highlightOnce sync.Once
	yamlLexer     chroma.Lexer
	jsonLexer     chroma.Lexer
	termFormatter chroma.Formatter
	kviewStyle    *chroma.Style
)

func initHighlight() {
	highlightOnce.Do(func() {
		yamlLexer = lexers.Get("yaml")
		if yamlLexer != nil {
			yamlLexer = chroma.Coalesce(yamlLexer)
		}

		jsonLexer = lexers.Get("json")
		if jsonLexer != nil {
			jsonLexer = chroma.Coalesce(jsonLexer)
		}

		termFormatter = formatters.Get("terminal16m")

		kviewStyle = styles.Register(chroma.MustNewStyle("kview", chroma.StyleEntries{
			chroma.Background:      "#E2E8F0 bg:#1B1B3A",
			chroma.Text:            "#E2E8F0",
			chroma.NameTag:         "#89B4FA",
			chroma.NameAttribute:   "#89B4FA",
			chroma.LiteralString:   "#10B981",
			chroma.LiteralNumber:   "#06B6D4",
			chroma.KeywordConstant: "#F59E0B",
			chroma.Comment:         "#64748B",
			chroma.Punctuation:     "#E2E8F0",
		}))
	})
}

// HighlightYAML applies chroma syntax highlighting to YAML content.
// Returns the original content unchanged on any error.
func HighlightYAML(content string) string {
	initHighlight()

	if yamlLexer == nil || termFormatter == nil || kviewStyle == nil {
		return content
	}

	tokens, err := yamlLexer.Tokenise(nil, content)
	if err != nil {
		return content
	}

	var buf bytes.Buffer
	if err := termFormatter.Format(&buf, kviewStyle, tokens); err != nil {
		return content
	}

	return strings.TrimRight(buf.String(), "\n")
}

// HighlightJSON applies chroma syntax highlighting to JSON content.
// Returns the original content unchanged on any error.
func HighlightJSON(content string) string {
	initHighlight()

	if jsonLexer == nil || termFormatter == nil || kviewStyle == nil {
		return content
	}

	tokens, err := jsonLexer.Tokenise(nil, content)
	if err != nil {
		return content
	}

	var buf bytes.Buffer
	if err := termFormatter.Format(&buf, kviewStyle, tokens); err != nil {
		return content
	}

	return strings.TrimRight(buf.String(), "\n")
}

// highlightLogLine applies JSON syntax highlighting to a log line if it looks like JSON.
// Only detects {...} objects (not [...] arrays — log lines are virtually never bare arrays).
func highlightLogLine(line string) string {
	trimmed := strings.TrimSpace(line)
	if len(trimmed) >= 2 && trimmed[0] == '{' && trimmed[len(trimmed)-1] == '}' {
		return HighlightJSON(line)
	}
	return line
}

// highlightSecretContent applies JSON highlighting to decoded secret values.
// Parses the GetSecretDecoded output format (── key ── section markers).
// JSON values (starting with { or [) get highlighted; other values pass through.
func highlightSecretContent(content string) string {
	lines := strings.Split(content, "\n")
	var result []string
	var valueLines []string
	inValue := false

	flush := func() {
		if len(valueLines) == 0 {
			return
		}
		joined := strings.Join(valueLines, "\n")
		trimmed := strings.TrimSpace(joined)
		if len(trimmed) > 0 && (trimmed[0] == '{' || trimmed[0] == '[') {
			result = append(result, HighlightJSON(joined))
		} else {
			result = append(result, joined)
		}
		valueLines = nil
	}

	for _, line := range lines {
		if strings.HasPrefix(line, "── ") && strings.HasSuffix(line, " ──") {
			flush()
			inValue = true
			result = append(result, line)
			continue
		}
		if inValue {
			valueLines = append(valueLines, line)
		} else {
			result = append(result, line)
		}
	}
	flush()

	return strings.Join(result, "\n")
}
