package operations

import (
	"strconv"
	"strings"
	"time"

	"github.com/wormhole-foundation/wormhole-explorer/api/handlers/operations"
	"github.com/wormhole-foundation/wormhole-explorer/api/internal/errors"
	"github.com/wormhole-foundation/wormhole-explorer/common/domain"
	"github.com/wormhole-foundation/wormhole-explorer/common/utils"

	sdk "github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

// OperationResponse definition.
type OperationResponse struct {
	ID             string         `json:"id"`
	EmitterChain   sdk.ChainID    `json:"emitterChain"`
	EmitterAddress EmitterAddress `json:"emitterAddress"`
	Sequence       string         `json:"sequence"`
	Vaa            []byte         `json:"vaa,omitempty"`
	Content        *Content       `json:"content,omitempty"`
	SourceChain    *SourceChain   `json:"sourceChain,omitempty"`
	TargetChain    *TargetChain   `json:"targetChain,omitempty"`
	Data           map[string]any `json:"data,omitempty"`
}

// EmitterAddress definition.
type EmitterAddress struct {
	Hex    string `json:"hex,omitempty"`
	Native string `json:"native,omitempty"`
}

// Content definition.
type Content struct {
	Payload                map[string]any                     `json:"payload,omitempty"`
	StandardizedProperties *operations.StandardizedProperties `json:"standarizedProperties,omitempty"`
}

// SourceChain definition.
type SourceChain struct {
	ChainId     sdk.ChainID `json:"chainId"`
	Timestamp   *time.Time  `json:"timestamp"`
	Transaction Transaction `json:"transaction"`
	From        string      `json:"from"`
	Status      string      `json:"status"`
	Data        *Data       `json:"attribute,omitempty"`
}

// TxHash definition.
type Transaction struct {
	TxHash       string  `json:"txHash"`
	SecondTxHash *string `json:"secondTxHash,omitempty"`
}

// TargetChain definition.
type TargetChain struct {
	ChainId     sdk.ChainID `json:"chainId"`
	Timestamp   *time.Time  `json:"timestamp"`
	Transaction Transaction `json:"transaction"`
	Status      string      `json:"status"`
	From        string      `json:"from"`
	To          string      `json:"to"`
}

// Data represents a custom attribute for a origin transaction.
type Data struct {
	Type  string         `bson:"type" json:"type"`
	Value map[string]any `bson:"value" json:"value"`
}

type ListOperationResponse struct {
	Operations []*OperationResponse `json:"operations"`
	Match      string               `json:"matched"`
}

// toOperationResponse converts an operations.OperationDto to an OperationResponse.
func toOperationResponse(operation *operations.OperationDto, log *zap.Logger) (*OperationResponse, error) {
	// Get emitter chain, address and sequence from operation.
	chainID, address, sequence, err := getChainEmitterSequence(operation)
	if err != nil {
		log.Error("Error parsing chainId, address, sequence from operation ID",
			zap.Error(err),
			zap.String("operationID", operation.ID))
		return nil, err
	}

	// Get emitter native address from chainID and address.
	emitterNativeAddress, err := domain.TranslateEmitterAddress(chainID, address)
	if err != nil {
		log.Warn("failed to translate emitter address",
			zap.Stringer("chain", chainID),
			zap.String("address", address),
			zap.Error(err),
		)
	}

	// Get rawVAA from operation.
	var rawVAA []byte
	if operation.Vaa != nil {
		rawVAA = operation.Vaa.Vaa
	}

	// Get content from operation.
	var content Content
	if len(operation.Payload) > 0 || operation.StandardizedProperties != nil {
		content = Content{
			Payload:                operation.Payload,
			StandardizedProperties: operation.StandardizedProperties,
		}
	}

	// Get sourceChain and targetChain events
	sourceChain, targetChain := getChainEvents(chainID, operation)
	r := OperationResponse{
		ID:           operation.ID,
		EmitterChain: chainID,
		EmitterAddress: EmitterAddress{
			Hex:    address,
			Native: emitterNativeAddress,
		},
		Sequence:    sequence,
		Vaa:         rawVAA,
		Content:     &content,
		Data:        getAdditionalData(operation),
		SourceChain: sourceChain,
		TargetChain: targetChain,
	}

	return &r, nil
}

// getChainEmitterSequence returns the chainID, address, sequence for the given operation.
func getChainEmitterSequence(operation *operations.OperationDto) (sdk.ChainID, string, string, error) {
	if operation.Vaa != nil {
		return operation.Vaa.EmitterChain, operation.Vaa.EmitterAddr, operation.Vaa.Sequence, nil
	} else {
		// Get emitter chain, address, sequence by operation ID.
		id := strings.Split(operation.ID, "/")
		if len(id) != 3 {
			return 0, "", "", errors.ErrInternalError
		}
		chainID, err := strconv.ParseUint(id[0], 10, 16)
		if err != nil {
			return 0, "", "", err
		}
		return sdk.ChainID(chainID), id[1], id[2], nil
	}
}

func getAdditionalData(operation *operations.OperationDto) map[string]interface{} {
	ok := operation.Symbol == "" && operation.TokenAmount == "" && operation.UsdAmount == ""
	if ok {
		return nil
	}
	return map[string]interface{}{
		"symbol":      operation.Symbol,
		"tokenAmount": operation.TokenAmount,
		"usdAmount":   operation.UsdAmount,
	}
}

// getChainEvents returns the sourceChain and targetChain events for the given operation.
func getChainEvents(chainID sdk.ChainID, operation *operations.OperationDto) (*SourceChain, *TargetChain) {
	if operation.SourceTx == nil && operation.DestinationTx == nil {
		return nil, nil
	}

	// if len(operation.GlobalTransations) == 0 {
	// 	return nil, nil
	// }

	// build sourceChain
	var sourceChain *SourceChain
	if operation.SourceTx != nil {
		var data *Data
		if operation.SourceTx.Attribute != nil {
			data = &Data{
				Type:  operation.SourceTx.Attribute.Type,
				Value: operation.SourceTx.Attribute.Value,
			}
		}

		// transactions
		var secondTxHash *string
		if data != nil {
			attributeTxHash, ok := data.Value["originTxHash"]
			if ok {
				txHash, ok := attributeTxHash.(string)
				if ok {
					secondTxHash = &txHash
				}
			}
		}
		transaction := Transaction{
			TxHash:       operation.SourceTx.TxHash,
			SecondTxHash: secondTxHash,
		}

		sourceChain = &SourceChain{
			ChainId:     chainID,
			Timestamp:   operation.SourceTx.Timestamp,
			Transaction: transaction,
			From:        operation.SourceTx.From,
			Status:      operation.SourceTx.Status,
			Data:        data,
		}
	}

	// build targetChain
	var targetChain *TargetChain
	if operation.DestinationTx != nil {
		targetChain = &TargetChain{
			ChainId:   operation.DestinationTx.ChainID,
			Timestamp: operation.DestinationTx.Timestamp,
			Transaction: Transaction{
				TxHash: operation.DestinationTx.TxHash,
			},
			Status: operation.DestinationTx.Status,
			From:   operation.DestinationTx.From,
			To:     operation.DestinationTx.To,
		}
	}

	return sourceChain, targetChain
}

func toListOperationResponse(operations []*operations.OperationDto, q string, log *zap.Logger) ListOperationResponse {
	response := ListOperationResponse{
		Operations: make([]*OperationResponse, 0, len(operations)),
	}

	for i := range operations {
		r, err := toOperationResponse(operations[i], log)
		if err == nil {
			response.Operations = append(response.Operations, r)
		}
	}

	response.Match = buildMatchedField(response, q)
	return response
}

func buildMatchedField(operations ListOperationResponse, q string) string {
	if q == "" {
		return ""
	}
	if len(operations.Operations) == 0 {
		return ""
	}
	operation := operations.Operations[0]
	if operation.ID == q {
		return "vaaId"
	}

	// format q to match values
	qHexa := strings.ToLower(q)
	if !utils.StartsWith0x(q) {
		qHexa = "0x" + strings.ToLower(qHexa)
	}

	// matched by sourceChain txHash
	if operation.SourceChain != nil {
		if operation.SourceChain.Transaction.TxHash == q || operation.SourceChain.Transaction.TxHash == qHexa {
			return "txHash"
		}

		if operation.SourceChain.Data != nil {
			if operation.SourceChain.Data.Value["OriginTxHash"] == q || operation.SourceChain.Data.Value["originTxHash"] == qHexa {
				return "txHash"
			}
		}
	}

	// matched by targetChain txHash
	if operation.TargetChain != nil {
		if operation.TargetChain.Transaction.TxHash == q || operation.TargetChain.Transaction.TxHash == qHexa {
			return "txHash"
		}
	}

	// matched by sourceChain from address
	if operation.SourceChain != nil {
		if operation.SourceChain.From == q || operation.SourceChain.From == qHexa {
			return "address"
		}
	}

	// matched by standardizedProperties to address
	if operation.Content.StandardizedProperties.ToAddress == q || operation.Content.StandardizedProperties.ToAddress == qHexa {
		return "address"
	}

	return ""
}
