package watertower

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/shibukawa/cloudcounter"
	"github.com/shibukawa/gocloudurls"
	"gocloud.dev/docstore"
	_ "gocloud.dev/pubsub/mempubsub"
)

type WaterTower struct {
	ctx        context.Context
	collection *docstore.Collection
	counter    *cloudcounter.Counter
	close      sync.Once
}

type Option struct {
	DocumentUrl        string
	CollectionOpener   func(ctx context.Context, documentURL, collectionPrefix, localFolder string) (*docstore.Collection, error)
	LocalFolder        string
	CollectionPrefix   string
	CustomFanOut       string
	CounterConcurrency int
	TitleScoreRatio    float64
}

const (
	documentID    cloudcounter.CounterKey = "document_id"
	documentCount                         = "document_count"
)

func DefaultCollectionOpener(ctx context.Context, documentURL, collectionPrefix, localFolder string) (*docstore.Collection, error) {
	var filename string
	if localFolder != "" {
		filename = filepath.Join(localFolder, "watertower.db")
	}
	url, err := gocloudurls.NormalizeDocStoreURL(documentURL, gocloudurls.Option{
		Collection: collectionPrefix + "watertower",
		KeyName:    "id",
		FileName:   filename,
	})
	if err != nil {
		return nil, fmt.Errorf("Can't parse document URL: %w", err)
	}
	collection, err := docstore.OpenCollection(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("Can't open collection: %w", err)
	}
	return collection, nil
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
	if option.CollectionOpener == nil {
		option.CollectionOpener = DefaultCollectionOpener
	}
	if option.TitleScoreRatio == 0.0 {
		option.TitleScoreRatio = 5.0
	}
	if option.CounterConcurrency == 0 {
		option.CounterConcurrency = 5
	}
	result := &WaterTower{
		ctx: ctx,
	}
	collection, err := option.CollectionOpener(ctx, option.DocumentUrl, option.CollectionPrefix, option.LocalFolder)
	if err != nil {
		return nil, err
	}
	result.collection = collection
	result.counter = cloudcounter.NewCounter(collection, cloudcounter.Option{
		Concurrency: option.CounterConcurrency,
		Prefix:      option.CollectionPrefix + "c",
	})
	err = result.counter.Register(ctx, documentID)
	if err != nil {
		return nil, err
	}
	err = result.counter.Register(ctx, documentCount)
	if err != nil {
		return nil, err
	}
	go func() {
		<-ctx.Done()
		result.Close()
	}()
	return result, nil
}

func (wt *WaterTower) Close() (err error) {
	wt.close.Do(func() {
		err = wt.collection.Close()
	})
	return
}
