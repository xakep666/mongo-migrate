// +build integration

package migrate

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

const globalTestCollection = "test-global"

func init() {
	Register(func(db *mongo.Database) error {
		_, err := db.Collection(globalTestCollection).InsertOne(context.TODO(), bson.D{{"a", "b"}})
		if err != nil {
			return err
		}
		return nil
	}, func(db *mongo.Database) error {
		_, err := db.Collection(globalTestCollection).DeleteOne(context.TODO(), bson.D{{"a", "b"}})
		if err != nil {
			return err
		}
		return nil
	})
}
