package repository

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

// VaaRepository is a repository for VAA.
type VaaRepository struct {
	db     *mongo.Database
	logger *zap.Logger
	vaas   *mongo.Collection
}

// VaaDoc is a document for VAA.
type VaaDoc struct {
	ID  string `bson:"_id" json:"id"`
	Vaa []byte `bson:"vaas" json:"vaa"`
}

// NewVaaRepository create a new Vaa repository.
func NewVaaRepository(db *mongo.Database, logger *zap.Logger) *VaaRepository {
	return &VaaRepository{db: db,
		logger: logger.With(zap.String("module", "VaaRepository")),
		vaas:   db.Collection("vaas"),
	}
}

// FindById finds VAA by id.
func (r *VaaRepository) FindById(ctx context.Context, id string) (*VaaDoc, error) {
	var vaaDoc VaaDoc
	err := r.vaas.FindOne(ctx, bson.M{"_id": id}).Decode(&vaaDoc)
	return &vaaDoc, err
}

// FindPageByTimeRange finds VAA by time range.
func (r *VaaRepository) FindPageByTimeRange(ctx context.Context, startTime time.Time, endTime time.Time, page, pageSize int64, sortAsc bool) ([]*VaaDoc, error) {
	filter := bson.M{
		"timestamp": bson.M{
			"$gte": startTime,
			"$lt":  endTime,
		},
	}
	sort := -1
	if sortAsc {
		sort = 1
	}

	skip := page * pageSize
	opts := &options.FindOptions{Skip: &skip, Limit: &pageSize, Sort: bson.M{"timestamp": sort}}
	cur, err := r.vaas.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	var vaas []*VaaDoc
	err = cur.All(ctx, &vaas)
	return vaas, err
}
