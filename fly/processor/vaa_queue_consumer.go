package processor

import (
	"context"

	"github.com/wormhole-foundation/wormhole-explorer/fly/internal/metrics"
	"github.com/wormhole-foundation/wormhole-explorer/fly/queue"
	"github.com/wormhole-foundation/wormhole-explorer/fly/storage"

	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

// VAAQueueConsumeFunc is a function to obtain messages from a queue
type VAAQueueConsumeFunc func(context.Context) <-chan queue.Message

// VAAQueueConsumer represents a VAA queue consumer.
type VAAQueueConsumer struct {
	consume    VAAQueueConsumeFunc
	repository *storage.Repository
	notifyFunc VAANotifyFunc
	metrics    metrics.Metrics
	logger     *zap.Logger
}

// NewVAAQueueConsumer creates a new VAA queue consumer instances.
func NewVAAQueueConsumer(
	consume VAAQueueConsumeFunc,
	repository *storage.Repository,
	notifyFunc VAANotifyFunc,
	metrics metrics.Metrics,
	logger *zap.Logger) *VAAQueueConsumer {
	return &VAAQueueConsumer{
		consume:    consume,
		repository: repository,
		notifyFunc: notifyFunc,
		metrics:    metrics,
		logger:     logger,
	}
}

// Start consumes messages from VAA queue and store those messages in a repository.
func (c *VAAQueueConsumer) Start(ctx context.Context) {
	go func() {
		for msg := range c.consume(ctx) {
			v, err := vaa.Unmarshal(msg.Data())
			if err != nil {
				c.logger.Error("Error unmarshalling vaa", zap.Error(err))
				msg.Failed()
				continue
			}

			if msg.IsExpired() {
				c.logger.Warn("Message with vaa expired", zap.String("id", v.MessageID()))
				msg.Failed()
				continue
			}

			c.metrics.IncVaaConsumedFromQueue(v.EmitterChain)

			err = c.repository.UpsertVaa(ctx, v, msg.Data())
			if err != nil {
				c.logger.Error("Error inserting vaa in repository",
					zap.String("id", v.MessageID()),
					zap.Error(err))
				msg.Failed()
				continue
			}

			err = c.notifyFunc(ctx, v, msg.Data())
			if err != nil {
				c.metrics.IncMaxSequenceCacheError(v.EmitterChain)
				c.logger.Error("Error notifying vaa",
					zap.String("id", v.MessageID()),
					zap.Error(err))
				msg.Failed()
				continue
			}

			msg.Done(ctx)
			c.logger.Info("Vaa save in repository", zap.String("id", v.MessageID()))
		}
	}()
}
