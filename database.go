package watertower

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strconv"

	"github.com/future-architect/watertower/nlp"
	_ "github.com/future-architect/watertower/nlp/bigram"
	_ "github.com/future-architect/watertower/nlp/unigram"
	"github.com/shibukawa/compints"
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
		oldDoc, err := wt.postDocumentBody(docID, uniqueKey, wordCount, titleWordCount, document)
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
	existingDocKey := documentKey{
		ID: id,
	}
	err := wt.storage.GetDocKey(&existingDocKey)
	if err == nil {
		return existingDocKey.DocumentID, nil
	}
	newID, err := wt.storage.IncrementDocID()
	if err != nil {
		return 0, err
	}
	key := &documentKey{
		ID:         id,
		DocumentID: uint32(newID),
	}
	err = wt.storage.Create(id, key)
	if err != nil {
		return 0, err
	}
	return uint32(newID), nil
}

func (wt *WaterTower) postDocumentBody(docID uint32, uniqueKey string, wordCount, titleWordCount int, doc *Document) (*Document, error) {
	idStr := "d" + strconv.FormatUint(uint64(docID), 16)
	existingDocument := Document{
		ID: idStr,
	}
	doc.ID = idStr
	doc.UniqueKey = uniqueKey
	doc.WordCount = wordCount
	doc.TitleWordCount = titleWordCount
	err := wt.storage.GetDoc(&existingDocument)
	if err != nil {
		_, err = wt.storage.IncrementDocCount()
		if err != nil {
			return nil, err
		}
		return nil, wt.storage.Create(idStr, doc)
	} else {
		return &existingDocument, wt.storage.Replace(idStr, doc)
	}
}

// RemoveDocumentByKey removes document via uniqueKey
func (wt *WaterTower) RemoveDocumentByKey(uniqueKey string) error {
	docID, existingDocKey, oldDoc, err := wt.findDocumentByKey(uniqueKey)
	if err != nil {
		return err
	}
	err = wt.storage.DeleteDocKey(existingDocKey)
	if err != nil {
		return err
	}
	err = wt.storage.DeleteDoc(oldDoc)
	if err != nil {
		return err
	}
	err = wt.storage.DecrementDocCount()
	if err != nil {
		return err
	}
	tags, tokens, _, _, err := wt.analyzeDocument("removed", oldDoc)
	if err != nil {
		return err
	}
	return wt.updateTagsAndTokens(docID, tags, nil, tokens, nil)
}

// RemoveDocumentByID removes document via ID
func (wt *WaterTower) RemoveDocumentByID(docID uint32) error {
	docs, err := wt.FindDocuments(docID)
	if err != nil {
		return err
	}
	existingDocKey := documentKey{
		ID: "k" + docs[0].UniqueKey,
	}
	err = wt.storage.DeleteDocKey(&existingDocKey)
	if err != nil {
		return err
	}
	err = wt.storage.DeleteDoc(docs[0])
	if err != nil {
		return err
	}
	err = wt.storage.DecrementDocCount()
	if err != nil {
		return err
	}
	tags, tokens, _, _, err := wt.analyzeDocument("removed", docs[0])
	if err != nil {
		return err
	}
	return wt.updateTagsAndTokens(docID, tags, nil, tokens, nil)
}

func (wt *WaterTower) analyzeDocument(label string, document *Document) (tags []string, tokens map[string]*nlp.Token, wordCount, titleWordCount int, err error) {
	if document == nil {
		return nil, make(map[string]*nlp.Token), 0, 0, nil
	}
	language := document.Language
	if language == "" {
		language = wt.defaultLanguage
	}
	if language == "" {
		tags, tokens, wordCount, titleWordCount, err := wt.analyzeDocumentWithLanguage(label, document, "bigram", 0)
		if err != nil {
			return nil, nil, 0, 0, err
		}
		_, tokens2, wordCount2, titleWordCount2, err := wt.analyzeDocumentWithLanguage(label, document, "unigram", wordCount)
		if err != nil {
			return nil, nil, 0, 0, err
		}
		// merge uni-gram index and bi-gram index
		for key, token := range tokens2 {
			for i, pos := range token.Positions {
				token.Positions[i] = pos + uint32(wordCount)
			}
			tokens[key] = token
		}
		if wordCount == 0 {
			wordCount = wordCount2
		}
		if titleWordCount == 0 {
			titleWordCount = titleWordCount2
		}
		return tags, tokens, wordCount, titleWordCount, nil
	} else {
		return wt.analyzeDocumentWithLanguage(label, document, language, 0)
	}
}

