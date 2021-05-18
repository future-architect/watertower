package watertower

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/future-architect/gocloudurls"
	"github.com/shibukawa/cloudcounter"
	"gocloud.dev/docstore"
	_ "gocloud.dev/pubsub/mempubsub"
)

type WaterTower struct {
	ctx             context.Context
	collection      *docstore.Collection
	counter         *cloudcounter.Counter
	close           sync.Once
	defaultLanguage string
}

type Option struct {
	DocumentUrl        string
	CollectionOpener   func(ctx context.Context, opt Option) (*docstore.Collection, error)
	LocalFolder        string
	Index              string
	CounterConcurrency int
	TitleScoreRatio    float64
	DefaultLanguage    string
}

const (
	documentID    cloudcounter.CounterKey = "document_id"
	documentCount                         = "document_count"
)

func DefaultCollectionOpener(ctx context.Context, opt Option) (*docstore.Collection, error) {
	url, err := defaultCollectionURL(opt)
	if err != nil {
		return nil, fmt.Errorf("Can't parse document URL: %w", err)
	}
	collection, err := docstore.OpenCollection(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("Can't open collection: %w", err)
	}
	return collection, nil
}

// DefaultCollectionURL returns collection URL. This function is for help message or debugging
func DefaultCollectionURL(opt ...Option) (string, error) {
	option, err := initOpt(opt...)
	if err != nil {
		return "", err
	}
	if option.CollectionOpener == nil {
		return defaultCollectionURL(option)
	}
	return "", errors.New("Can't generate collection URL for custom opener")
}

func defaultCollectionURL(opt Option) (string, error) {
	var filename string
	if opt.LocalFolder != "" {
		filename = filepath.Join(opt.LocalFolder, "watertower.db")
	}
	if opt.Index == "" {
		opt.Index = "index"
	}
	url, err := gocloudurls.NormalizeDocStoreURL(opt.DocumentUrl, gocloudurls.Option{
		Collection: opt.Index,
		KeyName:    "id",
		FileName:   filename,
	})
	return url, err
}

func initOpt(opt ...Option) (Option, error) {
	var option Option
	if len(opt) > 0 {
		option = opt[0]
	}
	if option.DocumentUrl == "" {
		option.DocumentUrl = os.Getenv("WATERTOWER_DOCUMENT_URL")
	}
	if option.DocumentUrl == "" {
		return option, errors.New("NewInvertedIndex: DocumentUrl is missign")
	}
	if option.TitleScoreRatio == 0.0 {
		option.TitleScoreRatio = 5.0
	}
	if option.CounterConcurrency == 0 {
		option.CounterConcurrency = 5
	}
	return option, nil
}

// NewWaterTower initialize WaterTower instance
func NewWaterTower(ctx context.Context, opt ...Option) (*WaterTower, error) {
	option, err := initOpt(opt...)
	if err != nil {
		return nil, err
	}
	if option.CollectionOpener == nil {
		option.CollectionOpener = DefaultCollectionOpener
	}
	result := &WaterTower{
		ctx: ctx,
		defaultLanguage: option.DefaultLanguage,
	}
	collection, err := option.CollectionOpener(ctx, option)
	if err != nil {
		return nil, err
	}
	result.collection = collection
	result.counter = cloudcounter.NewCounter(collection, cloudcounter.Option{
		Concurrency: option.CounterConcurrency,
		Prefix:      option.Index + "c",
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

// Close closes document store connection. Some docstore (at least memdocstore) needs Close() to store file
func (wt *WaterTower) Close() (err error) {
	wt.close.Do(func() {
		err = wt.collection.Close()
	})
	return
}
