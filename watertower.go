package watertower

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/future-architect/gocloudurls"
	"gocloud.dev/docstore"
	_ "gocloud.dev/pubsub/mempubsub"
)

type WaterTower struct {
	storage         Storage
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

type ReadOnlyOption struct {
	Input           io.Reader
	TitleScoreRatio float64
	DefaultLanguage string
}

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
	result := &WaterTower{
		defaultLanguage: option.DefaultLanguage,
	}
	if option.DocumentUrl != "" {
		if option.CollectionOpener == nil && option.DocumentUrl != "" {
			option.CollectionOpener = DefaultCollectionOpener
		}
		c, err := option.CollectionOpener(ctx, option)
		if err != nil {
			return nil, err
		}
		s, err := newDocstoreStorage(ctx, c, option.Index, option.CounterConcurrency)
		if err != nil {
			return nil, err
		}
		result.storage = s
	} else {
		result.storage = newLocalIndex()
	}
	go func() {
		<-ctx.Done()
		result.storage.Close()
	}()
	return result, nil
}

func NewReadOnlyWaterTower(ctx context.Context, opt ...ReadOnlyOption) (*WaterTower, error) {
	var opt2 Option
	var input io.Reader
	if len(opt) > 0 {
		opt2.TitleScoreRatio = opt[0].TitleScoreRatio
		opt2.DefaultLanguage = opt[0].DefaultLanguage
		input = opt[0].Input
	}
	wt, err := NewWaterTower(ctx, opt2)
	if err != nil {
		return nil, fmt.Errorf("open WaterTower error: %w", err)
	}

	err = wt.storage.ReadIndex(input)
	if err != nil {
		return nil, fmt.Errorf("load index error: %w", err)
	}
	return wt, nil
}

// Close closes document store connection. Some docstore (at least memdocstore) needs Close() to store file
func (wt *WaterTower) Close() (err error) {
	return wt.storage.Close()
}

func (wt *WaterTower) WriteIndex(w io.Writer) error {
	return wt.storage.WriteIndex(w)
}
