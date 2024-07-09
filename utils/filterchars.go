package utils

import (
	"regexp"
	"strings"
)

func FilterAndReplaceSymbols(input string) string {
	allowedPattern := `[\p{Hiragana}～ー\p{Katakana}\p{Han}\p{Latin}\d]`
	symbolPattern := `[ !"#$%&'()*+,\-./:;<=>?@[\\\]^_` + "`" + `{|}~ ]+`

	allowedRe := regexp.MustCompile(allowedPattern)
	symbolRe := regexp.MustCompile(symbolPattern)

	replaced := symbolRe.ReplaceAllString(input, "-")

	result := ""
	for _, char := range replaced {
		strChar := string(char)
		if allowedRe.MatchString(strChar) || strChar == "-" {
			result += strChar
		}
	}
	return strings.ToLower(result)
}
