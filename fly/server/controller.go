package server

import (
	"context"
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/wormhole-foundation/wormhole-explorer/common/client/alert"
	flyAlert "github.com/wormhole-foundation/wormhole-explorer/fly/internal/alert"
	"github.com/wormhole-foundation/wormhole-explorer/fly/internal/health"
	"github.com/wormhole-foundation/wormhole-explorer/fly/internal/sqs"
	"github.com/wormhole-foundation/wormhole-explorer/fly/storage"
	"go.uber.org/zap"
)

// Controller definition.
type Controller struct {
	guardianCheck *health.GuardianCheck
	repository    *storage.Repository
	consumer      *sqs.Consumer
	isLocal       bool
	logger        *zap.Logger
	alertClient   alert.AlertClient
}

// NewController creates a Controller instance.
func NewController(gCheck *health.GuardianCheck, repo *storage.Repository, consumer *sqs.Consumer, isLocal bool, alertClient alert.AlertClient, logger *zap.Logger) *Controller {
	return &Controller{guardianCheck: gCheck, repository: repo, consumer: consumer, isLocal: isLocal, alertClient: alertClient, logger: logger}
}

// HealthCheck handler for the endpoint /health.
func (c *Controller) HealthCheck(ctx *fiber.Ctx) error {
	// check guardian gossip network is ready.
	guardianErr := c.checkGuardianStatus(ctx.Context())
	if guardianErr != nil {
		c.logger.Error("Health check failed", zap.Error(guardianErr))
		// send alert when exists an error saving ptth vaa.
		alertContext := alert.AlertContext{
			Error: guardianErr,
		}
		c.alertClient.CreateAndSend(ctx.Context(), flyAlert.ErrorGuardianNoActivity, alertContext)
		return ctx.Status(fiber.StatusInternalServerError).JSON(struct {
			Status string `json:"status"`
			Error  string `json:"error"`
		}{Status: "NO", Error: guardianErr.Error()})
	}
	return ctx.JSON(struct {
		Status string `json:"status"`
	}{Status: "OK"})
}

// ReadyCheck handler for the endpoint /ready
func (c *Controller) ReadyCheck(ctx *fiber.Ctx) error {
	// check mongo db is ready.
	mongoErr := c.checkMongoStatus(ctx.Context())
	if mongoErr != nil {
		c.logger.Error("Ready check failed", zap.Error(mongoErr))
		return ctx.Status(fiber.StatusInternalServerError).JSON(struct {
			Ready string `json:"ready"`
			Error string `json:"error"`
		}{Ready: "NO", Error: mongoErr.Error()})
	}
	// check aws SQS is ready.
	queueErr := c.checkQueueStatus(ctx.Context())
	if queueErr != nil {
		c.logger.Error("Ready check failed", zap.Error(queueErr))
		return ctx.Status(fiber.StatusInternalServerError).JSON(struct {
			Ready string `json:"ready"`
			Error string `json:"error"`
		}{Ready: "NO", Error: queueErr.Error()})
	}
	// return success response.
	return ctx.Status(fiber.StatusOK).JSON(struct {
		Ready string `json:"ready"`
	}{Ready: "OK"})
}

func (c *Controller) checkMongoStatus(ctx context.Context) error {
	mongoStatus, err := c.repository.GetMongoStatus(ctx)
	if err != nil {
		return err
	}

	// check mongo server status
	mongoStatusCheck := (mongoStatus.Ok == 1 && mongoStatus.Pid > 0 && mongoStatus.Uptime > 0)
	if !mongoStatusCheck {
		return errors.New("mongo invalid status")
	}

	// check mongo connections
	if mongoStatus.Connections.Available <= 0 {
		return errors.New("mongo hasn't available connections")
	}
	return nil
}

func (c *Controller) checkQueueStatus(ctx context.Context) error {
	// vaa queue handle in memory [local enviroment]
	if c.isLocal {
		return nil
	}
	// get queue attributes
	queueAttributes, err := c.consumer.GetQueueAttributes(ctx)
	if err != nil {
		return err
	}
	if queueAttributes == nil {
		return errors.New("can't get attributes for sqs")
	}

	// check queue created
	createdTimestamp := queueAttributes.Attributes["CreatedTimestamp"]
	if createdTimestamp == "" {
		return errors.New("sqs queue hasn't been created")
	}
	return nil
}

func (c *Controller) checkGuardianStatus(ctx context.Context) error {
	isAlive := c.guardianCheck.IsAlive()
	if !isAlive {
		return errors.New("guardian healthcheck not arrive in time")
	}
	return nil
}
