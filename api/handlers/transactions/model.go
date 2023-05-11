package transactions

import (
	"fmt"
	"time"

	"github.com/wormhole-foundation/wormhole-explorer/api/internal/pagination"
	sdk "github.com/wormhole-foundation/wormhole/sdk/vaa"
)

type Scorecards struct {
	// Total number of VAAs emitted since the creation of the network (does not include Pyth messages).
	TotalTxCount string

	// Volume transferred through the token bridge since the creation of the network, in USD.
	TotalTxVolume string

	// Number of VAAs emitted in the last 24 hours (does not include Pyth messages).
	TxCount24h string

	// Volume transferred through the token bridge in the last 24 hours, in USD.
	Volume24h string
}

// AssetDTO is used for the return value of the function `GetTopAssetsByVolume`.
type AssetDTO struct {
	EmitterChain sdk.ChainID
	TokenChain   sdk.ChainID
	TokenAddress string
	Volume       string
}

// TopAssetsTimerange is used as an input parameter for the function `GetTopAssetsByVolume`.
type TopAssetsTimerange string

const (
	TopAssetsTimerange7Days  TopAssetsTimerange = "7d"
	TopAssetsTimerange15Days TopAssetsTimerange = "15d"
	TopAssetsTimerange30Days TopAssetsTimerange = "30d"
)

// NewTopAssetsTimerange parses a string and returns a `TopAssetsTimerange`.
func NewTopAssetsTimerange(s string) (*TopAssetsTimerange, error) {

	if s == string(TopAssetsTimerange7Days) ||
		s == string(TopAssetsTimerange15Days) ||
		s == string(TopAssetsTimerange30Days) {

		tmp := TopAssetsTimerange(s)
		return &tmp, nil
	}

	return nil, fmt.Errorf("invalid timerange: %s", s)
}

type GlobalTransactionDoc struct {
	ID            string         `bson:"_id" json:"id"`
	OriginTx      *OriginTx      `bson:"originTx" json:"originTx"`
	DestinationTx *DestinationTx `bson:"destinationTx" json:"destinationTx"`
}

// OriginTx representa a origin transaction.
type OriginTx struct {
	ChainID   sdk.ChainID `bson:"chainId" json:"chainId"`
	TxHash    string      `bson:"nativeTxHash" json:"txHash"`
	Timestamp *time.Time  `bson:"timestamp" json:"timestamp"`
	Status    string      `bson:"status" json:"status"`
}

// DestinationTx representa a destination transaction.
type DestinationTx struct {
	ChainID     sdk.ChainID `bson:"chainId" json:"chainId"`
	Status      string      `bson:"status" json:"status"`
	Method      string      `bson:"method" json:"method"`
	TxHash      string      `bson:"txHash" json:"txHash"`
	From        string      `bson:"from" json:"from"`
	To          string      `bson:"to" json:"to"`
	BlockNumber string      `bson:"blockNumber" json:"blockNumber"`
	Timestamp   *time.Time  `bson:"timestamp" json:"timestamp"`
	UpdatedAt   *time.Time  `bson:"updatedAt" json:"updatedAt"`
}

// TransactionUpdate represents a transaction document.
type TransactionUpdate struct {
}

// GlobalTransactionQuery respresent a query for the globalTransactions mongodb document.
type GlobalTransactionQuery struct {
	pagination.Pagination
	id string
}

// Query create a new VaaQuery with default pagination vaues.
func Query() *GlobalTransactionQuery {
	p := pagination.Default()
	return &GlobalTransactionQuery{Pagination: *p}
}

// SetId set the chainId field of the VaaQuery struct.
func (q *GlobalTransactionQuery) SetId(id string) *GlobalTransactionQuery {
	q.id = id
	return q
}

type TransactionCountQuery struct {
	TimeSpan      string
	SampleRate    string
	CumulativeSum bool
}

type TransactionCountResult struct {
	Time  time.Time `json:"time" mapstructure:"_time"`
	Count uint64    `json:"count" mapstructure:"_value"`
}

type ChainActivityResult struct {
	ChainSourceID      string `mapstructure:"chain_source_id"`
	ChainDestinationID string `mapstructure:"chain_destination_id"`
	Volume             uint64 `mapstructure:"volume"`
}

type ChainActivityQuery struct {
	Start      *time.Time
	End        *time.Time
	AppIDs     []string
	IsNotional bool
}

func (q *ChainActivityQuery) HasAppIDS() bool {
	return len(q.AppIDs) > 0
}

func (q *ChainActivityQuery) GetAppIDs() []string {
	return q.AppIDs
}

func (q *ChainActivityQuery) GetStart() time.Time {
	if q.Start == nil {
		return time.UnixMilli(0)
	}
	return *q.Start
}

func (q *ChainActivityQuery) GetEnd() time.Time {
	if q.End == nil {
		return time.Now()
	}
	return *q.End
}
