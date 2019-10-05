package watertower

import (
	"context"
	"errors"
	"github.com/shibukawa/gocloudurls"
	"gocloud.dev/docstore"
	_ "gocloud.dev/pubsub/mempubsub"
	"os"
	"path/filepath"
	"sync"
)

func localFilePath(filename, folder string) string {
	if folder == "" {
		return ""
	}
	return filepath.Join(folder, filename)
}

type WaterTower struct {
	ctx       context.Context
	documents *docstore.Collection
	docKeys   *docstore.Collection
	uniqueIDs *docstore.Collection
	tokens    *docstore.Collection
	tags      *docstore.Collection
	close     sync.Once
}

type Option struct {
	DocumentUrl      string
	LocalFolder      string
	CollectionPrefix string
	CustomFanOut     string
}

func NewWaterTower(ctx context.Context, opt ...Option) (*WaterTower, error) {
	var option Option
	if len(opt) > 0 {
		option = opt[0]
	}
	if option.DocumentUrl == "" {
		option.DocumentUrl = os.Getenv("WATERTOWER_DOCUMENT_URL")
	}
	if option.DocumentUrl == "" {
		return nil, errors.New("NewInvertedIndex: DocumentUrl is missign")
	}
	finalError := &CombinedError{
		Message: "Can't open collections for search engine",
	}
	result := &WaterTower{
		ctx: ctx,
	}
	configCollection := func(collectionName, keyName, fileName string) *docstore.Collection {
		url, err := gocloudurls.NormalizeDocStoreURL(option.DocumentUrl, gocloudurls.Option{
			Collection: collectionName,
			KeyName:    keyName,
			FileName:   localFilePath(fileName, option.LocalFolder),
		})
		if err != nil {
			finalError.append(err)
			return nil
		} else {
			collection, err := docstore.OpenCollection(ctx, url)
			if err != nil {
				finalError.append(err)
				return nil
			}
			return collection
		}
	}
	result.documents = configCollection(option.CollectionPrefix+"documents", "id", "documents.db")
	result.docKeys = configCollection(option.CollectionPrefix+"dockeys", "unique_key", "dockeys.db")
	result.uniqueIDs = configCollection(option.CollectionPrefix+"uniqueids", "collection", "docids.db")
	result.tokens = configCollection(option.CollectionPrefix+"tokens", "word", "tokens.db")
	result.tags = configCollection(option.CollectionPrefix+"tags", "tag", "tags.db")
	if len(finalError.Errors) > 0 {
		return nil, finalError
	}
	uniqueID := UniqueID{
		Collection: "documents",
		LatestID:   0,
	}
	err := result.uniqueIDs.Get(ctx, &uniqueID)
	if err != nil {
		result.uniqueIDs.Create(ctx, &uniqueID)
	}
	go func() {
		<-ctx.Done()
		result.Close()
	}()
	return result, nil
}

func (c *WaterTower) Close() (errors error) {
	finalError := &CombinedError{
		Message: "Can't close collections for search engine",
	}
	c.close.Do(func() {
		finalError.appendIfError(c.documents.Close())
		finalError.appendIfError(c.docKeys.Close())
		finalError.appendIfError(c.uniqueIDs.Close())
		finalError.appendIfError(c.tokens.Close())
		finalError.appendIfError(c.tags.Close())
	})
	if finalError.Errors != nil {
		return finalError
	}
	return nil
}
