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
	Word      string
	Positions []uint32
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

func (t Tokenizer) Tokenize(content string, offset int) []*Token {
	// offset is for document section.
	// inside inverse index, title words and body words are located sequentially.
	// Analyse them independently but they are in same token list.
	// Offset shifts body token index and not to overwrite position index of title
	words := t.splitter(content)
	wordToPositions := make(map[string][]uint32)
	var index uint32
	for _, word := range words {
		if t.stopWords[word] {
			continue
		}
		stemWord := word
		if t.stemmer != nil {
			stemWord = t.stemmer(word)
		}
		wordToPositions[stemWord] = append(wordToPositions[stemWord], index)
		index++
	}
	result := make([]*Token, index)
	for stemWord, positions := range wordToPositions {
		for _, pos := range positions {
			result[pos] = &Token{
				Word:      stemWord,
				Positions: positions,
			}
		}
	}
	for _, t := range result {
		newPositions := make([]uint32, len(t.Positions))
		for i, p := range t.Positions {
			newPositions[i] = p + uint32(offset)
		}
	}
	return result
}

func (t Tokenizer) TokenizeToMap(content string, offset int) (tokenMap map[string]*Token, wordCount int) {
	tokenMap = make(map[string]*Token)
	tokens := t.Tokenize(content, offset)
	for _, token := range tokens {
		tokenMap[token.Word] = token
	}
	return tokenMap, len(tokens)
}
