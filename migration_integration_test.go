//go:build integration

package migrate

import (
	"context"
	"errors"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const testCollection = "test"

type index struct {
	Key  map[string]int
	NS   string
	Name string
}

func cleanup(db *mongo.Database) {
	ctx := context.Background()
	opts := options.ListCollections().SetNameOnly(true)

	cursor, err := db.ListCollections(ctx, bson.D{}, opts)
	if err != nil {
		panic(err)
	}

	defer cursor.Close(ctx)

	var collections []collectionSpecification

	for cursor.Next(ctx) {
		var collection collectionSpecification

		err := cursor.Decode(&collection)
		if err != nil {
			panic(err)
		}

		collections = append(collections, collection)
	}

	if err := cursor.Err(); err != nil {
		panic(err)
	}

	for _, collection := range collections {
		_, err := db.Collection(collection.Name).Indexes().DropAll(ctx)
		if err != nil {
			panic(err)
		}
		err = db.Collection(collection.Name).Drop(ctx)
		if err != nil {
			panic(err)
		}
	}
}

var db *mongo.Database

func TestMain(m *testing.M) {
	addr, err := url.Parse(os.Getenv("MONGO_URL"))
	opt := options.Client().ApplyURI(addr.String())
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, opt)
	if err != nil {
		panic(err)
	}
	db = client.Database(strings.TrimLeft(addr.Path, "/"))
	defer cleanup(db)
	os.Exit(m.Run())
}

