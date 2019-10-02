package nlp

import (
	"fmt"
)

const (
	LanguageJapanese = "ja"
	LanguageEnglish  = "en"
)

var languages = make(map[string]*Tokenizer)

type Token struct {
	Word       string
	BeforeStem string
	Positions  []uint32
}

func FindTokenizer(lang string) (*Tokenizer, error) {
	tokenizer, ok := languages[lang]
	if !ok {
		return nil, fmt.Errorf("can't find tokenizer for %s", lang)
	}
	return tokenizer, nil
}

type Tokenizer struct {
	splitter    func(string) []string
	stemmer     func(string) string
	stopWords   map[string]bool
	properNouns map[string]string
}

func RegisterTokenizer(lang string, splitter func(string) []string, stemmer func(string) string, stopWords map[string]bool) {
	languages[lang] = &Tokenizer{
		splitter:    splitter,
		stemmer:     stemmer,
		stopWords:   stopWords,
		properNouns: make(map[string]string),
	}
}

func (t Tokenizer) StemWord(word string) string {
	return t.stemmer(word)
}

func (t Tokenizer) Tokenize(content string) map[string]*Token {
	words := t.splitter(content)
	tokens := make(map[string]*Token)
	var position uint32
	for _, word := range words {
		if t.stopWords[word] {
			continue
		}
		stemWord := t.stemmer(word)
		if token, ok := tokens[stemWord]; ok {
			token.Positions = append(token.Positions, position)
		} else {
			token := &Token{
				Word:       stemWord,
				BeforeStem: word,
				Positions:  []uint32{position},
			}
			tokens[stemWord] = token
		}
		position++
	}
	return tokens
}
