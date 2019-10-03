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

func (c Client) PostDocument(uniqueKey string, document *Document) (uint32, error) {
	retryCount := 50
	var lastError error
	var docID uint32
	for i := 0; i < retryCount; i++ {
		docID, lastError = c.postDocumentKey(uniqueKey)
		if lastError == nil {
			break
		}
	}
	if lastError != nil {
		return 0, fmt.Errorf("fail to register document's unique key: %w", lastError)
	}
	for i := 0; i < retryCount; i++ {
		oldDoc, err := c.postDocument(docID, uniqueKey, document)
		if err != nil {
			lastError = err
			continue
		}
		err = c.updateTagsAndTokens(docID, document, oldDoc)
		if err != nil {
			lastError = err
			continue
		}
		return docID, nil
	}
	return 0, fmt.Errorf("fail to register document: %w", lastError)
}

func (c Client) postDocumentKey(uniqueKey string) (uint32, error) {
	existingDocKey := DocumentKey{
		UniqueKey: uniqueKey,
	}
	err := c.docKeys.Get(c.ctx, &existingDocKey)
	if err == nil {
		return existingDocKey.ID, nil
	}
	newID, err := c.incrementID()
	if err != nil {
		return 0, err
	}
	err = c.docKeys.Create(c.ctx, &DocumentKey{
		UniqueKey: uniqueKey,
		ID:        newID,
	})
	if err != nil {
		return 0, err
	}
	return newID, nil
}

func (c Client) postDocument(docID uint32, uniqueKey string, document *Document) (*Document, error) {
	existingDocument := Document{
		ID: docID,
	}
	document.ID = docID
	document.UniqueKey = uniqueKey
	err := c.documents.Get(c.ctx, &existingDocument)
	if err != nil {
		return nil, c.documents.Create(c.ctx, document)
	} else {
		return &existingDocument, c.documents.Replace(c.ctx, document)
	}
}

func (c Client) incrementID() (uint32, error) {
	uniqueID := UniqueID{
		Collection: "documents",
	}
	latestUniqueID := UniqueID{
		Collection: "documents",
	}
	err := c.uniqueIDs.Actions().
		Update(&uniqueID, docstore.Mods{"latestID": docstore.Increment(1)}).
		Get(&latestUniqueID).
		Do(c.ctx)
	if err != nil {
		return 0, err
	}
	return latestUniqueID.LatestID, nil
}

func (c Client) RemoveDocument(uniqueKey string) error {
	docID, existingDocKey, oldDoc, err := c.findDocumentByKey(uniqueKey)
	if err != nil {
		return err
	}
	err = c.docKeys.Delete(c.ctx, existingDocKey)
	if err != nil {
		return err
	}
	err = c.documents.Delete(c.ctx, oldDoc)
	if err != nil {
		return err
	}
	return c.updateTagsAndTokens(docID, nil, oldDoc)
}