func (wt *WaterTower) analyzeDocumentWithLanguage(label string, document *Document, language string, offset int) (tags []string, tokens map[string]*nlp.Token, wordCount, titleWordCount int, err error) {
	tokenizer, err := nlp.FindTokenizer(language)
	if err != nil {
		return nil, nil, 0, 0, fmt.Errorf("Cannot find tokenizer for %s document: lang=%s, err=%w", label, document.Language, err)
	}
	tokens, titleWordCount = tokenizer.TokenizeToMap(document.Title, offset)
	bodyTokens, wordCount := tokenizer.TokenizeToMap(document.Content, offset+titleWordCount+1)
	for k, v := range bodyTokens {
		tokens[k] = v
	}
	return document.Tags, tokens, wordCount + titleWordCount, titleWordCount, nil
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
	return fmt.Errorf("fail to add tag: %w", lastError)
}

func (wt *WaterTower) tryAddingDocumentToTag(tag string, docID uint32) error {
	existingTag := tagEntity{
		ID: "t" + tag,
	}
	err := wt.storage.GetTag(&existingTag)
	if err != nil {
		tag := tagEntity{
			ID:          "t" + tag,
			DocumentIDs: compints.CompressToBytes([]uint32{docID}, true),
		}
		return wt.storage.Create(tag.ID, &tag)
	} else {
		docIDs, err := compints.DecompressFromBytes(existingTag.DocumentIDs, true)
		if err != nil {
			return fmt.Errorf("fail to decompress document IDs of tag '%s': %w", tag, err)
		}
		docIDs = append(docIDs, docID)
		sort.Slice(docIDs, func(i, j int) bool {
			return docIDs[i] < docIDs[j]
		})
		newTag := &tagEntity{
			ID:          "t" + tag,
			DocumentIDs: compints.CompressToBytes(docIDs, true),
		}
		err = wt.storage.Replace(newTag.ID, newTag)
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
	return fmt.Errorf("fail to remove tag: %w", lastError)
}

func (wt *WaterTower) removeDocumentFromTag(tag string, docID uint32) error {
	existingTag := tagEntity{
		ID: "t" + tag,
	}
	err := wt.storage.GetTag(&existingTag)
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
		return wt.storage.DeleteTag(&existingTag)
	} else {
		newTag := &tagEntity{
			ID:          "t" + tag,
			DocumentIDs: compints.CompressToBytes(newDocIDs, true),
		}
		existingTag.DocumentIDs = compints.CompressToBytes(newDocIDs, true)
		return wt.storage.Replace(newTag.ID, newTag)
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
	return fmt.Errorf("fail to add token to doc: %w", lastError)
}

func (wt *WaterTower) addDocumentIDToToken(word string, docID uint32, positions []uint32) error {
	existingToken := tokenEntity{
		ID: "w" + word,
	}
	err := wt.storage.GetToken(&existingToken)
	pe := postingEntity{
		DocumentID: docID,
		Positions:  compints.CompressToBytes(positions, true),
	}
	if err != nil {
		token := tokenEntity{
			ID:       "w" + word,
			Postings: []postingEntity{pe},
		}
		return wt.storage.Create(token.ID, &token)
	} else {
		newToken := tokenEntity{
			ID:       "w" + word,
			Postings: append(existingToken.Postings, pe),
		}
		sort.Slice(newToken.Postings, func(i, j int) bool {
			return newToken.Postings[i].DocumentID < newToken.Postings[j].DocumentID
		})
		err = wt.storage.Replace(newToken.ID, &newToken)
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
	return fmt.Errorf("fail to remove token from doc: %w", lastError)
}

func (wt *WaterTower) removeDocumentIDFromToken(word string, docID uint32) error {
	existingToken := tokenEntity{
		ID: "w" + word,
	}
	err := wt.storage.GetToken(&existingToken)
	if err != nil {
		return err
	} else {
		newPostings := make([]postingEntity, 0, len(existingToken.Postings)-1)
		for _, existingPosting := range existingToken.Postings {
			if existingPosting.DocumentID != docID {
				newPostings = append(newPostings, existingPosting)
			}
		}
		if len(newPostings) == 0 {
			return wt.storage.DeleteToken(&existingToken)
		} else {
			newToken := tokenEntity{
				ID:       "w" + word,
				Postings: newPostings,
			}
			return wt.storage.Replace(newToken.ID, &newToken)
		}
	}
}

func (wt *WaterTower) FindTags(tagNames ...string) ([]*tag, error) {
	return wt.FindTagsWithContext(wt.storage.Context(), tagNames...)
}

func (wt *WaterTower) FindTagsWithContext(ctx context.Context, tagNames ...string) ([]*tag, error) {
	if len(tagNames) == 0 {
		return nil, nil
	}
	existingTags := make([]*tagEntity, len(tagNames))
	for i, tagName := range tagNames {
		existingTags[i] = &tagEntity{
			ID: "t" + tagName,
		}
	}
	_, err := wt.storage.BatchTagGet(ctx, existingTags)
	if err != nil {
		return nil, err
	}
	result := make([]*tag, len(tagNames))
	for i, existingTag := range existingTags {
		docIDs, err := compints.DecompressFromBytes(existingTag.DocumentIDs, true)
		if err != nil {
			return nil, err
		}
		result[i] = &tag{
			ID:          existingTag.ID[1:],
			DocumentIDs: docIDs,
		}
	}
	return result, nil
}

func (wt *WaterTower) FindTokens(words ...string) ([]*token, error) {
	return wt.FindTokensWithContext(wt.storage.Context(), words...)
}

func (wt *WaterTower) FindTokensWithContext(ctx context.Context, words ...string) ([]*token, error) {
	if len(words) == 0 {
		return nil, nil
	}
	positions := make(map[string][]int)
	for i, word := range words {
		positions[word] = append(positions[word], i)
	}
	existingTokens := make([]*tokenEntity, len(positions))
	for i, word := range words {
		existingTokens[i] = &tokenEntity{
			ID: "w" + word,
		}
	}
	hasErrors, _ := wt.storage.BatchTokenGet(ctx, existingTokens)
	result := make([]*token, len(words))
	for i, existingToken := range existingTokens {
		token := &token{
			Word:  existingToken.ID[1:],
			Found: !hasErrors[i],
		}
		for _, p := range existingToken.Postings {
			positions, err := compints.DecompressFromBytes(p.Positions, true)
			if err != nil {
				return nil, fmt.Errorf("Compressed data is broken of position of doc %d of token %s: %w", p.DocumentID, existingToken.ID[1:], err)
			}
			token.Postings = append(token.Postings, posting{
				DocumentID: p.DocumentID,
				Positions:  positions,
			})
		}
		for _, pos := range positions[existingToken.ID[1:]] {
			result[pos] = token
		}
	}
	return result, nil
}

// FindDocuments returns documents by id list.
func (wt *WaterTower) FindDocuments(ids ...uint32) ([]*Document, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	docs := make([]*Document, len(ids))

	for i, id := range ids {
		doc := &Document{
			ID: "d" + strconv.FormatUint(uint64(id), 16),
		}
		docs[i] = doc
	}
	_, err := wt.storage.BatchDocGet(wt.storage.Context(), docs)
	if err != nil {
		return nil, err
	}

	return docs, nil
}

// FindDocumentByKey looks up document by uniqueKey.
func (wt *WaterTower) FindDocumentByKey(uniqueKey string) (*Document, error) {
	_, _, doc, err := wt.findDocumentByKey(uniqueKey)
	return doc, err
}

func (wt *WaterTower) findDocumentByKey(uniqueKey string) (uint32, *documentKey, *Document, error) {
	existingDocKey := documentKey{
		ID: "k" + uniqueKey,
	}
	err := wt.storage.GetDocKey(&existingDocKey)
	if err != nil {
		return 0, nil, nil, err
	}
	docID := existingDocKey.DocumentID
	oldDoc := Document{
		ID: "d" + strconv.FormatUint(uint64(docID), 16),
	}
	err = wt.storage.GetDoc(&oldDoc)
	if err != nil {
		return 0, nil, nil, err
	}
	return docID, &existingDocKey, &oldDoc, nil
}
