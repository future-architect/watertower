package watertower

import (
	"context"
	"fmt"
	"github.com/shibukawa/compints"
	"github.com/shibukawa/watertower/nlp"
	"gocloud.dev/docstore"
	"reflect"
	"sort"
)

func (wt WaterTower) PostDocument(uniqueKey string, document *Document) (uint32, error) {
	retryCount := 50
	var lastError error
	var docID uint32
	newTags, newDocTokens, wordCount, err := wt.analyzeDocument("new", document)
	if err != nil {
		return 0, err
	}
	for i := 0; i < retryCount; i++ {
		docID, lastError = wt.postDocumentKey(uniqueKey)
		if lastError == nil {
			break
		}
	}
	if lastError != nil {
		return 0, fmt.Errorf("fail to register document's unique key: %w", lastError)
	}
	for i := 0; i < retryCount; i++ {
		oldDoc, err := wt.postDocument(docID, uniqueKey, wordCount, document)
		if err != nil {
			lastError = err
			continue
		}
		oldTags, oldDocTokens, _, err := wt.analyzeDocument("old", oldDoc)
		if err != nil {
			return 0, err
		}
		err = wt.updateTagsAndTokens(docID, oldTags, newTags, oldDocTokens, newDocTokens)
		if err != nil {
			lastError = err
			continue
		}
		return docID, nil
	}
	return 0, fmt.Errorf("fail to register document: %w", lastError)
}

func (wt WaterTower) postDocumentKey(uniqueKey string) (uint32, error) {
	existingDocKey := DocumentKey{
		UniqueKey: uniqueKey,
	}
	err := wt.docKeys.Get(wt.ctx, &existingDocKey)
	if err == nil {
		return existingDocKey.ID, nil
	}
	newID, err := wt.incrementID()
	if err != nil {
		return 0, err
	}
	err = wt.docKeys.Create(wt.ctx, &DocumentKey{
		UniqueKey: uniqueKey,
		ID:        newID,
	})
	if err != nil {
		return 0, err
	}
	return newID, nil
}

func (wt WaterTower) postDocument(docID uint32, uniqueKey string, wordCount int, document *Document) (*Document, error) {
	existingDocument := Document{
		ID: docID,
	}
	document.ID = docID
	document.UniqueKey = uniqueKey
	document.WordCount = wordCount
	err := wt.documents.Get(wt.ctx, &existingDocument)
	if err != nil {
		return nil, wt.documents.Create(wt.ctx, document)
	} else {
		return &existingDocument, wt.documents.Replace(wt.ctx, document)
	}
}

func (wt WaterTower) incrementID() (uint32, error) {
	uniqueID := UniqueID{
		Collection: "documents",
	}
	latestUniqueID := UniqueID{
		Collection: "documents",
	}
	err := wt.uniqueIDs.Actions().
		Update(&uniqueID, docstore.Mods{"latestID": docstore.Increment(1)}).
		Get(&latestUniqueID).
		Do(wt.ctx)
	if err != nil {
		return 0, err
	}
	return latestUniqueID.LatestID, nil
}

func (wt WaterTower) RemoveDocument(uniqueKey string) error {
	docID, existingDocKey, oldDoc, err := wt.findDocumentByKey(uniqueKey)
	if err != nil {
		return err
	}
	err = wt.docKeys.Delete(wt.ctx, existingDocKey)
	if err != nil {
		return err
	}
	err = wt.documents.Delete(wt.ctx, oldDoc)
	if err != nil {
		return err
	}
	tags, tokens, _, err := wt.analyzeDocument("removed", oldDoc)
	if err != nil {
		return err
	}
	return wt.updateTagsAndTokens(docID, tags, nil, tokens, nil)
}

