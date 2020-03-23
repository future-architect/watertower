package watertower

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strconv"

	"github.com/shibukawa/compints"
	"github.com/shibukawa/watertower/nlp"
	"gocloud.dev/docstore"
)

// PostDocument registers single document to storage and update index
//
// uniqueKey is a key of document like URL.
//
// document's title and content fields are indexed and searched via natural language algorithms (tokenize, stemming).
//
// document's tags field contains texts and filter documents via complete match algorithm.
func (wt *WaterTower) PostDocument(uniqueKey string, document *Document) (uint32, error) {
	retryCount := 50
	var lastError error
	var docID uint32
	newTags, newDocTokens, wordCount, titleWordCount, err := wt.analyzeDocument("new", document)
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
		oldDoc, err := wt.postDocument(docID, uniqueKey, wordCount, titleWordCount, document)
		if err != nil {
			lastError = err
			continue
		}
		oldTags, oldDocTokens, _, _, err := wt.analyzeDocument("old", oldDoc)
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

func (wt *WaterTower) postDocumentKey(uniqueKey string) (uint32, error) {
	id := "k" + uniqueKey
	existingDocKey := DocumentKey{
		ID: id,
	}
	err := wt.collection.Get(wt.ctx, &existingDocKey)
	if err == nil {
		return existingDocKey.DocumentID, nil
	}
	newID, err := wt.counter.Increment(wt.ctx, documentID)
	if err != nil {
		return 0, err
	}
	err = wt.collection.Create(wt.ctx, &DocumentKey{
		ID:         id,
		DocumentID: uint32(newID),
	})
	if err != nil {
		return 0, err
	}
	return uint32(newID), nil
}

func (wt *WaterTower) postDocument(docID uint32, uniqueKey string, wordCount, titleWordCount int, document *Document) (*Document, error) {
	idStr := "d" + strconv.FormatUint(uint64(docID), 16)
	existingDocument := Document{
		ID: idStr,
	}
	document.ID = idStr
	document.UniqueKey = uniqueKey
	document.WordCount = wordCount
	document.TitleWordCount = titleWordCount
	err := wt.collection.Get(wt.ctx, &existingDocument)
	if err != nil {
		_, err = wt.counter.Increment(wt.ctx, documentCount)
		if err != nil {
			return nil, err
		}
		return nil, wt.collection.Create(wt.ctx, document)
	} else {
		return &existingDocument, wt.collection.Replace(wt.ctx, document)
	}
}

// RemoveDocument removes document via uniqueKey
func (wt *WaterTower) RemoveDocument(uniqueKey string) error {
	docID, existingDocKey, oldDoc, err := wt.findDocumentByKey(uniqueKey)
	if err != nil {
		return err
	}
	err = wt.collection.Delete(wt.ctx, existingDocKey)
	if err != nil {
		return err
	}
	err = wt.collection.Delete(wt.ctx, oldDoc)
	if err != nil {
		return err
	}
	err = wt.counter.Decrement(wt.ctx, documentCount)
	if err != nil {
		return err
	}
	tags, tokens, _, _, err := wt.analyzeDocument("removed", oldDoc)
	if err != nil {
		return err
	}
	return wt.updateTagsAndTokens(docID, tags, nil, tokens, nil)
}

func (wt *WaterTower) analyzeDocument(label string, document *Document) (tags []string, tokens map[string]*nlp.Token, wordCount, titleWordCount int, err error) {
	if document == nil {
		return nil, make(map[string]*nlp.Token), 0, 0, nil
	}
	tokenizer, err := nlp.FindTokenizer(document.Language)
	if err != nil {
		return nil, nil, 0, 0, fmt.Errorf("Cannot find tokenizer for %s document: lang=%s, err=%w", label, document.Language, err)
	}
	tokens, wordCount = tokenizer.TokenizeToMap(document.Title + "\n" + document.Content)
	titleWordCount = len(tokenizer.Tokenize(document.Title))
	return document.Tags, tokens, wordCount, titleWordCount, nil
}

func (wt *WaterTower) updateTagsAndTokens(docID uint32, oldTags, newTags []string, oldDocTokens, newDocTokens map[string]*nlp.Token) error {
	// update tags
	newTags, deletedTags := groupingTags(oldTags, newTags)
	for _, tag := range newTags {
		err := wt.addTagToDocumentID(tag, docID)
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
		err := wt.addDocumentToToken(token.Word, docID, token.Positions)
		if err != nil {
			return err
		}
	}
	for _, token := range deletedTokens {
		err := wt.removeDocumentFromToken(token.Word, docID)
		if err != nil {
			return err
		}
	}
	for _, token := range updateTokens {
		err := wt.addDocumentToToken(token.Word, docID, token.Positions)
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

// AddTagToDocument adds tag to existing document.
func (wt *WaterTower) AddTagToDocument(tag, uniqueKey string) error {
	return nil
}

func (wt *WaterTower) addTagToDocumentID(tag string, docID uint32) error {
	retryCount := 50
	var lastError error
	for i := 0; i < retryCount; i++ {
		err := wt.tryAddingDocumentToTag(tag, docID)
		if err != nil {
			lastError = err
			continue
		}
		return nil
	}
	return fmt.Errorf("fail to update tag: %w", lastError)
}

func (wt *WaterTower) tryAddingDocumentToTag(tag string, docID uint32) error {
	existingTag := TagEntity{
		ID: "t" + tag,
	}
	err := wt.collection.Get(wt.ctx, &existingTag)
	if err != nil {
		tag := TagEntity{
			ID:          "t" + tag,
			DocumentIDs: compints.CompressToBytes([]uint32{docID}, true),
		}
		return wt.collection.Create(wt.ctx, &tag)
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
			ID:          "t" + tag,
			DocumentIDs: compints.CompressToBytes(docIDs, true),
		}
		err = wt.collection.Replace(wt.ctx, newTag)
		if err != nil {
			return fmt.Errorf("fail to replace tag: '%s': %w", tag, err)
		}
		return nil
	}
}

func (wt *WaterTower) RemoveDocumentFromTag(tag string, docID uint32) error {
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

func (wt *WaterTower) removeDocumentFromTag(tag string, docID uint32) error {
	existingTag := TagEntity{
		ID: "t" + tag,
	}
	err := wt.collection.Get(wt.ctx, &existingTag)
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
		return wt.collection.Delete(wt.ctx, &existingTag)
	} else {
		newTag := &TagEntity{
			ID:          "t" + tag,
			DocumentIDs: compints.CompressToBytes(newDocIDs, true),
		}
		existingTag.DocumentIDs = compints.CompressToBytes(newDocIDs, true)
		return wt.collection.Replace(wt.ctx, newTag)
	}
}

func (wt *WaterTower) addDocumentToToken(word string, docID uint32, positions []uint32) error {
	retryCount := 50
	var lastError error
	for i := 0; i < retryCount; i++ {
		err := wt.addDocumentIDToToken(word, docID, positions)
		if err != nil {
			lastError = err
			continue
		}
		return nil
	}
	return fmt.Errorf("fail to update tag: %w", lastError)
}

func (wt *WaterTower) addDocumentIDToToken(word string, docID uint32, positions []uint32) error {
	existingToken := TokenEntity{
		ID: "w" + word,
	}
	err := wt.collection.Get(wt.ctx, &existingToken)
	postingEntity := PostingEntity{
		DocumentID: docID,
		Positions:  compints.CompressToBytes(positions, true),
	}
	if err != nil {
		token := TokenEntity{
			ID:       "w" + word,
			Postings: []PostingEntity{postingEntity},
		}
		return wt.collection.Create(wt.ctx, &token)
	} else {
		newToken := TokenEntity{
			ID:       "w" + word,
			Postings: append(existingToken.Postings, postingEntity),
		}
		sort.Slice(newToken.Postings, func(i, j int) bool {
			return newToken.Postings[i].DocumentID < newToken.Postings[j].DocumentID
		})
		err = wt.collection.Replace(wt.ctx, &newToken)
		if err != nil {
			return fmt.Errorf("fail to replace token: '%s': %w", word, err)
		}
		return nil
	}
}

func (wt *WaterTower) removeDocumentFromToken(word string, docID uint32) error {
	retryCount := 50
	var lastError error
	for i := 0; i < retryCount; i++ {
		err := wt.removeDocumentIDFromToken(word, docID)
		if err != nil {
			lastError = err
			continue
		}
		return nil
	}
	return fmt.Errorf("fail to update tag: %w", lastError)
}

func (wt *WaterTower) removeDocumentIDFromToken(word string, docID uint32) error {
	existingToken := TokenEntity{
		ID: "w" + word,
	}
	err := wt.collection.Get(wt.ctx, &existingToken)
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
			return wt.collection.Delete(wt.ctx, &existingToken)
		} else {
			newToken := TokenEntity{
				ID:       "w" + word,
				Postings: newPostings,
			}
			return wt.collection.Replace(wt.ctx, &newToken)
		}
	}
}

