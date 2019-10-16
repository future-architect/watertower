package watertower

import (
	"fmt"
	"github.com/shibukawa/watertower/nlp"
	"golang.org/x/sync/errgroup"
	"math"
	"sort"
)

func (wt WaterTower) Search(searchWord string, tags []string, lang string) ([]*Document, error) {
	tokenizer, err := nlp.FindTokenizer(lang)
	if err != nil {
		return nil, fmt.Errorf("tokenizer for language '%s' is not found: %w", lang, err)
	}
	searchTokens, _ := tokenizer.TokenizeToMap(searchWord)

	if len(searchTokens) == 0 && len(tags) == 0 {
		return nil, nil
	}

	errGroup, ctx := errgroup.WithContext(wt.ctx)
	var tagDocIDGroups [][]uint32

	if len(tags) > 0 {
		errGroup.Go(func() error {
			findTags, err := wt.FindTagsWithContext(ctx, tags...)
			if err != nil {
				return err
			}
			tagDocIDGroups = make([][]uint32, len(findTags))
			for i, findTag := range findTags {
				tagDocIDGroups[i] = findTag.DocumentIDs
			}
			return nil
		})
	}

	var foundTokens []*Token
	var tokenDocIDGroups [][]uint32
	var docCount int

	errGroup.Go(func() (err error) {
		docCount, err = wt.counter.Get(ctx, documentCount)
		return
	})

	if len(searchTokens) > 0 {
		errGroup.Go(func() (err error) {
			tokens := make([]string, 0, len(searchTokens))
			for token := range searchTokens {
				tokens = append(tokens, token)
			}
			foundTokens, err = wt.FindTokensWithContext(ctx, tokens...)
			for _, token := range foundTokens {
				docIDs := make([]uint32, len(token.Postings))
				for i, posting := range token.Postings {
					docIDs[i] = posting.DocumentID
				}
				tokenDocIDGroups = append(tokenDocIDGroups, docIDs)
			}
			return
		})
	}

	err = errGroup.Wait()
	if err != nil {
		return nil, err
	}

	var docIDs []uint32
	if len(tags) > 0 && len(searchTokens) > 0 {
		docIDGroups := append(tagDocIDGroups, tokenDocIDGroups...)
		docIDs = intersection(docIDGroups...)
	} else if len(searchTokens) > 0 {
		docIDs = intersection(tokenDocIDGroups...)
	} else {
		// len(tags) > 0
		docIDs = intersection(tagDocIDGroups...)
	}

	if len(searchTokens) > 0 {
		docIDs, _ = phraseSearchFilter(docIDs, searchTokens, foundTokens)
	}

	docs, err := wt.FindDocuments(docIDs...)
	if err != nil {
		return nil, err
	}
	for i, doc := range docs {
		doc.Score = calcScore(foundTokens, docCount, docIDs[i])
	}
	sort.Slice(docs, func(i, j int) bool {
		return docs[i].Score < docs[j].Score
	})
	return docs, nil
}

func phraseSearchFilter(docIDs []uint32, searchTokens map[string]*nlp.Token, foundTokens []*Token) (matchedDocIDs []uint32, foundPositions [][]uint32) {
	for _, docID := range docIDs {
		tokenPositionMap := convertToTokenPositionMap(foundTokens, docID)
		var relativePositionGroups [][]uint32
		for word, positionMap := range tokenPositionMap {
			relativePositions := findPhraseMatchPositions(searchTokens[word], positionMap)
			relativePositionGroups = append(relativePositionGroups, relativePositions)
		}
		relativePositions := intersection(relativePositionGroups...)
		if len(relativePositions) > 0 {
			matchedDocIDs = append(matchedDocIDs, docID)
			foundPositions = append(foundPositions, relativePositions)
		}
	}
	return
}

func findPhraseMatchPositions(token *nlp.Token, positionMap map[uint32]bool) []uint32 {
	firstPos := token.Positions[0]
	var result []uint32
	for position := range positionMap {
		match := true
		for i := 1; i < len(token.Positions); i++ {
			otherPos := token.Positions[i]
			if !positionMap[position-firstPos+otherPos] {
				match = false
				break
			}
		}
		if match {
			result = append(result, position-firstPos)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i] < result[j]
	})
	return result
}

func convertToTokenPositionMap(foundTokens []*Token, docID uint32) map[string]map[uint32]bool {
	foundTokenMaps := make(map[string]map[uint32]bool)
	for _, foundToken := range foundTokens {
		positionMap := make(map[uint32]bool)
		for _, posting := range foundToken.Postings {
			if posting.DocumentID == docID {
				for _, pos := range posting.Positions {
					positionMap[pos] = true
				}
				break
			}
		}
		foundTokenMaps[foundToken.Word] = positionMap
	}
	return foundTokenMaps
}

func calcScore(foundTokens []*Token, docCount int, documentID uint32) float64 {
	var totalScore float64
	for _, token := range foundTokens {
		for _, posting := range token.Postings {
			if posting.DocumentID == documentID {
				totalScore += tfIdfScore(len(posting.Positions), docCount, len(token.Postings))
			}
		}
	}
	return totalScore
}

func tfIdfScore(tokenCount, allDocCount, docCount int) float64 {
	var tf float64
	if tokenCount > 0 {
		tf = 1.0 + math.Log(float64(tokenCount))
	}
	idf := math.Log(float64(allDocCount) / float64(docCount))
	return tf * idf
}
