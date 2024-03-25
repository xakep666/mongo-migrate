//go:build integration

package migrate

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

const globalTestCollection = "test-global"

func init() {
	MustRegister(func(ctx context.Context, db *mongo.Database) error {
		_, err := db.Collection(globalTestCollection).InsertOne(ctx, bson.D{{"a", "b"}})
		if err != nil {
			return err
		}
		return nil
	}, func(ctx context.Context, db *mongo.Database) error {
		_, err := db.Collection(globalTestCollection).DeleteOne(ctx, bson.D{{"a", "b"}})
		if err != nil {
			return err
		}
		return nil
	})
}
