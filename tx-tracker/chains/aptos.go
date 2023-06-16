package chains

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/wormhole-foundation/wormhole-explorer/txtracker/config"
)

const (
	aptosCoreContractAddress = "0x5bc11445584a763c1fa7ed39081f1b920954da14e04b32440cba863d03e19625"
)

type aptosEvent struct {
	Version uint64 `json:"version,string"`
}

type aptosTx struct {
	Timestamp uint64 `json:"timestamp,string"`
	Sender    string `json:"sender"`
	Hash      string `json:"hash"`
}

func fetchAptosTx(
	ctx context.Context,
	cfg *config.RpcProviderSettings,
	txHash string,
) (*TxDetail, error) {

	// Parse the Aptos event creation number
	creationNumber, err := strconv.ParseUint(txHash, 16, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse event creation number from Aptos tx hash: %w", err)
	}

	// Get the event from the Aptos node API.
	var events []aptosEvent
	{
		// Build the URI for the events endpoint
		uri := fmt.Sprintf("%s/accounts/%s/events/%s::state::WormholeMessageHandle/event?start=%d&limit=1",
			cfg.AptosBaseUrl,
			aptosCoreContractAddress,
			aptosCoreContractAddress,
			creationNumber,
		)

		// Query the events endpoint
		body, err := httpGet(ctx, uri)
		if err != nil {
			return nil, fmt.Errorf("failed to query events endpoint: %w", err)
		}

		// Deserialize the response
		err = json.Unmarshal(body, &events)
		if err != nil {
			return nil, fmt.Errorf("failed to parse response body from events endpoint: %w", err)
		}
	}
	if len(events) != 1 {
		return nil, fmt.Errorf("expected exactly one event, but got %d", len(events))
	}

	// Get the transacton
	var tx aptosTx
	{
		// Build the URI for the events endpoint
		uri := fmt.Sprintf("%s/transactions/by_version/%d", cfg.AptosBaseUrl, events[0].Version)

		// Query the events endpoint
		body, err := httpGet(ctx, uri)
		if err != nil {
			return nil, fmt.Errorf("failed to query transactions endpoint: %w", err)
		}

		// Deserialize the response
		err = json.Unmarshal(body, &tx)
		if err != nil {
			return nil, fmt.Errorf("failed to parse response body from transactions endpoint: %w", err)
		}
	}

	// Build the result struct and return
	TxDetail := TxDetail{
		NativeTxHash: tx.Hash,
		From:         tx.Sender,
		Timestamp:    time.UnixMicro(int64(tx.Timestamp)),
	}
	return &TxDetail, nil
}
