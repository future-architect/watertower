package watertower

import "time"

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

type DocumentKey struct {
	ID         string `json:"-" docstore:"id"`
	UniqueKey  string `json:"unique_key" docstore:"unique_key"`
	DocumentID uint32 `json:"docid" docstore:"docid"`
}

type Token struct {
	Word     string
	Found    bool
	Postings []Posting
}

func (t Token) toPostingMap() map[uint32]Posting {
	result := make(map[uint32]Posting)
	for _, posting := range t.Postings {
		result[posting.DocumentID] = posting
	}
	return result
}

type TokenEntity struct {
	ID       string          `docstore:"id"`
	Postings []PostingEntity `docstore:"postings"`
}

type Posting struct {
	DocumentID uint32   `json:"document_id"`
	Positions  []uint32 `json:"positions"`
}

type PostingEntity struct {
	DocumentID uint32 `docstore:"document_id"`
	Positions  []byte `docstore:"positions"`
}

type Tag struct {
	ID          string   `docstore:"id"`
	DocumentIDs []uint32 `docstore:"documentIDs"`
}

type TagEntity struct {
	ID          string `docstore:"id"`
	DocumentIDs []byte `docstore:"documentIDs"`
}

type CounterEntity struct {
	ID      string `docstore:"id"`
	Counter int    `docstore:"counter"`
}