func (c Client) updateTagsAndTokens(docID uint32, newDocument, oldDocument *Document) error {
	load := func(label string, document *Document) ([]string, map[string]*nlp.Token, error) {
		if document == nil {
			return nil, make(map[string]*nlp.Token), nil
		}
		tokenizer, err := nlp.FindTokenizer(document.Language)
		if err != nil {
			return nil, nil, fmt.Errorf("Cannot find tokenizer for %s document: lang=%s, err=%w", label, document.Language, err)
		}
		documentTokens := tokenizer.TokenizeToMap(document.Content)
		return document.Tags, documentTokens, nil
	}

	newTags, newDocumentTokens, err := load("new", newDocument)
	if err != nil {
		return err
	}
	oldTags, oldDocumentTokens, err := load("old", oldDocument)
	if err != nil {
		return err
	}

	// update tags
	newTags, deletedTags := groupingTags(oldTags, newTags)
	for _, tag := range newTags {
		err := c.AddDocumentToTag(tag, docID)
		if err != nil {
			return err
		}
	}
	for _, tag := range deletedTags {
		err := c.RemoveDocumentFromTag(tag, docID)
		if err != nil {
			return err
		}
	}

	// update tokens
	newTokens, deletedTokens, updateTokens := groupingTokens(oldDocumentTokens, newDocumentTokens)
	for _, token := range newTokens {
		err := c.AddDocumentToToken(token.Word, docID, token.Positions)
		if err != nil {
			return err
		}
	}
	for _, token := range deletedTokens {
		err := c.RemoveDocumentFromToken(token.Word, docID)
		if err != nil {
			return err
		}
	}
	for _, token := range updateTokens {
		err := c.AddDocumentToToken(token.Word, docID, token.Positions)
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

func (c Client) AddDocumentToTag(tag string, docID uint32) error {
	retryCount := 50
	var lastError error
	for i := 0; i < retryCount; i++ {
		err := c.addDocumentToTag(tag, docID)
		if err != nil {
			lastError = err
			continue
		}
		return nil
	}
	return fmt.Errorf("fail to update tag: %w", lastError)
}

func (c Client) addDocumentToTag(tag string, docID uint32) error {
	existingTag := TagEntity{
		Tag: tag,
	}
	err := c.tags.Get(c.ctx, &existingTag)
	if err != nil {
		tag := TagEntity{
			Tag:         tag,
			DocumentIDs: compints.CompressToBytes([]uint32{docID}, true),
		}
		return c.tags.Create(c.ctx, &tag)
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
		err = c.tags.Replace(c.ctx, newTag)
		if err != nil {
			return fmt.Errorf("fail to replace tag: '%s': %w", tag, err)
		}
		return nil
	}
}

func (c Client) RemoveDocumentFromTag(tag string, docID uint32) error {
	retryCount := 50
	var lastError error
	for i := 0; i < retryCount; i++ {
		err := c.removeDocumentFromTag(tag, docID)
		if err != nil {
			lastError = err
			continue
		}
		return nil
	}
	return fmt.Errorf("fail to update tag: %w", lastError)
}

func (c Client) removeDocumentFromTag(tag string, docID uint32) error {
	existingTag := TagEntity{
		Tag: tag,
	}
	err := c.tags.Get(c.ctx, &existingTag)
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
		return c.tags.Delete(c.ctx, &existingTag)
	} else {
		newTag := &TagEntity{
			Tag:         tag,
			DocumentIDs: compints.CompressToBytes(newDocIDs, true),
		}
		existingTag.DocumentIDs = compints.CompressToBytes(newDocIDs, true)
		return c.tags.Replace(c.ctx, newTag)
	}
}

func (c Client) AddDocumentToToken(word string, docID uint32, positions []uint32) error {
	retryCount := 50
	var lastError error
	for i := 0; i < retryCount; i++ {
		err := c.addDocumentToToken(word, docID, positions)
		if err != nil {
			lastError = err
			continue
		}
		return nil
	}
	return fmt.Errorf("fail to update tag: %w", lastError)
}

func (c Client) addDocumentToToken(word string, docID uint32, positions []uint32) error {
	existingToken := TokenEntity{
		Word: word,
	}
	err := c.tokens.Get(c.ctx, &existingToken)
	postingEntity := PostingEntity{
		DocumentID: docID,
		Positions:  compints.CompressToBytes(positions, true),
	}
	if err != nil {
		token := TokenEntity{
			Word:     word,
			Postings: []PostingEntity{postingEntity},
		}
		return c.tokens.Create(c.ctx, &token)
	} else {
		newToken := TokenEntity{
			Word:     word,
			Postings: append(existingToken.Postings, postingEntity),
		}
		sort.Slice(newToken.Postings, func(i, j int) bool {
			return newToken.Postings[i].DocumentID < newToken.Postings[j].DocumentID
		})
		err = c.tokens.Replace(c.ctx, &newToken)
		if err != nil {
			return fmt.Errorf("fail to replace token: '%s': %w", word, err)
		}
		return nil
	}
}

func (c Client) RemoveDocumentFromToken(word string, docID uint32) error {
	retryCount := 50
	var lastError error
	for i := 0; i < retryCount; i++ {
		err := c.removeDocumentFromToken(word, docID)
		if err != nil {
			lastError = err
			continue
		}
		return nil
	}
	return fmt.Errorf("fail to update tag: %w", lastError)
}

func (c Client) removeDocumentFromToken(word string, docID uint32) error {
	existingToken := TokenEntity{
		Word: word,
	}
	err := c.tokens.Get(c.ctx, &existingToken)
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
			return c.tokens.Delete(c.ctx, &existingToken)
		} else {
			newToken := TokenEntity{
				Word:     word,
				Postings: newPostings,
			}
			return c.tokens.Replace(c.ctx, &newToken)
		}
	}
}

func (c Client) FindTags(tagNames ...string) ([]*Tag, error) {
	return c.FindTagsWithContext(c.ctx, tagNames...)
}

func (c Client) FindTagsWithContext(ctx context.Context, tagNames ...string) ([]*Tag, error) {
	if len(tagNames) == 0 {
		return nil, nil
	}
	existingTags := make([]TagEntity, len(tagNames))
	actions := c.tags.Actions()
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

func (c Client) FindTokens(words ...string) ([]*Token, error) {
	return c.FindTokensWithContext(c.ctx, words...)
}

func (c Client) FindTokensWithContext(ctx context.Context, words ...string) ([]*Token, error) {
	if len(words) == 0 {
		return nil, nil
	}
	positions := make(map[string][]int)
	for i, word := range words {
		positions[word] = append(positions[word], i)
	}
	existingTokens := make([]TokenEntity, len(positions))
	actions := c.tokens.Actions()
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

func (c Client) FindDocuments(ids ...uint32) ([]*Document, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	result := make([]*Document, len(ids))
	actions := c.documents.Actions()
	for i, id := range ids {
		result[i] = &Document{
			ID: id,
		}
		actions = actions.Get(result[i])
	}
	err := actions.Do(c.ctx)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (c Client) FindDocumentByKey(uniqueKey string) (*Document, error) {
	_, _, doc, err := c.findDocumentByKey(uniqueKey)
	return doc, err
}

func (c Client) findDocumentByKey(uniqueKey string) (uint32, *DocumentKey, *Document, error) {
	existingDocKey := DocumentKey{
		UniqueKey: uniqueKey,
	}
	err := c.docKeys.Get(c.ctx, &existingDocKey)
	if err != nil {
		return 0, nil, nil, err
	}
	docID := existingDocKey.ID
	oldDoc := Document{
		ID: docID,
	}
	err = c.documents.Get(c.ctx, &oldDoc)
	if err != nil {
		return 0, nil, nil, err
	}
	return docID, &existingDocKey, &oldDoc, nil
}
