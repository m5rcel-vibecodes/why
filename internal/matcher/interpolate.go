package matcher

import (
	"bytes"
	"strings"
	"text/template"
)

// interpolateTemplate safely renders a template string with extracted params.
// If rendering fails or template syntax is absent, returns original text.
func interpolateTemplate(tmplStr string, params map[string]string) string {
	if !strings.Contains(tmplStr, "{{") {
		// Also support simple <key> fallback replacement if key is in params
		result := tmplStr
		for k, v := range params {
			if v != "" {
				result = strings.ReplaceAll(result, "<"+k+">", v)
			}
		}
		return result
	}

	tmpl, err := template.New("sub").Option("missingkey=zero").Parse(tmplStr)
	if err != nil {
		return tmplStr
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, params); err != nil {
		return tmplStr
	}

	rendered := buf.String()
	// Clean up any remaining <key> tags if the param is available
	for k, v := range params {
		if v != "" {
			rendered = strings.ReplaceAll(rendered, "<"+k+">", v)
		}
	}

	return rendered
}
