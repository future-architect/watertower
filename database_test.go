package watertower

import (
	"context"
	"testing"
	"time"

	_ "github.com/future-architect/watertower/nlp/english"
	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	_ "gocloud.dev/docstore/memdocstore"
)

func Test_PostDocument_IncrementID(t *testing.T) {
	wt, err := NewWaterTower(context.Background(), Option{
		Index:       xid.New().String(),
		DocumentUrl: "mem://",
	})
	assert.Nil(t, err)
	defer func() {
		err := wt.Close()
		assert.Nil(t, err)
	}()

	docID, err := wt.postDocumentKey("key")
	assert.Nil(t, err)
	assert.Equal(t, uint32(1), docID)

	docID, err = wt.postDocumentKey("new-key")
	assert.Nil(t, err)
	assert.Equal(t, uint32(2), docID)
}

func Test_PostDocument(t *testing.T) {
	wt, err := NewWaterTower(context.Background(), Option{
		Index:       xid.New().String(),
		DocumentUrl: "mem://",
	})
	assert.Nil(t, err)
	defer func() {
		err := wt.Close()
		assert.Nil(t, err)
	}()

	doc := &Document{
		Language:  "en",
		Title:     "old title",
		UpdatedAt: time.Time{},
		Tags:      []string{"go", "website", "introduction"},
		Content:   "Go is an open source programming language that makes it easy to build simple, reliable, and efficient software.",
	}
	docID, err := wt.postDocumentKey("test")
	assert.Nil(t, err)
	_, _, wordCount, titleWordCount, err := wt.analyzeDocument("new", doc)
	assert.Nil(t, err)
	oldDoc, err := wt.postDocumentBody(docID, "test", wordCount, titleWordCount, doc)
	assert.Nil(t, err)
	assert.Nil(t, oldDoc)

	loadedDocs, err := wt.FindDocuments(docID)
	assert.Nil(t, err)
	assert.Equal(t, "old title", loadedDocs[0].Title)
	assert.Equal(t, wordCount, loadedDocs[0].WordCount)
	assert.Equal(t, titleWordCount, loadedDocs[0].TitleWordCount)

	doc.Title = "new title"
	oldDoc, err = wt.postDocumentBody(docID, "test", doc.WordCount, doc.TitleWordCount, doc)
	assert.Nil(t, err)
	assert.Equal(t, "old title", oldDoc.Title)

	loadedDoc, err := wt.FindDocumentByKey("test")
	assert.Nil(t, err)
	assert.Equal(t, "new title", loadedDoc.Title)
}

func Test_analyzeDocumentWithNgram(t *testing.T) {
	wt, err := NewWaterTower(context.Background(), Option{
		Index:       xid.New().String(),
		DocumentUrl: "mem://",
	})
	assert.Nil(t, err)
	defer func() {
		err := wt.Close()
		assert.Nil(t, err)
	}()

	type args struct {
		content string
	}

	tests := []struct {
		name          string
		args          args
		wantTokens    int
		wantWordCount int
	}{
		{
			name: "uni-gram: single word",
			args: args{
				content: "G",
			},
			wantTokens:    1,
			wantWordCount: 1,
		},
		{
			name: "bi-gram + uni-gram: single word",
			args: args{
				content: "Go",
			},
			wantTokens:    1 + 2,
			wantWordCount: 1,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			doc := &Document{
				Language:  "", // fallback to n-gram
				Title:     "",
				UpdatedAt: time.Time{},
				Tags:      []string{},
				Content:   tc.args.content,
			}
			_, tokens, wordCount, _, err := wt.analyzeDocument("test", doc)
			assert.Nil(t, err)
			assert.Equal(t, tc.wantTokens, len(tokens))
			for k, v := range tokens {
				t.Log(k, *v)
			}
			assert.Equal(t, tc.wantWordCount, wordCount)
			t.Log(wordCount)
		})
	}
}

func Test_PostDocumentWithNgram(t *testing.T) {
	wt, err := NewWaterTower(context.Background(), Option{
		Index:       xid.New().String(),
		DocumentUrl: "mem://",
	})
	assert.Nil(t, err)
	defer func() {
		err := wt.Close()
		assert.Nil(t, err)
	}()

	src := &Document{
		Language:  "", // if language is empty, watertower uses n-gram tokenizer when default language is unavailable
		Title:     "old title",
		UpdatedAt: time.Time{},
		Tags:      []string{"go", "website", "introduction"},
		Content:   "Go is an open source programming language that makes it easy to build simple, reliable, and efficient software.",
	}
	_, err = wt.PostDocument("test", src)
	assert.Nil(t, err)

	type args struct {
		searchWord string
	}

	tests := []struct {
		name  string
		args  args
		found bool
	}{
		{
			name: "bi-gram match: single word",
			args: args{
				searchWord: "Go",
			},
			found: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			results, err := wt.Search(tc.args.searchWord, nil, "")
			assert.Nil(t, err)
			if tc.found {
				assert.Equal(t, 1, len(results))
				if len(results) > 0 {
					assert.Equal(t, src.Title, results[0].Title)
				}
			} else {
				assert.Equal(t, 0, len(results))
			}
		})
	}
}

