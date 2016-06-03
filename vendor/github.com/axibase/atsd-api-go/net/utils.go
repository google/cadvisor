package net

import "strings"

func escapeField(text string) string {
	return strings.Replace(text, "\"", "\"\"", -1)
}
