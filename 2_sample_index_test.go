//go:build integration

package migrate

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const globalTestIndexName = "test_idx_2"

func init() {
	MustRegister(func(ctx context.Context, db *mongo.Database) error {
		keys := bson.D{{"a", 1}}
		opt := options.Index().SetName(globalTestIndexName)
		model := mongo.IndexModel{Keys: keys, Options: opt}
		_, err := db.Collection(globalTestCollection).Indexes().CreateOne(ctx, model)
		if err != nil {
			return err
		}
		return nil
	}, func(ctx context.Context, db *mongo.Database) error {
		err := db.Collection(globalTestCollection).Indexes().DropOne(ctx, globalTestIndexName)
		if err != nil {
			return err
		}
		return nil
	})
}
