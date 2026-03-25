package playerctl

import (
	"bytes"
	"fmt"
	"html"
	"regexp"
	"strings"
	"text/template"
	"time"
)

// Formatter expands text/template expressions using a string context.
type Formatter struct {
	raw  string
	tmpl *template.Template
}

var (
	formatWordRe = regexp.MustCompile(`[a-zA-Z_][a-zA-Z0-9_]*`)
	formatExprRe = regexp.MustCompile(`(?s)\{\{(.*?)\}\}`)
	formatIdentRe = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

	predefinedTemplateFuncs = map[string]bool{
		"lc": true, "uc": true, "add": true, "sub": true, "default": true,
		"duration": true, "markup_escape": true, "emoji": true, "trunc": true,
		"has_playlist": true, "has_tracklist": true,
		"if": true, "else": true, "end": true, "range": true, "with": true,
		"and": true, "or": true, "not": true, "len": true, "index": true,
		"true": true, "false": true, "nil": true,
		"eq": true, "ne": true, "lt": true, "le": true, "gt": true, "ge": true,
		"print": true, "printf": true, "println": true,
		"call": true, "html": true, "js": true, "urlquery": true,
	}
)

func helperDuration(input string) string {
	if input == "" {
		return ""
	}
	if n, err := time.ParseDuration(input); err == nil {
		total := int64(n.Seconds())
		if total < 0 {
			total = -total
		}
		h := total / 3600
		m := (total % 3600) / 60
		s := total % 60
		if h > 0 {
			return fmt.Sprintf("%d:%02d:%02d", h, m, s)
		}
		return fmt.Sprintf("%d:%02d", m, s)
	}
	return input
}

func helperEmoji(status string) string {
	switch strings.ToLower(status) {
	case "playing":
		return "▶️"
	case "paused":
		return "⏸️"
	case "stopped":
		return "⏹️"
	default:
		return ""
	}
}

func helperTrunc(v string, max int) string {
	if max <= 0 || len(v) <= max {
		return v
	}
	if max <= 1 {
		return "…"
	}
	return v[:max-1] + "…"
}

// isPredefined checks if a given word is a Go template built-in function or keyword.
func isPredefined(word string) bool {
	return predefinedTemplateFuncs[word]
}

// NewFormatter constructs a formatter for a template string.
func NewFormatter(format string) (*Formatter, error) {
	if strings.TrimSpace(format) == "" {
		return nil, FormatError{Message: "empty format template"}
	}

	// text/template variables cannot contain a colon (e.g. .mpris:artUrl).
	// We replace ":" with "_" to match the context variables.
	processedFormat := strings.ReplaceAll(format, ".mpris:", ".mpris_")
	processedFormat = strings.ReplaceAll(processedFormat, ".xesam:", ".xesam_")

	funcs := template.FuncMap{
		"lc":  strings.ToLower,
		"uc":  strings.ToUpper,
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
		"default": func(v any, fallback string) string {
			if v == nil {
				return fallback
			}
			str, ok := v.(string)
			if !ok || str == "" {
				return fallback
			}
			return str
		},
		"duration":      helperDuration,
		"markup_escape": html.EscapeString,
		"emoji":         helperEmoji,
		"trunc":         helperTrunc,
		"has_playlist":  func(count string) bool { return count != "" && count != "0" },
		"has_tracklist": func(count string) bool { return count != "" && count != "0" },
	}

	// Pre-define all words in the format as dummy functions so that template.Parse succeeds.
	for _, match := range formatExprRe.FindAllString(processedFormat, -1) {
		for _, word := range formatWordRe.FindAllString(match, -1) {
			if isPredefined(word) {
				continue
			}
			if _, ok := funcs[word]; !ok {
				funcs[word] = func() string { return "" }
			}
		}
	}

	tmpl, err := template.New("playerctl-format").
		Funcs(funcs).
		Option("missingkey=zero").
		Parse(processedFormat)
	if err != nil {
		return nil, FormatError{Message: err.Error()}
	}

	return &Formatter{raw: processedFormat, tmpl: tmpl}, nil
}

// ContainsKey reports whether the template references the provided key.
func (f *Formatter) ContainsKey(key string) bool {
	if key == "" {
		return false
	}
	needle := "." + key
	return strings.Contains(f.raw, needle)
}

// Expand substitutes template variables using values from context.
func (f *Formatter) Expand(context map[string]any) (string, error) {
	clone, err := f.tmpl.Clone()
	if err != nil {
		return "", FormatError{Message: fmt.Sprintf("failed to clone template: %v", err)}
	}

	expandFuncs := template.FuncMap{}
	for key, val := range context {
		if !isPredefined(key) && formatIdentRe.MatchString(key) {
			v := val
			expandFuncs[key] = func() any { return v }
		}
	}

	clone.Funcs(expandFuncs)

	var b bytes.Buffer
	if err := clone.Execute(&b, context); err != nil {
		return "", FormatError{Message: fmt.Sprintf("template execution failed: %v", err)}
	}
	return b.String(), nil
}
