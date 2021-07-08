package utils

import (
	"unicode/utf8"
)

func ValidateText(text string) bool {
	count := utf8.RuneCountInString(text)

	if count > 20 {
		return false
	}

	return true
}