func (wt WaterTower) analyzeDocument(label string, document *Document) (tags []string, tokens map[string]*nlp.Token, wordCount int, err error) {
	if document == nil {
		return nil, make(map[string]*nlp.Token), 0, nil
	}
	tokenizer, err := nlp.FindTokenizer(document.Language)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("Cannot find tokenizer for %s document: lang=%s, err=%w", label, document.Language, err)
	}
	tokens, wordCount = tokenizer.TokenizeToMap(document.Content)
	return document.Tags, tokens, wordCount, nil
}

func (wt WaterTower) updateTagsAndTokens(docID uint32, oldTags, newTags []string, oldDocTokens, newDocTokens map[string]*nlp.Token) error {
	// update tags
	newTags, deletedTags := groupingTags(oldTags, newTags)
	for _, tag := range newTags {
		err := wt.AddDocumentToTag(tag, docID)
		if err != nil {
			return err
		}
	}
	for _, tag := range deletedTags {
		err := wt.RemoveDocumentFromTag(tag, docID)
		if err != nil {
			return err
		}
	}

	// update tokens
	newTokens, deletedTokens, updateTokens := groupingTokens(oldDocTokens, newDocTokens)
	for _, token := range newTokens {
		err := wt.AddDocumentToToken(token.Word, docID, token.Positions)
		if err != nil {
			return err
		}
	}
	for _, token := range deletedTokens {
		err := wt.RemoveDocumentFromToken(token.Word, docID)
		if err != nil {
			return err
		}
	}
	for _, token := range updateTokens {
		err := wt.AddDocumentToToken(token.Word, docID, token.Positions)
		if err != nil {
			return err
		}
	}
	return nil
}

func groupingTags(oldGroup, newGroup []string) (newItems, deletedItems []string) {
	oldMap := make(map[string]bool)
	for _, item := range oldGroup {
		oldMap[item] = true
	}
	newMap := make(map[string]bool)
	for _, item := range newGroup {
		newMap[item] = true
		if !oldMap[item] {
			newItems = append(newItems, item)
		}
	}
	for _, item := range oldGroup {
		if !newMap[item] {
			deletedItems = append(deletedItems, item)
		}
	}
	return
}

func groupingTokens(oldGroup, newGroup map[string]*nlp.Token) (newItems, deletedItems, updateItems []*nlp.Token) {
	for key, newToken := range newGroup {
		if oldToken, ok := oldGroup[key]; ok {
			// skip if completely match
			if !reflect.DeepEqual(newToken.Positions, oldToken.Positions) {
				updateItems = append(updateItems, newToken)
			}
		} else {
			newItems = append(newItems, newToken)
		}
	}
	for key, oldToken := range oldGroup {
		if _, ok := newGroup[key]; !ok {
			deletedItems = append(deletedItems, oldToken)
		}
	}
	return
}

func (wt WaterTower) AddDocumentToTag(tag string, docID uint32) error {
	retryCount := 50
	var lastError error
	for i := 0; i < retryCount; i++ {
		err := wt.addDocumentToTag(tag, docID)
		if err != nil {
			lastError = err
			continue
		}
		return nil
	}
	return fmt.Errorf("fail to update tag: %w", lastError)
}

func (wt WaterTower) addDocumentToTag(tag string, docID uint32) error {
	existingTag := TagEntity{
		Tag: tag,
	}
	err := wt.tags.Get(wt.ctx, &existingTag)
	if err != nil {
		tag := TagEntity{
			Tag:         tag,
			DocumentIDs: compints.CompressToBytes([]uint32{docID}, true),
		}
		return wt.tags.Create(wt.ctx, &tag)
	} else {
		docIDs, err := compints.DecompressFromBytes(existingTag.DocumentIDs, true)
		if err != nil {
			return fmt.Errorf("fail to decompress document IDs of tag '%s': %w", tag, err)
		}
		docIDs = append(docIDs, docID)
		sort.Slice(docIDs, func(i, j int) bool {
			return docIDs[i] < docIDs[j]
		})
		newTag := &TagEntity{
			Tag:         tag,
			DocumentIDs: compints.CompressToBytes(docIDs, true),
		}
		err = wt.tags.Replace(wt.ctx, newTag)
		if err != nil {
			return fmt.Errorf("fail to replace tag: '%s': %w", tag, err)
		}
		return nil
	}
}

