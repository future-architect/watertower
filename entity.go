package watertower

import (
	"fmt"
	"strconv"
	"time"
)

type Document struct {
	ID             string            `json:"-" docstore:"id"`
	UniqueKey      string            `json:"unique_key" docstore:"unique_key"`
	Language       string            `json:"lang" docstore:"lang"`
	Title          string            `json:"title" docstore:"title"`
	UpdatedAt      time.Time         `json:"updated_at,omitempty" docstore:"updated_at"`
	Tags           []string          `json:"tags,omitempty" docstore:"tags"`
	Content        string            `json:"content" docstore:"content"`
	WordCount      int               `json:"-" docstore:"wc"`
	Metadata       map[string]string `json:"metadata,omitempty" docstore:"metadata"`
	TitleWordCount int               `json:"-" docstore:"twc"`
	Score          float64           `json:"score,omitempty" docstore:"-"`

	Schema  string `json:"$schema,omitempty" docstore:"-"`
	Comment string `json:"$comment,omitempty" docstore:"-"`
}

func (d Document) DocumentID() (uint32, error) {
	str := d.ID[1:]
	docID, err := strconv.ParseUint(str, 16, 32)
	if err != nil {
		return 0, fmt.Errorf("Can't parse documentID: %s", d.ID)
	}
	return uint32(docID), nil
}

type documentKey struct {
	ID         string `json:"-" docstore:"id"`
	UniqueKey  string `json:"unique_key" docstore:"unique_key"`
	DocumentID uint32 `json:"docid" docstore:"docid"`
}

type token struct {
	Word     string
	Found    bool
	Postings []posting
}

func (t token) toPostingMap() map[uint32]posting {
	result := make(map[uint32]posting)
	for _, posting := range t.Postings {
		result[posting.DocumentID] = posting
	}
	return result
}

type tokenEntity struct {
	ID       string          `docstore:"id"`
	Postings []postingEntity `docstore:"postings"`
}

type posting struct {
	DocumentID uint32   `json:"document_id"`
	Positions  []uint32 `json:"positions"`
}

type postingEntity struct {
	DocumentID uint32 `docstore:"document_id"`
	Positions  []byte `docstore:"positions"`
}

type tag struct {
	ID          string   `docstore:"id"`
	DocumentIDs []uint32 `docstore:"documentIDs"`
}

type tagEntity struct {
	ID          string `docstore:"id"`
	DocumentIDs []byte `docstore:"documentIDs"`
}