func (wt *WaterTower) FindTags(tagNames ...string) ([]*Tag, error) {
	return wt.FindTagsWithContext(wt.ctx, tagNames...)
}

func (wt *WaterTower) FindTagsWithContext(ctx context.Context, tagNames ...string) ([]*Tag, error) {
	if len(tagNames) == 0 {
		return nil, nil
	}
	existingTags := make([]TagEntity, len(tagNames))
	actions := wt.collection.Actions()
	for i, tagName := range tagNames {
		existingTags[i].ID = "t" + tagName
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
			ID:          existingTag.ID[1:],
			DocumentIDs: docIDs,
		}
	}
	return result, nil
}

func (wt *WaterTower) FindTokens(words ...string) ([]*Token, error) {
	return wt.FindTokensWithContext(wt.ctx, words...)
}

func (wt *WaterTower) FindTokensWithContext(ctx context.Context, words ...string) ([]*Token, error) {
	if len(words) == 0 {
		return nil, nil
	}
	positions := make(map[string][]int)
	for i, word := range words {
		positions[word] = append(positions[word], i)
	}
	existingTokens := make([]TokenEntity, len(positions))
	actions := wt.collection.Actions()
	for i, word := range words {
		existingTokens[i].ID = "w" + word
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
			Word:  existingToken.ID[1:],
			Found: !hasErrors[i],
		}
		for _, posting := range existingToken.Postings {
			positions, err := compints.DecompressFromBytes(posting.Positions, true)
			if err != nil {
				return nil, fmt.Errorf("Compressed data is broken of position of doc %d of token %s: %w", posting.DocumentID, existingToken.ID[1:], err)
			}
			token.Postings = append(token.Postings, Posting{
				DocumentID: posting.DocumentID,
				Positions:  positions,
			})
		}
		for _, pos := range positions[existingToken.ID[1:]] {
			result[pos] = token
		}
	}
	return result, nil
}

func (wt *WaterTower) FindDocuments(ids ...uint32) ([]*Document, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	result := make([]*Document, len(ids))
	actions := wt.collection.Actions()
	for i, id := range ids {
		result[i] = &Document{
			ID: "d" + strconv.FormatUint(uint64(id), 16),
		}
		actions = actions.Get(result[i])
	}
	err := actions.Do(wt.ctx)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (wt *WaterTower) FindDocumentByKey(uniqueKey string) (*Document, error) {
	_, _, doc, err := wt.findDocumentByKey(uniqueKey)
	return doc, err
}

func (wt *WaterTower) findDocumentByKey(uniqueKey string) (uint32, *DocumentKey, *Document, error) {
	existingDocKey := DocumentKey{
		ID: "k" + uniqueKey,
	}
	err := wt.collection.Get(wt.ctx, &existingDocKey)
	if err != nil {
		return 0, nil, nil, err
	}
	docID := existingDocKey.DocumentID
	oldDoc := Document{
		ID: "d" + strconv.FormatUint(uint64(docID), 16),
	}
	err = wt.collection.Get(wt.ctx, &oldDoc)
	if err != nil {
		return 0, nil, nil, err
	}
	return docID, &existingDocKey, &oldDoc, nil
}