func TestSetGetVersion(t *testing.T) {
	defer cleanup(db)
	migrate := NewMigrate(db)
	ctx := context.Background()
	if err := migrate.SetVersion(ctx, 1, "hello"); err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	version, description, err := migrate.Version(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	if version != 1 || description != "hello" {
		t.Errorf("Unexpected version/description %v %v", version, description)
		return
	}

	if err := migrate.SetVersion(ctx, 2, "world"); err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	version, description, err = migrate.Version(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	if version != 2 || description != "world" {
		t.Errorf("Unexpected version/description %v %v", version, description)
		return
	}

	if err := migrate.SetVersion(ctx, 1, "hello"); err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	version, description, err = migrate.Version(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	if version != 1 || description != "hello" {
		t.Errorf("Unexpected version/description %v %v", version, description)
		return
	}
}

func TestVersionBeforeSet(t *testing.T) {
	defer cleanup(db)
	migrate := NewMigrate(db)
	ctx := context.Background()
	version, _, err := migrate.Version(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	if version != 0 {
		t.Errorf("Unexpected version: %v", err)
		return
	}
}

func TestUpMigrations(t *testing.T) {
	defer cleanup(db)
	ctx := context.Background()
	migrate := NewMigrate(db,
		Migration{Version: 1, Description: "hello", Up: func(ctx context.Context, db *mongo.Database) error {
			_, err := db.Collection(testCollection).InsertOne(ctx, bson.D{{"hello", "world"}})
			if err != nil {
				return err
			}

			return nil
		}},
		Migration{Version: 2, Description: "world", Up: func(ctx context.Context, db *mongo.Database) error {
			opt := options.Index().SetName("test_idx")
			keys := bson.D{{"hello", 1}}
			model := mongo.IndexModel{Keys: keys, Options: opt}
			_, err := db.Collection(testCollection).Indexes().CreateOne(ctx, model)
			if err != nil {
				return err
			}

			return nil
		}},
	)
	if err := migrate.Up(ctx, AllAvailable); err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	version, description, err := migrate.Version(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	if version != 2 || description != "world" {
		t.Errorf("Unexpected version/description %v %v", version, description)
		return
	}
	result := db.Collection(testCollection).FindOne(ctx, bson.D{{"hello", "world"}})
	if result.Err() != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	doc := bson.M{}
	if err := result.Decode(&doc); err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	if doc["hello"].(string) != "world" {
		t.Errorf("Unexpected data")
		return
	}
	cursor, err := db.Collection(testCollection).Indexes().List(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	defer cursor.Close(ctx)

	var indexes []index
	for cursor.Next(ctx) {
		var index index

		err := cursor.Decode(&index)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
			return
		}

		indexes = append(indexes, index)
	}

	if err := cursor.Err(); err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}

	for _, v := range indexes {
		if v.Name == "test_idx" {
			return
		}
	}

	t.Errorf("Expected index not found")
}

func TestDownMigrations(t *testing.T) {
	defer cleanup(db)
	ctx := context.Background()
	migrate := NewMigrate(db,
		Migration{Version: 1, Description: "hello", Up: func(ctx context.Context, db *mongo.Database) error {
			_, err := db.Collection(testCollection).InsertOne(ctx, bson.D{{"hello", "world"}})
			if err != nil {
				return err
			}
			return nil
		}, Down: func(ctx context.Context, db *mongo.Database) error {
			_, err := db.Collection(testCollection).DeleteOne(ctx, bson.D{{"hello", "world"}})
			if err != nil {
				return err
			}
			return nil
		}},
		Migration{Version: 2, Description: "world", Up: func(ctx context.Context, db *mongo.Database) error {
			opt := options.Index().SetName("test_idx")
			keys := bson.D{{"hello", 1}}
			model := mongo.IndexModel{Keys: keys, Options: opt}
			_, err := db.Collection(testCollection).Indexes().CreateOne(ctx, model)
			if err != nil {
				return err
			}

			return nil
		}, Down: func(ctx context.Context, db *mongo.Database) error {
			_, err := db.Collection(testCollection).Indexes().DropOne(ctx, "test_idx")
			if err != nil {
				return err
			}
			return nil
		}},
	)
	if err := migrate.Up(ctx, AllAvailable); err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	if err := migrate.Down(ctx, AllAvailable); err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	version, _, err := migrate.Version(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	if version != 0 {
		t.Errorf("Unexpected version: %v", version)
		return
	}
	result := db.Collection(testCollection).FindOne(ctx, bson.D{{"hello", "world"}})
	if err := result.Decode(&bson.D{}); !errors.Is(err, mongo.ErrNoDocuments) {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	cursor, err := db.Collection(testCollection).Indexes().List(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	defer cursor.Close(ctx)

	var indexes []index
	for cursor.Next(ctx) {
		var index index

		err := cursor.Decode(&index)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
			return
		}

		indexes = append(indexes, index)
	}

	if err := cursor.Err(); err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}

	for _, v := range indexes {
		if v.Name == "test_idx" {
			t.Errorf("Index unexpectedly found")
			return
		}
	}
}

func TestPartialUpMigrations(t *testing.T) {
	defer cleanup(db)
	ctx := context.Background()
	migrate := NewMigrate(db,
		Migration{Version: 1, Description: "hello", Up: func(ctx context.Context, db *mongo.Database) error {
			_, err := db.Collection(testCollection).InsertOne(ctx, bson.D{{"hello", "world"}})
			if err != nil {
				return err
			}
			return nil
		}},
		Migration{Version: 2, Description: "world", Up: func(ctx context.Context, db *mongo.Database) error {
			opt := options.Index().SetName("test_idx")
			keys := bson.D{{"hello", 1}}
			model := mongo.IndexModel{Keys: keys, Options: opt}
			_, err := db.Collection(testCollection).Indexes().CreateOne(ctx, model)
			if err != nil {
				return err
			}
			return nil
		}},
		Migration{Version: 3, Description: "shouldn`t be applied", Up: func(ctx context.Context, db *mongo.Database) error {
			_, err := db.Collection(testCollection).InsertOne(ctx, bson.D{{"a", "b"}})
			if err != nil {
				return err
			}
			return nil
		}},
	)
	if err := migrate.Up(ctx, 2); err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	version, description, err := migrate.Version(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	if version != 2 || description != "world" {
		t.Errorf("Unexpected version/description %v %v", version, description)
		return
	}
	result := db.Collection(testCollection).FindOne(ctx, bson.D{{"hello", "world"}})
	if err := result.Err(); err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	var doc bson.M
	err = result.Decode(&doc)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	if doc["hello"].(string) != "world" {
		t.Errorf("Unexpected data")
		return
	}
	cursor, err := db.Collection(testCollection).Indexes().List(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	defer cursor.Close(ctx)

	var indexes []index
	for cursor.Next(ctx) {
		var index index

		err := cursor.Decode(&index)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
			return
		}

		indexes = append(indexes, index)
	}

	if err := cursor.Err(); err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}

	for _, index := range indexes {
		if index.Name == "test_idx" {
			goto okIndex
		}
	}
	t.Errorf("Expected index not found")
okIndex:
	res := db.Collection(testCollection).FindOne(ctx, bson.D{{"a", "b"}})
	if err := res.Decode(&bson.D{}); !errors.Is(err, mongo.ErrNoDocuments) {
		t.Errorf("Unexpectedly found data from non-applied migration")
		return
	}
}

func TestPartialDownMigrations(t *testing.T) {
	defer cleanup(db)
	ctx := context.Background()
	migrate := NewMigrate(db,
		Migration{Version: 1, Description: "hello", Up: func(ctx context.Context, db *mongo.Database) error {
			_, err := db.Collection(testCollection).InsertOne(ctx, bson.D{{"hello", "world"}})
			if err != nil {
				return err
			}
			return nil
		}, Down: func(ctx context.Context, db *mongo.Database) error {
			_, err := db.Collection(testCollection).DeleteOne(ctx, bson.D{{"hello", "world"}})
			if err != nil {
				return err
			}
			return nil
		}},
		Migration{Version: 2, Description: "world", Up: func(ctx context.Context, db *mongo.Database) error {
			keys := bson.D{{"hello", 1}}
			opt := options.Index().SetName("test_idx")
			model := mongo.IndexModel{Keys: keys, Options: opt}
			_, err := db.Collection(testCollection).Indexes().CreateOne(ctx, model)
			if err != nil {
				return err
			}
			return err
		}, Down: func(ctx context.Context, db *mongo.Database) error {
			_, err := db.Collection(testCollection).Indexes().DropOne(ctx, "test_idx")
			if err != nil {
				return err
			}
			return nil
		}},
		Migration{Version: 3, Description: "next", Up: func(ctx context.Context, db *mongo.Database) error {
			_, err := db.Collection(testCollection).InsertOne(ctx, bson.D{{"a", "b"}})
			if err != nil {
				return err
			}
			return nil
		}, Down: func(ctx context.Context, db *mongo.Database) error {
			_, err := db.Collection(testCollection).DeleteOne(ctx, bson.D{{"a", "b"}})
			if err != nil {
				return err
			}
			return nil
		}},
	)
	if err := migrate.Up(ctx, AllAvailable); err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	result := db.Collection(testCollection).FindOne(ctx, bson.D{{"a", "b"}})
	if err := result.Err(); err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	if err := migrate.Down(ctx, 1); err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	version, description, err := migrate.Version(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	if version != 2 || description != "world" {
		t.Errorf("Unexpected version/description: %v %v", version, description)
		return
	}
	res := db.Collection(testCollection).FindOne(ctx, bson.D{{"a", "b"}})
	if err := res.Decode(&bson.D{}); !errors.Is(err, mongo.ErrNoDocuments) {
		t.Errorf("Unexpected error: %v", err)
		return
	}
}

func TestUpMigrationWithErrors(t *testing.T) {
	defer cleanup(db)
	expectedErr := errors.New("normal error")
	ctx := context.Background()
	migrate := NewMigrate(db,
		Migration{Version: 1, Description: "hello", Up: func(ctx context.Context, db *mongo.Database) error {
			return expectedErr
		}},
	)
	if err := migrate.Up(ctx, AllAvailable); !errors.Is(err, expectedErr) {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	version, _, err := migrate.Version(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	if version != 0 {
		t.Errorf("Unexpected version: %v", version)
		return
	}
}

func TestDownMigrationWithErrors(t *testing.T) {
	defer cleanup(db)
	expectedErr := errors.New("normal error")
	ctx := context.Background()
	migrate := NewMigrate(db,
		Migration{Version: 1, Description: "hello", Up: func(ctx context.Context, db *mongo.Database) error {
			_, err := db.Collection(testCollection).InsertOne(ctx, bson.D{{"hello", "world"}})
			if err != nil {
				return err
			}
			return nil
		}, Down: func(ctx context.Context, db *mongo.Database) error {
			return expectedErr
		}},
	)
	if err := migrate.Up(ctx, AllAvailable); err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	if err := migrate.Down(ctx, AllAvailable); !errors.Is(err, expectedErr) {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	version, _, err := migrate.Version(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	if version != 1 {
		t.Errorf("Unexpected version: %v", version)
		return
	}
}

func TestMultipleUpMigration(t *testing.T) {
	defer cleanup(db)
	var cnt int
	ctx := context.Background()
	migrate := NewMigrate(db,
		Migration{Version: 1, Description: "hello", Up: func(ctx context.Context, db *mongo.Database) error {
			cnt++
			return nil
		}},
		Migration{Version: 2, Description: "world", Up: func(ctx context.Context, db *mongo.Database) error {
			cnt++
			return nil
		}},
	)
	if err := migrate.Up(ctx, AllAvailable); err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	if err := migrate.Up(ctx, AllAvailable); err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	version, description, err := migrate.Version(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	if version != 2 || description != "world" {
		t.Errorf("Unexpected version/description %v %v", version, description)
		return
	}
	if cnt != 2 {
		t.Errorf("Unexpected apply call count: %v", cnt)
		return
	}
}

func TestMultipleDownMigration(t *testing.T) {
	defer cleanup(db)
	var cnt int
	ctx := context.Background()
	migrate := NewMigrate(db,
		Migration{Version: 1, Description: "hello", Up: func(ctx context.Context, db *mongo.Database) error {
			return nil
		}, Down: func(ctx context.Context, db *mongo.Database) error {
			cnt++
			return nil
		}},
		Migration{Version: 2, Description: "world", Up: func(ctx context.Context, db *mongo.Database) error {
			return nil
		}, Down: func(ctx context.Context, db *mongo.Database) error {
			cnt++
			return nil
		}},
	)
	if err := migrate.Up(ctx, AllAvailable); err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	if err := migrate.Down(ctx, AllAvailable); err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	if err := migrate.Down(ctx, AllAvailable); err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	version, _, err := migrate.Version(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	if version != 0 {
		t.Errorf("Unexpected version: %v", version)
		return
	}
	if cnt != 2 {
		t.Errorf("Unexpected apply call count: %v", cnt)
		return
	}
}
