package watertower

import (
	"fmt"
	"github.com/shibukawa/watertower/nlp"
	"golang.org/x/sync/errgroup"
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

	docIDMap := make(map[uint32]bool)
	if len(tags) > 0 && len(searchTokens) > 0 {
		docIDGroups := append(tagDocIDGroups, tokenDocIDGroups...)
		docIDMap = intersection(docIDGroups...)
	} else if len(searchTokens) > 0 {
		docIDMap = intersection(tokenDocIDGroups...)
	} else {
		// len(tags) > 0
		docIDMap = intersection(tagDocIDGroups...)
	}

	docIDs := make([]uint32, 0, len(docIDMap))
	for docID := range docIDMap {
		docIDs = append(docIDs, docID)
	}

	return wt.FindDocuments(docIDs...)
}
