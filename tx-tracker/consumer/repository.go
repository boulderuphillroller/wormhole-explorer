package consumer

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/wormhole-foundation/wormhole-explorer/api/handlers/vaa"
	"github.com/wormhole-foundation/wormhole-explorer/common/domain"
	"github.com/wormhole-foundation/wormhole-explorer/txtracker/chains"
	sdk "github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

// Repository exposes operations over the `globalTransactions` collection.
type Repository struct {
	logger             *zap.Logger
	globalTransactions *mongo.Collection
	vaas               *mongo.Collection
}

// New creates a new repository.
func NewRepository(logger *zap.Logger, db *mongo.Database) *Repository {

	r := Repository{
		logger:             logger,
		globalTransactions: db.Collection("globalTransactions"),
		vaas:               db.Collection("vaas"),
	}

	return &r
}

// UpsertDocumentParams is a struct that contains the parameters for the upsertDocument method.
type UpsertDocumentParams struct {
	VaaId     string
	ChainId   sdk.ChainID
	TxDetail  *chains.TxDetail
	TxStatus  domain.SourceTxStatus
	Timestamp *time.Time
}

func (r *Repository) UpsertDocument(ctx context.Context, params *UpsertDocumentParams) error {

	fields := bson.D{
		{Key: "status", Value: params.TxStatus},
	}

	if params.TxDetail != nil {
		fields = append(fields, primitive.E{Key: "nativeTxHash", Value: params.TxDetail.NativeTxHash})
		fields = append(fields, primitive.E{Key: "from", Value: params.TxDetail.From})
		if params.Timestamp != nil {
			fields = append(fields, primitive.E{Key: "timestamp", Value: params.Timestamp})
		}
		if params.TxDetail.Attribute != nil {
			fields = append(fields, primitive.E{Key: "attribute", Value: params.TxDetail.Attribute})
		}
	}

	update := bson.D{
		{
			Key: "$set",
			Value: bson.D{
				{
					Key:   "originTx",
					Value: fields,
				},
			},
		},
	}

	opts := options.Update().SetUpsert(true)

	_, err := r.globalTransactions.UpdateByID(ctx, params.VaaId, update, opts)
	if err != nil {
		return fmt.Errorf("failed to upsert source tx information: %w", err)
	}

	return nil
}

// AlreadyProcessed returns true if the given VAA ID has already been processed.
func (r *Repository) AlreadyProcessed(ctx context.Context, vaaId string) (bool, error) {

	result := r.
		globalTransactions.
		FindOne(ctx, bson.D{{"_id", vaaId}})

	var tx GlobalTransaction
	err := result.Decode(&tx)
	if err == mongo.ErrNoDocuments {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("failed to decode already processed VAA id: %w", err)
	} else {
		return true, nil
	}
}

// CountDocumentsByTimeRange returns the number of documents that match the given time range.
func (r *Repository) CountDocumentsByTimeRange(
	ctx context.Context,
	timeAfter time.Time,
	timeBefore time.Time,
) (uint64, error) {

	// Build the aggregation pipeline
	var pipeline mongo.Pipeline
	{
		// filter by time range
		pipeline = append(pipeline, bson.D{
			{"$match", bson.D{
				{"timestamp", bson.D{{"$gte", timeAfter}}},
			}},
		})
		pipeline = append(pipeline, bson.D{
			{"$match", bson.D{
				{"timestamp", bson.D{{"$lte", timeBefore}}},
			}},
		})

		// Count the number of results
		pipeline = append(pipeline, bson.D{
			{"$count", "numDocuments"},
		})
	}

	// Execute the aggregation pipeline
	cur, err := r.vaas.Aggregate(ctx, pipeline)
	if err != nil {
		r.logger.Error("failed execute aggregation pipeline", zap.Error(err))
		return 0, err
	}

	// Read results from cursor
	var results []struct {
		NumDocuments uint64 `bson:"numDocuments"`
	}
	err = cur.All(ctx, &results)
	if err != nil {
		r.logger.Error("failed to decode cursor", zap.Error(err))
		return 0, err
	}
	if len(results) == 0 {
		return 0, nil
	}
	if len(results) > 1 {
		r.logger.Error("too many results", zap.Int("numResults", len(results)))
		return 0, err
	}

	return results[0].NumDocuments, nil
}

// CountIncompleteDocuments returns the number of documents that have destTx data, but don't have sourceTx data.
func (r *Repository) CountIncompleteDocuments(ctx context.Context) (uint64, error) {

	// Build the aggregation pipeline
	var pipeline mongo.Pipeline
	{
		// Look up transactions that either:
		// 1. have not been processed
		// 2. have been processed, but encountered an internal error
		pipeline = append(pipeline, bson.D{
			{"$match", bson.D{
				{"$or", bson.A{
					bson.D{{"originTx", bson.D{{"$exists", false}}}},
					bson.D{{"originTx.status", bson.M{"$eq": domain.SourceTxStatusInternalError}}},
				}},
			}},
		})

		// Count the number of results
		pipeline = append(pipeline, bson.D{
			{"$count", "numDocuments"},
		})
	}

	// Execute the aggregation pipeline
	cur, err := r.globalTransactions.Aggregate(ctx, pipeline)
	if err != nil {
		r.logger.Error("failed execute aggregation pipeline", zap.Error(err))
		return 0, err
	}

	// Read results from cursor
	var results []struct {
		NumDocuments uint64 `bson:"numDocuments"`
	}
	err = cur.All(ctx, &results)
	if err != nil {
		r.logger.Error("failed to decode cursor", zap.Error(err))
		return 0, err
	}
	if len(results) == 0 {
		return 0, nil
	}
	if len(results) > 1 {
		r.logger.Error("too many results", zap.Int("numResults", len(results)))
		return 0, err
	}

	return results[0].NumDocuments, nil
}

