package watertower

import (
	"context"
	"errors"
	"os"

	"gocloud.dev/pubsub"
	_ "gocloud.dev/pubsub/mempubsub"
)

type WaterTower struct {
	documentUrl      string
	localFolder      string
	fanOutUrl        string
	collectionPrefix string
	cancel           context.CancelFunc
}

type Option struct {
	DocumentUrl      string
	LocalFolder      string
	FanOutUrl        string
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
	customFanOut := true
	if option.CustomFanOut == "" {
		option.CustomFanOut = os.Getenv("WATERTOWER_FAN_OUT_URL")
	}
	if option.CustomFanOut == "" {
		option.CustomFanOut = "mem://watertower_fanout"
		customFanOut = false
	}
	wt := &WaterTower{
		documentUrl:      option.DocumentUrl,
		localFolder:      option.LocalFolder,
		fanOutUrl:        option.CustomFanOut,
		collectionPrefix: option.CollectionPrefix,
	}
	if !customFanOut {
		ctx, cancel := context.WithCancel(context.Background())
		wt.cancel = cancel
		// mempubsub needs create topic first before subscribe
		_, err := pubsub.OpenTopic(ctx, option.CustomFanOut)
		if err != nil {
			return nil, err
		}
		subscription, err := pubsub.OpenSubscription(ctx, option.CustomFanOut)
		if err != nil {
			return nil, err
		}
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				default:
					msg, err := subscription.Receive(ctx)
					if err != nil {
						return
					}
					msg.Ack()
					wt.ConsumeTask(msg)
				}
			}
		}()
	}

	return wt, nil
}

func (wt *WaterTower) Close() {
	wt.cancel()
}

func (wt *WaterTower) ConsumeTask(msg *pubsub.Message) error {
	return nil
}
