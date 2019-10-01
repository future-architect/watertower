package watertower

import (
	"github.com/shibukawa/compints"
)

func (c Client) FindTag(tagName string) (*Tag, error) {
	existingTag := TagEntity{
		Tag: tagName,
	}
	err := c.tags.Get(c.ctx, &existingTag)
	if err != nil {
		return nil, err
	}
	docIDs, err := compints.DecompressFromBytes(existingTag.DocumentIDs, true)
	if err != nil {
		return nil, err
	}
	tag := &Tag{
		Tag:         tagName,
		DocumentIDs: docIDs,
	}
	return tag, nil
}

func (c Client) FindToken(word string) (*Token, error) {
	existingToken := TokenEntity{
		Word: word,
	}
	err := c.tokens.Get(c.ctx, &existingToken)
	if err != nil {
		return nil, err
	}
	token := &Token{
		Word:     word,
		Postings: make(map[uint32]Posting),
	}
	for _, posting := range existingToken.Postings {
		positions, err := compints.DecompressFromBytes(posting.Positions, true)
		if err != nil {
			return nil, err
		}
		token.Postings[posting.DocumentID] = Posting{
			DocumentID: posting.DocumentID,
			Positions:  positions,
		}
	}
	return token, nil
}

func (c Client) FindDocument(id uint32) (*Document, error) {
	existingDocument := Document{
		ID: id,
	}
	err := c.documents.Get(c.ctx, &existingDocument)
	if err != nil {
		return nil, err
	}
	return &existingDocument, nil
}

func (c Client) FindDocumentByKey(uniqueKey string) (*Document, error) {
	_, _, doc, err := c.findDocumentByID(uniqueKey)
	return doc, err
}

func (c Client) findDocumentByID(uniqueKey string) (uint32, *DocumentKey, *Document, error) {
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

func (c Client) Search(searchWord string, tags []string) ([]*Document, error) {
	panic("not implemented")
}