func (wt WaterTower) RemoveDocumentFromTag(tag string, docID uint32) error {
	retryCount := 50
	var lastError error
	for i := 0; i < retryCount; i++ {
		err := wt.removeDocumentFromTag(tag, docID)
		if err != nil {
			lastError = err
			continue
		}
		return nil
	}
	return fmt.Errorf("fail to update tag: %w", lastError)
}

func (wt WaterTower) removeDocumentFromTag(tag string, docID uint32) error {
	existingTag := TagEntity{
		Tag: tag,
	}
	err := wt.tags.Get(wt.ctx, &existingTag)
	if err != nil {
		return err
	}
	existingDocIDs, err := compints.DecompressFromBytes(existingTag.DocumentIDs, true)
	if err != nil {
		return err
	}
	newDocIDs := make([]uint32, 0, len(existingDocIDs)-1)
	for _, existingDocID := range existingDocIDs {
		if existingDocID != docID {
			newDocIDs = append(newDocIDs, existingDocID)
		}
	}
	if len(newDocIDs) == 0 {
		return wt.tags.Delete(wt.ctx, &existingTag)
	} else {
		newTag := &TagEntity{
			Tag:         tag,
			DocumentIDs: compints.CompressToBytes(newDocIDs, true),
		}
		existingTag.DocumentIDs = compints.CompressToBytes(newDocIDs, true)
		return wt.tags.Replace(wt.ctx, newTag)
	}
}

func (wt WaterTower) AddDocumentToToken(word string, docID uint32, positions []uint32) error {
	retryCount := 50
	var lastError error
	for i := 0; i < retryCount; i++ {
		err := wt.addDocumentToToken(word, docID, positions)
		if err != nil {
			lastError = err
			continue
		}
		return nil
	}
	return fmt.Errorf("fail to update tag: %w", lastError)
}

func (wt WaterTower) addDocumentToToken(word string, docID uint32, positions []uint32) error {
	existingToken := TokenEntity{
		Word: word,
	}
	err := wt.tokens.Get(wt.ctx, &existingToken)
	postingEntity := PostingEntity{
		DocumentID: docID,
		Positions:  compints.CompressToBytes(positions, true),
	}
	if err != nil {
		token := TokenEntity{
			Word:     word,
			Postings: []PostingEntity{postingEntity},
		}
		return wt.tokens.Create(wt.ctx, &token)
	} else {
		newToken := TokenEntity{
			Word:     word,
			Postings: append(existingToken.Postings, postingEntity),
		}
		sort.Slice(newToken.Postings, func(i, j int) bool {
			return newToken.Postings[i].DocumentID < newToken.Postings[j].DocumentID
		})
		err = wt.tokens.Replace(wt.ctx, &newToken)
		if err != nil {
			return fmt.Errorf("fail to replace token: '%s': %w", word, err)
		}
		return nil
	}
}

func (wt WaterTower) RemoveDocumentFromToken(word string, docID uint32) error {
	retryCount := 50
	var lastError error
	for i := 0; i < retryCount; i++ {
		err := wt.removeDocumentFromToken(word, docID)
		if err != nil {
			lastError = err
			continue
		}
		return nil
	}
	return fmt.Errorf("fail to update tag: %w", lastError)
}

func (wt WaterTower) removeDocumentFromToken(word string, docID uint32) error {
	existingToken := TokenEntity{
		Word: word,
	}
	err := wt.tokens.Get(wt.ctx, &existingToken)
	if err != nil {
		return err
	} else {
		newPostings := make([]PostingEntity, 0, len(existingToken.Postings)-1)
		for _, existingPosting := range existingToken.Postings {
			if existingPosting.DocumentID != docID {
				newPostings = append(newPostings, existingPosting)
			}
		}
		if len(newPostings) == 0 {
			return wt.tokens.Delete(wt.ctx, &existingToken)
		} else {
			newToken := TokenEntity{
				Word:     word,
				Postings: newPostings,
			}
			return wt.tokens.Replace(wt.ctx, &newToken)
		}
	}
}

