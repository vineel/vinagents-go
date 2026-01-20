package worker

// Helper functions shared between clauser workers

// ptr returns a pointer to the given value
func ptr[T any](v T) *T {
	return &v
}

// deref returns the value of a string pointer, or empty string if nil
func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// extractJSON attempts to extract JSON from text that might have markdown code blocks
func extractJSON(text string) string {
	// Look for JSON code block
	start := 0
	if idx := findIndex(text, "```json"); idx >= 0 {
		start = idx + 7
	} else if idx := findIndex(text, "```"); idx >= 0 {
		start = idx + 3
	}

	end := len(text)
	if start > 0 {
		if idx := findIndex(text[start:], "```"); idx >= 0 {
			end = start + idx
		}
	}

	result := text[start:end]

	// Find the first { and last }
	firstBrace := findIndex(result, "{")
	lastBrace := findLastIndex(result, "}")

	if firstBrace >= 0 && lastBrace > firstBrace {
		return result[firstBrace : lastBrace+1]
	}

	return result
}

func findIndex(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func findLastIndex(s, substr string) int {
	for i := len(s) - len(substr); i >= 0; i-- {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// truncate shortens a string to maxLen characters, adding "..." if truncated
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
