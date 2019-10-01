package watertower

import "time"

type Document struct {
	ID        uint32    `json:"id,omitempty" docstore:"id"`
	UniqueKey string    `json:"unique_key" docstore:"unique_key"`
	Language  string    `json:"lang" docstore:"lang"`
	Title     string    `json:"title" docstore:"title"`
	UpdatedAt time.Time `json:"updated_at,omitempty" docstore:"updated_at"`
	Tags      []string  `json:"tags" docstore:"tags"`
	Content   string    `json:"content" docstore:"content"`
	Summary   string    `json:"summary" docstore:"summary"`
}

type DocumentKey struct {
	UniqueKey string `json:"unique_key" docstore:"unique_key"`
	ID        uint32 `json:"id" docstore:"id"`
}

type Token struct {
	Word     string             `json:"word"`
	Postings map[uint32]Posting `json:"postings"`
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
