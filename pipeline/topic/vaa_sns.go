package topic

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/wormhole-foundation/wormhole-explorer/common/client/alert"
	pipelineAlert "github.com/wormhole-foundation/wormhole-explorer/pipeline/internal/alert"
	"github.com/wormhole-foundation/wormhole-explorer/pipeline/internal/metrics"
	"github.com/wormhole-foundation/wormhole-explorer/pipeline/internal/sns"
	"go.uber.org/zap"
)

// SQS represents a VAA queue in SNS.
type SNS struct {
	producer    *sns.Producer
	alertClient alert.AlertClient
	metrics     metrics.Metrics
	logger      *zap.Logger
}

// NewVAASNS creates a VAA topic in SNS instances.
func NewVAASNS(producer *sns.Producer, alertClient alert.AlertClient, metrics metrics.Metrics, logger *zap.Logger) *SNS {
	s := &SNS{
		producer:    producer,
		alertClient: alertClient,
		metrics:     metrics,
		logger:      logger,
	}
	return s
}

// Publish sends the message to a SNS topic.
func (s *SNS) Publish(ctx context.Context, message *Event) error {
	body, err := json.Marshal(message)
	if err != nil {
		return err
	}

	groupID := fmt.Sprintf("%d/%s", message.ChainID, message.EmitterAddress)
	s.logger.Debug("Publishing message", zap.String("groupID", groupID))
	err = s.producer.SendMessage(ctx, groupID, message.ID, string(body))
	if err == nil {
		s.metrics.IncVaaSendNotification(message.ChainID)
	} else {
		// Alert error pushing event.
		alertContext := alert.AlertContext{
			Details: map[string]string{
				"groupID":   groupID,
				"messageID": message.ID,
			},
			Error: err,
		}
		s.alertClient.CreateAndSend(ctx, pipelineAlert.ErrorPushEventSNS, alertContext)
	}
	return err
}
