package watertower

import (
	"encoding/base64"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strconv"

	"github.com/shibukawa/compints"
	"github.com/shibukawa/watertower/nlp"
	"gocloud.dev/docstore"
	"gocloud.dev/pubsub"
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
	iter := c.docKeys.Query().OrderBy("id", docstore.Descending).Limit(1).Get(c.ctx)
	defer iter.Stop()
	var latestKey DocumentKey
	err = iter.Next(c.ctx, &latestKey)
	var newID uint32
	if err == io.EOF {
		newID = 1
	} else if err != nil {
		return 0, err
	} else {
		newID = latestKey.ID + 1
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
	docIDstr := strconv.Itoa(int(docID))
	load := func(label string, document *Document) ([]string, map[string]*nlp.Token, error) {
		if document == nil {
			return nil, make(map[string]*nlp.Token), nil
		}
		tokenizer, err := nlp.FindTokenizer(document.Language)
		if err != nil {
			return nil, nil, fmt.Errorf("Cannot find tokenizer for %s document: lang=%s, err=%w", label, document.Language, err)
		}
		documentTokens := tokenizer.Tokenize(document.Content)
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
	err = c.updateTags(newTags, docIDstr, "new")
	if err != nil {
		return err
	}
	err = c.updateTags(deletedTags, docIDstr, "delete")
	if err != nil {
		return err
	}

	// update tokens
	newTokens, deletedTokens, updateTokens := groupingTokens(oldDocumentTokens, newDocumentTokens)
	err = c.updateTokens(newTokens, docIDstr, "new", true)
	if err != nil {
		return err
	}
	err = c.updateTokens(deletedTokens, docIDstr, "delete", true)
	if err != nil {
		return err
	}
	err = c.updateTokens(updateTokens, docIDstr, "update", true)
	if err != nil {
		return err
	}
	return nil
}

func (c Client) updateTokens(tokens []*nlp.Token, docIDstr string, action string, sendPositions bool) error {
	for _, token := range tokens {
		metadata := map[string]string{
			"docID":  docIDstr,
			"target": "token",
			"token":  token.Word,
			"action": action,
		}
		if sendPositions {
			metadata["positions"] = base64.StdEncoding.EncodeToString(compints.CompressToBytes(token.Positions, true))
		}
		err := c.fanOut(&pubsub.Message{
			Metadata: metadata,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (c Client) updateTags(tags []string, docIDstr string, action string) error {
	for _, tag := range tags {
		err := c.fanOut(&pubsub.Message{
			Metadata: map[string]string{
				"docID":  docIDstr,
				"target": "tag",
				"tag":    tag,
				"action": action,
			},
		})
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
	if len(tagNames) == 0 {
		return nil, nil
	}
	existingTags := make([]TagEntity, len(tagNames))
	actions := c.tags.Actions()
	for i, tagName := range tagNames {
		existingTags[i].Tag = tagName
		actions = actions.Get(&existingTags[i])
	}
	err := actions.Do(c.ctx)
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
	if len(words) == 0 {
		return nil, nil
	}
	existingTokens := make([]TokenEntity, len(words))
	actions := c.tokens.Actions()
	for i, word := range words {
		existingTokens[i].Word = word
		actions = actions.Get(&existingTokens[i])
	}
	err := actions.Do(c.ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*Token, len(words))
	for i, existingToken := range existingTokens {
		token := &Token{
			Word: existingToken.Word,
		}
		for _, posting := range existingToken.Postings {
			positions, err := compints.DecompressFromBytes(posting.Positions, true)
			if err != nil {
				return nil, err
			}
			token.Postings = append(token.Postings, Posting{
				DocumentID: posting.DocumentID,
				Positions:  positions,
			})
		}
		result[i] = token
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
