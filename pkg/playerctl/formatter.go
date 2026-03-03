package playerctl

import (
	"bytes"
	"fmt"
	"html"
	"strings"
	"text/template"
	"time"
)

// Formatter expands text/template expressions using a string context.
type Formatter struct {
	raw  string
	tmpl *template.Template
}

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

// NewFormatter constructs a formatter for a template string.
func NewFormatter(format string) (*Formatter, error) {
	if strings.TrimSpace(format) == "" {
		return nil, FormatError{Message: "empty format template"}
	}

	tmpl, err := template.New("playerctl-format").
		Funcs(template.FuncMap{
			"lc":  strings.ToLower,
			"uc":  strings.ToUpper,
			"add": func(a, b int) int { return a + b },
			"sub": func(a, b int) int { return a - b },
			"default": func(v, fallback string) string {
				if v == "" {
					return fallback
				}
				return v
			},
			"duration":      helperDuration,
			"markup_escape": html.EscapeString,
			"emoji":         helperEmoji,
			"trunc":         helperTrunc,
		}).
		Option("missingkey=zero").
		Parse(format)
	if err != nil {
		return nil, FormatError{Message: err.Error()}
	}

	return &Formatter{raw: format, tmpl: tmpl}, nil
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
func (f *Formatter) Expand(context map[string]string) (string, error) {
	var b bytes.Buffer
	if err := f.tmpl.Execute(&b, context); err != nil {
		return "", FormatError{Message: fmt.Sprintf("template execution failed: %v", err)}
	}
	return b.String(), nil
}
