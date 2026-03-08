package views

import (
	"bytes"
	"fmt"
	"image/color"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"

	"github.com/bijaya/kview/internal/ui/theme"
)

// colorToHex converts a color.Color to a hex string (#RRGGBB).
func colorToHex(c color.Color) string {
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("#%02X%02X%02X", r>>8, g>>8, b>>8)
}

func init() {
	theme.OnComputeStyles(ResetHighlight)
}

var (
	highlightInitialized bool
	yamlLexer            chroma.Lexer
	jsonLexer            chroma.Lexer
	termFormatter        chroma.Formatter
	kviewStyle           *chroma.Style
)

// ResetHighlight clears the cached chroma style so it rebuilds from the
// current theme colors on next use. Call after theme.Apply()/ComputeStyles().
func ResetHighlight() {
	highlightInitialized = false
	kviewStyle = nil
}

func initHighlight() {
	if highlightInitialized {
		return
	}
	highlightInitialized = true

	yamlLexer = lexers.Get("yaml")
	if yamlLexer != nil {
		yamlLexer = chroma.Coalesce(yamlLexer)
	}

	jsonLexer = lexers.Get("json")
	if jsonLexer != nil {
		jsonLexer = chroma.Coalesce(jsonLexer)
	}

	termFormatter = formatters.Get("terminal16m")

	kviewStyle = chroma.MustNewStyle("kview", chroma.StyleEntries{
		chroma.Background:      colorToHex(theme.ColorText) + " bg:" + colorToHex(theme.ColorBackground),
		chroma.Text:            colorToHex(theme.ColorText),
		chroma.NameTag:         colorToHex(theme.ColorHighlight),
		chroma.NameAttribute:   colorToHex(theme.ColorHighlight),
		chroma.Literal:         colorToHex(theme.ColorSuccess),
		chroma.LiteralString:   colorToHex(theme.ColorSuccess),
		chroma.LiteralNumber:   colorToHex(theme.ColorAccent),
		chroma.KeywordConstant: colorToHex(theme.ColorWarning),
		chroma.Comment:         colorToHex(theme.ColorMuted),
		chroma.Punctuation:     colorToHex(theme.ColorText),
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