func Test_PostDocumentWithDefaultLanguage(t *testing.T) {
	wt, err := NewWaterTower(context.Background(), Option{
		Index:           xid.New().String(),
		DocumentUrl:     "mem://",
		DefaultLanguage: "en",
	})
	assert.Nil(t, err)
	defer func() {
		err := wt.Close()
		assert.Nil(t, err)
	}()

	src := &Document{
		Language:  "", // if language is empty, watertower uses default language tokenizer
		Title:     "old title",
		UpdatedAt: time.Time{},
		Tags:      []string{"go", "website", "introduction"},
		Content:   "Go is an open source programming language that makes it easy to build simple, reliable, and efficient software.",
	}
	_, err = wt.PostDocument("test", src)
	assert.Nil(t, err)

	type args struct {
		searchWord string
	}

	tests := []struct {
		name  string
		args  args
		found bool
	}{
		{
			name: "english match: single word",
			args: args{
				searchWord: "programming",
			},
			found: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			results, err := wt.Search(tc.args.searchWord, nil, "en")
			assert.Nil(t, err)
			if tc.found {
				assert.Equal(t, 1, len(results))
				if len(results) > 0 {
					assert.Equal(t, src.Title, results[0].Title)
				}
			} else {
				assert.Equal(t, 0, len(results))
			}
		})
	}
}

func Test_grouping(t *testing.T) {
	tests := []struct {
		name             string
		oldGroup         []string
		newGroup         []string
		wantNewItems     []string
		wantDeletedItems []string
	}{
		{
			name:             "all new",
			oldGroup:         []string{},
			newGroup:         []string{"a", "b"},
			wantNewItems:     []string{"a", "b"},
			wantDeletedItems: nil,
		},
		{
			name:             "all delete",
			oldGroup:         []string{"a", "b"},
			newGroup:         []string{},
			wantNewItems:     nil,
			wantDeletedItems: []string{"a", "b"},
		},
		{
			name:             "all same",
			oldGroup:         []string{"a", "b"},
			newGroup:         []string{"a", "b"},
			wantNewItems:     nil,
			wantDeletedItems: nil,
		},
		{
			name:             "new and delete",
			oldGroup:         []string{"a"},
			newGroup:         []string{"b"},
			wantNewItems:     []string{"b"},
			wantDeletedItems: []string{"a"},
		},
		{
			name:             "new and same",
			oldGroup:         []string{"a"},
			newGroup:         []string{"a", "b"},
			wantNewItems:     []string{"b"},
			wantDeletedItems: nil,
		},
		{
			name:             "delete and same",
			oldGroup:         []string{"a", "b"},
			newGroup:         []string{"a"},
			wantNewItems:     nil,
			wantDeletedItems: []string{"b"},
		},
		{
			name:             "new and delete and same",
			oldGroup:         []string{"a", "b"},
			newGroup:         []string{"a", "c"},
			wantNewItems:     []string{"c"},
			wantDeletedItems: []string{"b"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotNewItems, gotDeletedItems := groupingTags(tt.oldGroup, tt.newGroup)
			assert.EqualValues(t, tt.wantNewItems, gotNewItems)
			assert.EqualValues(t, tt.wantDeletedItems, gotDeletedItems)
		})
	}
}

func Test_AddDocumentToTag(t *testing.T) {
	wt, err := NewWaterTower(context.Background(), Option{
		Index:       xid.New().String(),
		DocumentUrl: "mem://",
	})
	assert.Nil(t, err)
	defer func() {
		err := wt.Close()
		assert.Nil(t, err)
	}()

	err = wt.addTagToDocumentID("tag", 10)
	assert.Nil(t, err)

	err = wt.addTagToDocumentID("tag", 14)
	assert.Nil(t, err)

	err = wt.addTagToDocumentID("tag", 12)
	assert.Nil(t, err)

	tags, err := wt.FindTags("tag")
	assert.Nil(t, err)
	tag := tags[0]
	assert.Equal(t, "tag", tag.ID)
	assert.EqualValues(t, []uint32{10, 12, 14}, tag.DocumentIDs)
}