func (wt WaterTower) FindTags(tagNames ...string) ([]*Tag, error) {
	return wt.FindTagsWithContext(wt.ctx, tagNames...)
}

func (wt WaterTower) FindTagsWithContext(ctx context.Context, tagNames ...string) ([]*Tag, error) {
	if len(tagNames) == 0 {
		return nil, nil
	}
	existingTags := make([]TagEntity, len(tagNames))
	actions := wt.tags.Actions()
	for i, tagName := range tagNames {
		existingTags[i].Tag = tagName
		actions = actions.Get(&existingTags[i])
	}
	err := actions.Do(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*Tag, len(tagNames))
	for i, existingTag := range existingTags {
		docIDs, err := compints.DecompressFromBytes(existingTag.DocumentIDs, true)
		if err != nil {
			return nil, err
		}
		result[i] = &Tag{
			Tag:         existingTag.Tag,
			DocumentIDs: docIDs,
		}
	}
	return result, nil
}

func (wt WaterTower) FindTokens(words ...string) ([]*Token, error) {
	return wt.FindTokensWithContext(wt.ctx, words...)
}

func (wt WaterTower) FindTokensWithContext(ctx context.Context, words ...string) ([]*Token, error) {
	if len(words) == 0 {
		return nil, nil
	}
	positions := make(map[string][]int)
	for i, word := range words {
		positions[word] = append(positions[word], i)
	}
	existingTokens := make([]TokenEntity, len(positions))
	actions := wt.tokens.Actions()
	for i, word := range words {
		existingTokens[i].Word = word
		actions = actions.Get(&existingTokens[i])
	}
	hasErrors := make(map[int]bool)
	if errs, ok := actions.Do(ctx).(docstore.ActionListError); ok {
		for _, err := range errs {
			hasErrors[err.Index] = true
		}
	}
	result := make([]*Token, len(words))
	for i, existingToken := range existingTokens {
		token := &Token{
			Word:  existingToken.Word,
			Found: !hasErrors[i],
		}
		for _, posting := range existingToken.Postings {
			positions, err := compints.DecompressFromBytes(posting.Positions, true)
			if err != nil {
				return nil, fmt.Errorf("Compressed data is broken of position of doc %d of token %s: %w", posting.DocumentID, existingToken.Word, err)
			}
			token.Postings = append(token.Postings, Posting{
				DocumentID: posting.DocumentID,
				Positions:  positions,
			})
		}
		for _, pos := range positions[existingToken.Word] {
			result[pos] = token
		}
	}
	return result, nil
}

func (wt WaterTower) FindDocuments(ids ...uint32) ([]*Document, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	result := make([]*Document, len(ids))
	actions := wt.documents.Actions()
	for i, id := range ids {
		result[i] = &Document{
			ID: id,
		}
		actions = actions.Get(result[i])
	}
	err := actions.Do(wt.ctx)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (wt WaterTower) FindDocumentByKey(uniqueKey string) (*Document, error) {
	_, _, doc, err := wt.findDocumentByKey(uniqueKey)
	return doc, err
}

func (wt WaterTower) findDocumentByKey(uniqueKey string) (uint32, *DocumentKey, *Document, error) {
	existingDocKey := DocumentKey{
		UniqueKey: uniqueKey,
	}
	err := wt.docKeys.Get(wt.ctx, &existingDocKey)
	if err != nil {
		return 0, nil, nil, err
	}
	docID := existingDocKey.ID
	oldDoc := Document{
		ID: docID,
	}
	err = wt.documents.Get(wt.ctx, &oldDoc)
	if err != nil {
		return 0, nil, nil, err
	}
	return docID, &existingDocKey, &oldDoc, nil
}
