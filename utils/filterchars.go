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

	// 冒頭のハイフンを全て削除
	for strings.HasPrefix(result, "-") {
		result = strings.TrimPrefix(result, "-")
	}
	// 末尾のハイフンを全て削除
	for strings.HasSuffix(result, "-") {
		result = strings.TrimSuffix(result, "-")
	}

	return strings.ToLower(result)
}