func Test_RemoveDocumentFromTag(t *testing.T) {
	wt, err := NewWaterTower(context.Background(), Option{
		Index:       xid.New().String(),
		DocumentUrl: "mem://",
	})
	assert.Nil(t, err)
	defer func() {
		err := wt.Close()
		assert.Nil(t, err)
	}()

	wt.addTagToDocumentID("tag", 10)
	wt.addTagToDocumentID("tag", 12)

	// 12, 10 -> 10
	err = wt.RemoveDocumentFromTag("tag", 12)
	assert.Nil(t, err)

	tags, err := wt.FindTags("tag")
	assert.Nil(t, err)
	tag := tags[0]
	assert.Equal(t, "tag", tag.ID)
	assert.EqualValues(t, []uint32{10}, tag.DocumentIDs)

	// 10 -> removed
	err = wt.RemoveDocumentFromTag("tag", 10)
	assert.Nil(t, err)

	tags, err = wt.FindTags("tag")
	assert.Error(t, err)
	assert.Equal(t, 0, len(tags))
}

func Test_AddDocumentToToken(t *testing.T) {
	wt, err := NewWaterTower(context.Background(), Option{
		Index:       xid.New().String(),
		DocumentUrl: "mem://",
	})
	assert.Nil(t, err)
	defer func() {
		err := wt.Close()
		assert.Nil(t, err)
	}()

	err = wt.addDocumentToToken("token", 10, []uint32{10, 20, 30})
	assert.Nil(t, err)

	err = wt.addDocumentToToken("token", 14, []uint32{10, 20, 30})
	assert.Nil(t, err)

	err = wt.addDocumentToToken("token", 12, []uint32{10, 20, 30})
	assert.Nil(t, err)

	tokens, err := wt.FindTokens("token")
	assert.Nil(t, err)
	if len(tokens) > 0 {
		token := tokens[0]
		assert.Equal(t, "token", token.Word)
		postingMap := token.toPostingMap()
		assert.EqualValues(t, []uint32{10, 20, 30}, postingMap[10].Positions)
		assert.EqualValues(t, []uint32{10, 20, 30}, postingMap[12].Positions)
		assert.EqualValues(t, []uint32{10, 20, 30}, postingMap[14].Positions)
	}
}

func Test_RemoveDocumentFromToken(t *testing.T) {
	wt, err := NewWaterTower(context.Background(), Option{
		Index:       xid.New().String(),
		DocumentUrl: "mem://",
	})
	assert.Nil(t, err)
	defer func() {
		err := wt.Close()
		assert.Nil(t, err)
	}()

	wt.addDocumentToToken("token", 10, []uint32{10, 12, 14})
	wt.addDocumentToToken("token", 12, []uint32{10, 12, 14})

	// 12, 10 -> 10
	err = wt.removeDocumentFromToken("token", 12)
	assert.Nil(t, err)

	tokens, err := wt.FindTokens("token")
	assert.Nil(t, err)
	assert.Equal(t, "token", tokens[0].Word)
	postingMap := tokens[0].toPostingMap()
	assert.EqualValues(t, []uint32{10, 12, 14}, postingMap[10].Positions)

	// 10 -> removed
	err = wt.removeDocumentFromToken("token", 10)
	assert.Nil(t, err)

	tokens, err = wt.FindTokens("token")
	assert.Nil(t, err)
	assert.Equal(t, 1, len(tokens))
	assert.False(t, tokens[0].Found)
}

func TestFindTokens(t *testing.T) {
	wt, err := NewWaterTower(context.Background(), Option{
		Index:       xid.New().String(),
		DocumentUrl: "mem://",
	})
	assert.Nil(t, err)
	defer func() {
		err := wt.Close()
		assert.Nil(t, err)
	}()

	wt.addDocumentIDToToken("test1", 1, []uint32{10, 20})
	wt.addDocumentIDToToken("test2", 1, []uint32{10, 20})
	wt.addDocumentIDToToken("test3", 1, []uint32{10, 20})
	wt.addDocumentIDToToken("test4", 1, []uint32{10, 20})

	tokens, err := wt.FindTokens("test1", "test2", "test3", "test4")
	assert.Nil(t, err)
	t.Log(tokens)
	assert.Equal(t, 4, len(tokens))
	assert.Equal(t, "test1", tokens[0].Word)
	assert.Equal(t, uint32(1), tokens[0].Postings[0].DocumentID)
	assert.Equal(t, "test2", tokens[1].Word)
	assert.Equal(t, uint32(1), tokens[1].Postings[0].DocumentID)
	assert.Equal(t, "test3", tokens[2].Word)
	assert.Equal(t, uint32(1), tokens[2].Postings[0].DocumentID)
	assert.Equal(t, "test4", tokens[3].Word)
	assert.Equal(t, uint32(1), tokens[3].Postings[0].DocumentID)
}
