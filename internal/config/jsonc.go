package config

import (
	"strings"
)

// StripJSONComments removes // and /* */ comments from JSONC content
func StripJSONComments(data []byte) []byte {
	input := string(data)
	var result strings.Builder
	result.Grow(len(input))

	i := 0
	inString := false
	for i < len(input) {
		// Track string state (to avoid stripping inside strings)
		if input[i] == '"' && (i == 0 || input[i-1] != '\\') {
			inString = !inString
			result.WriteByte(input[i])
			i++
			continue
		}

		// Only process comments when not inside a string
		if !inString {
			// Line comment //
			if i+1 < len(input) && input[i] == '/' && input[i+1] == '/' {
				// Skip to end of line
				for i < len(input) && input[i] != '\n' {
					i++
				}
				continue
			}

			// Block comment /* */
			if i+1 < len(input) && input[i] == '/' && input[i+1] == '*' {
				i += 2
				// Find closing */
				for i+1 < len(input) {
					if input[i] == '*' && input[i+1] == '/' {
						i += 2
						break
					}
					i++
				}
				continue
			}
		}

		result.WriteByte(input[i])
		i++
	}

	return []byte(result.String())
}
