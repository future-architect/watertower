package watertower

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDocStoreConflict(t *testing.T) {
	wt, err := NewWaterTower(Option{
		DocumentUrl: "mem://",
	})
	assert.Nil(t, err)
	defer wt.Close()
	client, err := wt.SearchClient(context.Background())
	client.fanOut = dummyFanOut
	assert.Nil(t, err)
	defer func() {
		err := client.Close()
		assert.Nil(t, err)
	}()
	doc := &Document{
		ID:      1,
		Title:   "test",
		Content: "test",
	}
	// first create
	err = client.documents.Create(client.ctx, doc)
	assert.Nil(t, err)
	// second create
	err = client.documents.Create(client.ctx, doc)
	assert.Error(t, err)
}

func TestDocStoreNotFound(t *testing.T) {
	wt, err := NewWaterTower(Option{
		DocumentUrl: "mem://",
	})
	assert.Nil(t, err)
	defer wt.Close()
	client, err := wt.SearchClient(context.Background())
	client.fanOut = dummyFanOut
	assert.Nil(t, err)
	defer func() {
		err := client.Close()
		assert.Nil(t, err)
	}()
	doc := &Document{
		ID: 1,
	}

	err = client.documents.Get(client.ctx, doc)
	t.Log(err)
	assert.Error(t, err)
}
