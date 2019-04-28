// +build integration

package migrate

import (
	"context"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/x/bsonx"
)

const globalTestIndexName = "test_idx_2"

func init() {
	_ = Register(func(db *mongo.Database) error {
		coll := db.Collection(globalTestCollection)
		indexView := coll.Indexes()
		_, err := indexView.CreateOne(context.Background(), mongo.IndexModel{
			Keys: bsonx.Doc{{"a", bsonx.String(globalTestIndexName)}},
		})

		return err

	}, func(db *mongo.Database) error {
		coll := db.Collection(globalTestCollection)
		indexView := coll.Indexes()
		_, err := indexView.DropOne(context.Background(), globalTestIndexName)
		return err
	})
}
