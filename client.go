package watertower

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/shibukawa/gocloudurls"
	"gocloud.dev/docstore"
	"gocloud.dev/pubsub"
)

type Client struct {
	ctx       context.Context
	documents *docstore.Collection
	docKeys   *docstore.Collection
	docIds    *docstore.Collection
	tokens    *docstore.Collection
	tags      *docstore.Collection
	fanOut    func(msg *pubsub.Message) error
}

var dummyFanOut = func(message *pubsub.Message) error { return nil }

func localFilePath(filename, folder string) string {
	if folder == "" {
		return ""
	}
	return filepath.Join(folder, filename)
}

func (w WaterTower) SearchClient(ctx context.Context) (*Client, error) {
	var errors []string
	result := &Client{
		ctx: ctx,
	}
	configCollection := func(collectionName, keyName, fileName string) *docstore.Collection {
		url, err := gocloudurls.NormalizeDocStoreURL(w.documentUrl, gocloudurls.Option{
			Collection: collectionName,
			KeyName:    keyName,
			FileName:   localFilePath(fileName, w.localFolder),
		})
		if err != nil {
			errors = append(errors, err.Error())
			return nil
		} else {
			collection, err := docstore.OpenCollection(ctx, url)
			if err != nil {
				errors = append(errors, err.Error())
			}
			return collection
		}
	}
	result.documents = configCollection(w.collectionPrefix+"documents", "id", "documents.db")
	result.docKeys = configCollection(w.collectionPrefix+"dockeys", "unique_key", "dockeys.db")
	result.tokens = configCollection(w.collectionPrefix+"tokens", "word", "tokens.db")
	result.tags = configCollection(w.collectionPrefix+"tags", "tag", "tags.db")
	fanOutUrl, err := gocloudurls.NormalizePubSubURL(w.fanOutUrl)
	if err != nil {
		errors = append(errors, err.Error())
	} else {
		topic, err := pubsub.OpenTopic(ctx, fanOutUrl)
		if err != nil {
			errors = append(errors, err.Error())
		} else {
			result.fanOut = func(msg *pubsub.Message) error {
				return topic.Send(ctx, msg)
			}
		}
	}
	if len(errors) > 0 {
		return nil, fmt.Errorf("Can't open collections :%s", strings.Join(errors, ", "))
	}
	return result, nil
}

func (c *Client) Close() (errors []error) {
	err := c.documents.Close()
	if err != nil {
		errors = append(errors, err)
	}
	err = c.docKeys.Close()
	if err != nil {
		errors = append(errors, err)
	}
	err = c.tokens.Close()
	if err != nil {
		errors = append(errors, err)
	}
	err = c.tags.Close()
	if err != nil {
		errors = append(errors, err)
	}
	return
}
