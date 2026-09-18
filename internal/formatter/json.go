package formatter

import (
	"encoding/json"

	"github.com/m5rcel-vibecodes/why/internal/model"
)

// FormatJSON outputs the match result as indented JSON bytes.
func FormatJSON(res *model.MatchResult) ([]byte, error) {
	if res == nil {
		empty := map[string]any{
			"found":   false,
			"message": "No explanation found in local knowledge base",
		}
		return json.MarshalIndent(empty, "", "  ")
	}
	return json.MarshalIndent(res, "", "  ")
}
