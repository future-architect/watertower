package nlp_test

import (
	"testing"

	"github.com/shibukawa/watertower/nlp"
	_ "github.com/shibukawa/watertower/nlp/english"
	_ "github.com/shibukawa/watertower/nlp/japanese"
	"github.com/stretchr/testify/assert"
)

func TestEnglish(t *testing.T) {
	tokenizer, err := nlp.FindTokenizer(nlp.LanguageEnglish)
	assert.Nil(t, err)
	tokens := tokenizer.Tokenize("Go is an open source programming language that makes it easy to build simple, reliable, and efficient software.")
	/*for _, token := range tokens {
		t.Log(token.Word)
	}*/
	_, ok := tokens[tokenizer.StemWord("programming")]
	assert.True(t, ok)

	// "a" is a stor word
	_, ok = tokens[tokenizer.StemWord("a")]
	assert.False(t, ok)
}

func TestJapanese(t *testing.T) {
	tokenizer, err := nlp.FindTokenizer(nlp.LanguageJapanese)
	assert.Nil(t, err)
	// https://medium.com/@torufurukawa/%E3%83%89%E3%83%AA%E3%83%AB%E3%81%A7%E3%81%AF%E3%81%AA%E3%81%8F%E7%A9%B4%E3%81%8C%E6%AC%B2%E3%81%97%E3%81%84-%E7%A9%B4%E3%81%8C%E5%BF%85%E8%A6%81%E3%81%AA%E3%82%B7%E3%83%81%E3%83%A5%E3%82%A8%E3%83%BC%E3%82%B7%E3%83%A7%E3%83%B3%E3%81%AF-127c65d1b78b
	tokens := tokenizer.Tokenize("「顧客はドリルではなく穴が欲しい」とよく言われる。もう一歩進んで穴が必要なシチュエーションも考えてみましょう、と。そうすると穴が必要であることを自覚していない人を、ドリルの顧客にできるかも知れない。")
	for _, token := range tokens {
		t.Log(token.Word)
	}
	_, ok := tokens[tokenizer.StemWord("ドリル")]
	assert.True(t, ok)

	// "a" is a stor word
	_, ok = tokens[tokenizer.StemWord("は")]
	assert.False(t, ok)
}

