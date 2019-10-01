package watertower

import (
	"context"
	"testing"
	"time"

	"github.com/rs/xid"
	_ "github.com/shibukawa/watertower/nlp/english"
	"github.com/stretchr/testify/assert"
	_ "gocloud.dev/docstore/memdocstore"
	"gocloud.dev/pubsub"
)

func TestClient_PostDocument_IncrementID(t *testing.T) {
	wt, err := NewWaterTower(Option{
		CollectionPrefix: xid.New().String(),
		DocumentUrl:      "mem://",
	})
	assert.Nil(t, err)
	defer wt.Close()
	client, err := wt.SearchClient(context.Background())
	assert.Nil(t, err)
	defer func() {
		err := client.Close()
		assert.Nil(t, err)
	}()

	docID, err := client.postDocumentKey("key")
	assert.Nil(t, err)
	assert.Equal(t, uint32(1), docID)

	docID, err = client.postDocumentKey("new-key")
	assert.Nil(t, err)
	assert.Equal(t, uint32(2), docID)
}

func TestClient_PostDocument(t *testing.T) {
	wt, err := NewWaterTower(Option{
		CollectionPrefix: xid.New().String(),
		DocumentUrl:      "mem://",
	})
	assert.Nil(t, err)
	defer wt.Close()
	client, err := wt.SearchClient(context.Background())
	assert.Nil(t, err)
	defer func() {
		err := client.Close()
		assert.Nil(t, err)
	}()

	doc := &Document{
		Language:  "en",
		Title:     "old title",
		UpdatedAt: time.Time{},
		Tags:      []string{"go", "website", "introduction"},
		Content:   "Go is an open source programming language that makes it easy to build simple, reliable, and efficient software.",
		Summary:   "Summary",
	}
	docID, err := client.postDocumentKey("test")
	assert.Nil(t, err)
	oldDoc, err := client.postDocument(docID, "test", doc)
	assert.Nil(t, err)
	assert.Nil(t, oldDoc)

	loadedDoc, err := client.FindDocument(docID)
	assert.Nil(t, err)
	assert.Equal(t, "old title", loadedDoc.Title)

	doc.Title = "new title"
	oldDoc, err = client.postDocument(docID, "test", doc)
	assert.Nil(t, err)
	assert.Equal(t, "old title", oldDoc.Title)

	loadedDoc, err = client.FindDocumentByKey("test")
	assert.Nil(t, err)
	assert.Equal(t, "new title", loadedDoc.Title)
}

func TestClient_PostDocument_FanOut(t *testing.T) {
	wt, err := NewWaterTower(Option{
		CollectionPrefix: xid.New().String(),
		DocumentUrl:      "mem://",
	})
	assert.Nil(t, err)
	defer wt.Close()
	client, err := wt.SearchClient(context.Background())
	assert.Nil(t, err)
	defer func() {
		err := client.Close()
		assert.Nil(t, err)
	}()
	var tagCount int
	var tokenCount int
	client.fanOut = func(m *pubsub.Message) error {
		switch m.Metadata["target"] {
		case "tag":
			tagCount++
		case "token":
			tokenCount++
		}
		return nil
	}

	doc := &Document{
		Title:    "test",
		Language: "en",
		Tags:     []string{"go", "website", "introduction"},
		Content:  "Go is an open source programming language that makes it easy to build simple, reliable, and efficient software.",
	}
	docID, err := client.PostDocument("test", doc)
	assert.Equal(t, uint32(1), docID)
	assert.Nil(t, err)

	// fanOut function is called
	assert.Equal(t, 3, tagCount)
	assert.Greater(t, 20, tokenCount)
}