type GlobalTransaction struct {
	Id   string       `bson:"_id"`
	Vaas []vaa.VaaDoc `bson:"vaas"`
}

// GetDocumentsByTimeRange iterates through documents within a specified time range.
func (r *Repository) GetDocumentsByTimeRange(
	ctx context.Context,
	lastId string,
	lastTimestamp *time.Time,
	limit uint,
	timeAfter time.Time,
	timeBefore time.Time,
) ([]GlobalTransaction, error) {

	// Build the aggregation pipeline
	var pipeline mongo.Pipeline
	{
		// Specify sorting criteria
		pipeline = append(pipeline, bson.D{
			{"$sort", bson.D{
				bson.E{"timestamp", -1},
				bson.E{"_id", 1},
			}},
		})

		// filter out already processed documents
		//
		// We use the timestap field as a pagination cursor
		if lastTimestamp != nil {
			pipeline = append(pipeline, bson.D{
				{"$match", bson.D{
					{"$or", bson.A{
						bson.D{{"timestamp", bson.M{"$lt": *lastTimestamp}}},
						bson.D{{"$and", bson.A{
							bson.D{{"timestamp", bson.M{"$eq": *lastTimestamp}}},
							bson.D{{"_id", bson.M{"$gt": lastId}}},
						}}},
					}},
				}},
			})
		}

		// filter by time range
		pipeline = append(pipeline, bson.D{
			{"$match", bson.D{
				{"timestamp", bson.D{{"$gte", timeAfter}}},
			}},
		})
		pipeline = append(pipeline, bson.D{
			{"$match", bson.D{
				{"timestamp", bson.D{{"$lte", timeBefore}}},
			}},
		})

		// Limit size of results
		pipeline = append(pipeline, bson.D{
			{"$limit", limit},
		})
	}

	// Execute the aggregation pipeline
	cur, err := r.vaas.Aggregate(ctx, pipeline)
	if err != nil {
		r.logger.Error("failed execute aggregation pipeline", zap.Error(err))
		return nil, errors.WithStack(err)
	}

	// Read results from cursor
	var documents []vaa.VaaDoc
	err = cur.All(ctx, &documents)
	if err != nil {
		r.logger.Error("failed to decode cursor", zap.Error(err))
		return nil, errors.WithStack(err)
	}

	// Build the result
	var globalTransactions []GlobalTransaction
	for i := range documents {
		globalTransaction := GlobalTransaction{
			Id:   documents[i].ID,
			Vaas: []vaa.VaaDoc{documents[i]},
		}
		globalTransactions = append(globalTransactions, globalTransaction)
	}

	return globalTransactions, nil
}

// GetIncompleteDocuments gets a batch of VAA IDs from the database.
func (r *Repository) GetIncompleteDocuments(
	ctx context.Context,
	lastId string,
	lastTimestamp *time.Time,
	limit uint,
) ([]GlobalTransaction, error) {

	// Build the aggregation pipeline
	var pipeline mongo.Pipeline
	{
		// Specify sorting criteria
		pipeline = append(pipeline, bson.D{
			{"$sort", bson.D{bson.E{"_id", 1}}},
		})

		// filter out already processed documents
		//
		// We use the _id field as a pagination cursor
		pipeline = append(pipeline, bson.D{
			{"$match", bson.D{{"_id", bson.M{"$gt": lastId}}}},
		})

		// Look up transactions that either:
		// 1. have not been processed
		// 2. have been processed, but encountered an internal error
		pipeline = append(pipeline, bson.D{
			{"$match", bson.D{
				{"$or", bson.A{
					bson.D{{"originTx", bson.D{{"$exists", false}}}},
					bson.D{{"originTx.status", bson.M{"$eq": domain.SourceTxStatusInternalError}}},
				}},
			}},
		})

		// Left join on the VAA collection
		pipeline = append(pipeline, bson.D{
			{"$lookup", bson.D{
				{"from", "vaas"},
				{"localField", "_id"},
				{"foreignField", "_id"},
				{"as", "vaas"},
			}},
		})

		// Limit size of results
		pipeline = append(pipeline, bson.D{
			{"$limit", limit},
		})
	}

	// Execute the aggregation pipeline
	cur, err := r.globalTransactions.Aggregate(ctx, pipeline)
	if err != nil {
		r.logger.Error("failed execute aggregation pipeline", zap.Error(err))
		return nil, errors.WithStack(err)
	}

	// Read results from cursor
	var documents []GlobalTransaction
	err = cur.All(ctx, &documents)
	if err != nil {
		r.logger.Error("failed to decode cursor", zap.Error(err))
		return nil, errors.WithStack(err)
	}

	return documents, nil
}
