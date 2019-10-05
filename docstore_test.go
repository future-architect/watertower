package watertower

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDocStoreConflict(t *testing.T) {
	wt, err := NewWaterTower(context.Background(), Option{
		DocumentUrl: "mem://",
	})
	assert.Nil(t, err)
	defer func() {
		err := wt.Close()
		assert.Nil(t, err)
	}()
	doc := &Document{
		ID:      1,
		Title:   "test",
		Content: "test",
	}
	// first create
	err = wt.documents.Create(wt.ctx, doc)
	assert.Nil(t, err)
	// second create
	err = wt.documents.Create(wt.ctx, doc)
	assert.Error(t, err)
}

func TestDocStoreNotFound(t *testing.T) {
	wt, err := NewWaterTower(context.Background(), Option{
		DocumentUrl: "mem://",
	})
	assert.Nil(t, err)
	defer func() {
		err := wt.Close()
		assert.Nil(t, err)
	}()
	doc := &Document{
		ID: 1,
	}

	err = wt.documents.Get(wt.ctx, doc)
	t.Log(err)
	assert.Error(t, err)
}

func TestDocStore_Close(t *testing.T) {
	wt, err := NewWaterTower(context.Background(), Option{
		DocumentUrl: "mem://",
	})
	doc := &Document{
		Content:  "test",
		Language: "en",
	}
	id, err := wt.PostDocument("test", doc)
	assert.Nil(t, err)

	// Close clear all data for memdocstore
	t.Log(wt.Close())

	wt2, err := NewWaterTower(context.Background(), Option{
		DocumentUrl: "mem://",
	})
	assert.Nil(t, err)
	defer wt2.Close()
	doc2 := &Document{
		ID: id,
	}
	wt2.documents.Get(context.Background(), doc2)
	assert.Equal(t, "", doc2.Content)
}
