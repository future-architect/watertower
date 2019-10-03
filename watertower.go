package watertower

import (
	"errors"
	_ "gocloud.dev/pubsub/mempubsub"
	"os"
)

type WaterTower struct {
	documentUrl      string
	localFolder      string
	collectionPrefix string
}

type Option struct {
	DocumentUrl      string
	LocalFolder      string
	CollectionPrefix string
	CustomFanOut     string
}

func NewWaterTower(opt ...Option) (*WaterTower, error) {
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
	wt := &WaterTower{
		documentUrl:      option.DocumentUrl,
		localFolder:      option.LocalFolder,
		collectionPrefix: option.CollectionPrefix,
	}
	return wt, nil
}