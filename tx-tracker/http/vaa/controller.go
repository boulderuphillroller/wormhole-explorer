package vaa

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/wormhole-foundation/wormhole-explorer/txtracker/config"
	"github.com/wormhole-foundation/wormhole-explorer/txtracker/consumer"
	sdk "github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

// Controller definition.
type Controller struct {
	logger              *zap.Logger
	vaaRepository       *Repository
	repository          *consumer.Repository
	rpcProviderSettings *config.RpcProviderSettings
	p2pNetwork          string
}

// NewController creates a Controller instance.
func NewController(vaaRepository *Repository, repository *consumer.Repository, rpcProviderSettings *config.RpcProviderSettings, p2pNetwork string, logger *zap.Logger) *Controller {
	return &Controller{vaaRepository: vaaRepository, repository: repository, rpcProviderSettings: rpcProviderSettings, p2pNetwork: p2pNetwork, logger: logger}
}

func (c *Controller) Process(ctx *fiber.Ctx) error {
	payload := struct {
		ID string `json:"id"`
	}{}

	if err := ctx.BodyParser(&payload); err != nil {
		return err
	}

	c.logger.Info("Processing VAA from endpoint", zap.String("id", payload.ID))

	v, err := c.vaaRepository.FindById(ctx.Context(), payload.ID)
	if err != nil {
		return err
	}

	vaa, err := sdk.Unmarshal(v.Vaa)
	if err != nil {
		return err
	}

	p := &consumer.ProcessSourceTxParams{
		Timestamp: &vaa.Timestamp,
		VaaId:     vaa.MessageID(),
		ChainId:   vaa.EmitterChain,
		Emitter:   vaa.EmitterAddress.String(),
		Sequence:  strconv.FormatUint(vaa.Sequence, 10),
		TxHash:    v.TxHash,
		Overwrite: true,
	}

	result, err := consumer.ProcessSourceTx(ctx.Context(), c.logger, c.rpcProviderSettings, c.repository, p, c.p2pNetwork)
	if err != nil {
		return err
	}

	return ctx.JSON(struct {
		Result any `json:"result"`
	}{Result: result})
}
