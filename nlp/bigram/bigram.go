package bigram

import (
	"github.com/future-architect/watertower/nlp"
	"strings"
)

const Language = "bigram"

func init() {
	stopWords := make(map[string]bool)
	nlp.RegisterTokenizer(Language, bigramSplitter, nil, stopWords)
}

func bigramSplitter(content string) []string {
	if len(content) < 2 {
		return []string{}
	}
	chars := strings.Split(content, "")
	result := make([]string, len(chars)-1)
	for i := 0; i < len(chars) - 1; i++ {
		result[i] = chars[i] + chars[i+1]
	}
	return result
}
