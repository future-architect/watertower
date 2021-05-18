package unigram

import (
	"github.com/future-architect/watertower/nlp"
	"strings"
)

const Language = "unigram"

func init() {
	stopWords := make(map[string]bool)
	nlp.RegisterTokenizer(Language, unigramSplitter, nil, stopWords)
}

func unigramSplitter(content string) []string {
	return strings.Split(content, "")
}