func TestClient_RemoveDocument(t *testing.T) {
	wt, err := NewWaterTower(Option{
		CollectionPrefix: xid.New().String(),
		DocumentUrl:      "mem://",
	})
	assert.Nil(t, err)
	defer wt.Close()
	client, err := wt.SearchClient(context.Background())
	assert.Nil(t, err)
	defer func() {
		err := client.Close()
		assert.Nil(t, err)
	}()
	var tagCount int
	var tokenCount int
	client.fanOut = func(m *pubsub.Message) error {
		if m.Metadata["action"] != "delete" {
			return nil
		}
		switch m.Metadata["target"] {
		case "tag":
			tagCount++
		case "token":
			tokenCount++
		}
		return nil
	}
	doc := &Document{
		Title:    "test",
		Language: "en",
		Tags:     []string{"go", "website", "introduction"},
		Content:  "Go is an open source programming language that makes it easy to build simple, reliable, and efficient software.",
	}
	client.PostDocument("test", doc)

	err = client.RemoveDocument("test")
	assert.Nil(t, err)
	// fanOut function is called
	assert.Equal(t, 3, tagCount)
	assert.Greater(t, 20, tokenCount)
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

func TestClient_AddDocumentToTag(t *testing.T) {
	wt, err := NewWaterTower(Option{
		CollectionPrefix: xid.New().String(),
		DocumentUrl:      "mem://",
	})
	assert.Nil(t, err)
	defer wt.Close()
	client, err := wt.SearchClient(context.Background())
	assert.Nil(t, err)
	defer func() {
		err := client.Close()
		assert.Nil(t, err)
	}()

	err = client.AddDocumentToTag("tag", 10)
	assert.Nil(t, err)

	err = client.AddDocumentToTag("tag", 14)
	assert.Nil(t, err)

	err = client.AddDocumentToTag("tag", 12)
	assert.Nil(t, err)

	tag, err := client.FindTag("tag")
	assert.Nil(t, err)
	assert.Equal(t, "tag", tag.Tag)
	assert.EqualValues(t, []uint32{10, 12, 14}, tag.DocumentIDs)
}

func TestClient_RemoveDocumentFromTag(t *testing.T) {
	wt, err := NewWaterTower(Option{
		CollectionPrefix: xid.New().String(),
		DocumentUrl:      "mem://",
	})
	assert.Nil(t, err)
	defer wt.Close()
	client, err := wt.SearchClient(context.Background())
	assert.Nil(t, err)
	defer func() {
		err := client.Close()
		assert.Nil(t, err)
	}()

	client.AddDocumentToTag("tag", 10)
	client.AddDocumentToTag("tag", 12)

	// 12, 10 -> 10
	err = client.RemoveDocumentFromTag("tag", 12)
	assert.Nil(t, err)

	tag, err := client.FindTag("tag")
	assert.Nil(t, err)
	assert.Equal(t, "tag", tag.Tag)
	assert.EqualValues(t, []uint32{10}, tag.DocumentIDs)

	// 10 -> removed
	err = client.RemoveDocumentFromTag("tag", 10)
	assert.Nil(t, err)

	tag, err = client.FindTag("tag")
	assert.Error(t, err)
	assert.Nil(t, tag)
}

func TestClient_AddDocumentToToken(t *testing.T) {
	wt, err := NewWaterTower(Option{
		CollectionPrefix: xid.New().String(),
		DocumentUrl:      "mem://",
	})
	assert.Nil(t, err)
	defer wt.Close()
	client, err := wt.SearchClient(context.Background())
	assert.Nil(t, err)
	defer func() {
		err := client.Close()
		assert.Nil(t, err)
	}()

	err = client.AddDocumentToToken("token", 10, []uint32{10, 20, 30})
	assert.Nil(t, err)

	err = client.AddDocumentToToken("token", 14, []uint32{10, 20, 30})
	assert.Nil(t, err)

	err = client.AddDocumentToToken("token", 12, []uint32{10, 20, 30})
	assert.Nil(t, err)

	token, err := client.FindToken("token")
	assert.Nil(t, err)
	if token != nil {
		assert.Equal(t, "token", token.Word)
		assert.EqualValues(t, []uint32{10, 20, 30}, token.Postings[10].Positions)
		assert.EqualValues(t, []uint32{10, 20, 30}, token.Postings[12].Positions)
		assert.EqualValues(t, []uint32{10, 20, 30}, token.Postings[14].Positions)
	}
}

func TestClient_RemoveDocumentFromToken(t *testing.T) {
	wt, err := NewWaterTower(Option{
		CollectionPrefix: xid.New().String(),
		DocumentUrl:      "mem://",
	})
	assert.Nil(t, err)
	defer wt.Close()
	client, err := wt.SearchClient(context.Background())
	assert.Nil(t, err)
	defer func() {
		err := client.Close()
		assert.Nil(t, err)
	}()

	client.AddDocumentToToken("token", 10, []uint32{10, 12, 14})
	client.AddDocumentToToken("token", 12, []uint32{10, 12, 14})

	// 12, 10 -> 10
	err = client.RemoveDocumentFromToken("token", 12)
	assert.Nil(t, err)

	token, err := client.FindToken("token")
	assert.Nil(t, err)
	assert.Equal(t, "token", token.Word)
	assert.EqualValues(t, []uint32{10, 12, 14}, token.Postings[10].Positions)

	// 10 -> removed
	err = client.RemoveDocumentFromToken("token", 10)
	assert.Nil(t, err)

	token, err = client.FindToken("token")
	assert.Error(t, err)
	assert.Nil(t, token)
}
