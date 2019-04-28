// +build integration

package migrate

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

const globalTestCollection = "test-global"

func init() {
	_ = Register(func(db *mongo.Database) error {
		_, err := db.Collection(globalTestCollection).InsertOne(context.Background(), bson.M{"a": "b"})
		return err
	}, func(db *mongo.Database) error {
		_, err := db.Collection(globalTestCollection).DeleteOne(context.Background(), bson.M{"a": "b"})
		return err
	})
}
