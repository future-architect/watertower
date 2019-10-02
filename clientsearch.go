package watertower

type ResultPerWord struct {
	Word  string `json:"word"`
	Count int    `json:"count"`
}

func (c Client) Search(searchWord string, tags []string, lang string) ([]*Document, []ResultPerWord, error) {
	/*useTagFilter := len(tags) > 0
	useWordFilter := searchWord != ""
	var tagDocIds []uint32
	if useTagFilter {
		actions := c.tags.Actions()
		foundTags := make([]TagEntity, len(tags))
		for i, tag := range tags {
			foundTags[i].Tag = tag
			actions = actions.Get(&foundTags[i])
		}
		err := actions.Do(c.ctx)
		if err != nil {
			return nil, nil, err
		}
	}*/
	return nil, nil, nil

	/*tokenizer, err := nlp.FindTokenizer(lang)
	if err != nil {
		return nil, nil, fmt.Errorf("tokenizer for language '%s' is not found: %w", err)
	}
	searchTokens := tokenizer.Tokenize(searchWord)
	resultPerWord := make([]ResultPerWord, 0, len(searchTokens))
	foundTokens := make(map[string]*Token)
	for token, tokenInfo := range searchTokens {
		foundToken, err := c.FindTokens(token)
		if err != nil {
			resultPerWord = append(resultPerWord, ResultPerWord{
				Word: tokenInfo.BeforeStem,
				Count: 0,
			})
		} else {
			resultPerWord = append(resultPerWord, ResultPerWord{
				Word: tokenInfo.BeforeStem,
				Count: len(foundToken.Postings),
			})
			foundTokens[token] = foundToken
		}
	}

	panic("not implemented")*/
}
