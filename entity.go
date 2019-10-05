package watertower

import "time"

type Document struct {
	ID          uint32            `json:"id,omitempty" docstore:"id"`
	UniqueKey   string            `json:"unique_key" docstore:"unique_key"`
	Language    string            `json:"lang" docstore:"lang"`
	Title       string            `json:"title" docstore:"title"`
	UpdatedAt   time.Time         `json:"updated_at,omitempty" docstore:"updated_at"`
	Tags        []string          `json:"tags,omitempty" docstore:"tags"`
	Content     string            `json:"content" docstore:"content"`
	WordCount   int               `json:"-" docstore:"wordcount"`
	Metadata    map[string]string `json:"metadata,omitempty" docstore:"metadata"`
	TitleTokens int               `json:"-" docstore:"title_tokens"`
	Score       float64           `json:"score,omitempty" docstore:"-"`

	Schema  string `json:"$schema,omitempty" docstore:"-"`
	Comment string `json:"$comment,omitempty" docstore:"-"`
}

type DocumentKey struct {
	UniqueKey string `json:"unique_key" docstore:"unique_key"`
	ID        uint32 `json:"id" docstore:"id"`
}

type Token struct {
	Word     string    `json:"word"`
	Found    bool      `json:"found"`
	Postings []Posting `json:"postings"`
}

func (t Token) toPostingMap() map[uint32]Posting {
	result := make(map[uint32]Posting)
	for _, posting := range t.Postings {
		result[posting.DocumentID] = posting
	}
	return result
}

type TokenEntity struct {
	Word     string          `docstore:"word"`
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
	Tag         string   `json:"tag" docstore:"tag"`
	DocumentIDs []uint32 `json:"documentIDs" docstore:"documentIDs"`
}

type TagEntity struct {
	Tag         string `docstore:"tag"`
	DocumentIDs []byte `docstore:"documentIDs"`
}

type UniqueID struct {
	Collection string `docstore:"collection"`
	LatestID   uint32 `docstore:"latestID"`
}
