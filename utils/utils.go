package utils

import (
	"regexp"
	"strings"
)

// SanitizeURLTitle converts a title to a URL-friendly string.
func SanitizeURLTitle(title string) string {
	lower := strings.ToLower(title)
	re := regexp.MustCompile(`[^a-z0-9]+`)
	replaced := re.ReplaceAllString(lower, "-")
	return strings.Trim(replaced, "-")
}

// SanitizeFilename creates a safe filename from the title.
func SanitizeFilename(title string) string {
	title = strings.ReplaceAll(title, " ", "_")
	re := regexp.MustCompile(`[^a-zA-Z0-9_-]`)
	return re.ReplaceAllString(title, "")
}
