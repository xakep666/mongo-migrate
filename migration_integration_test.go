// +build integration

package migrate

import (
	"context"
	"errors"
	"fmt"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/bsonx"
	"os"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

const testCollection = "test"

func cleanup(db *mongo.Database) {
	fmt.Println(db)
	fmt.Println(mongoClient)
	fmt.Println(mongoDB)
	if db == nil {
		db = mongoClient.Database(os.Getenv("MONGO_DB"))
	}
	fmt.Println(db)
	cursor, err := db.ListCollections(context.Background(),nil, nil)
	if err != nil {
		panic(err)
	}
	for cursor.Next(context.Background()) {
		next := &bsonx.Doc{}
		err = cursor.Decode(next)
		if err != nil {
			panic(err)
		}
		elem, err := next.LookupErr("name")
		if err != nil {
			panic(err)
		}
		elemName := elem.StringValue()
		_, err = db.Collection(elemName).Indexes().DropAll(context.Background())
		if err != nil {
			panic(err)
		}
		err = db.Collection(elemName).Drop(context.Background())
		if err != nil {
			panic(err)
		}
	}
}

var mongoDB *mongo.Database
var mongoClient *mongo.Client

func TestMain(m *testing.M) {
	clientOptions := options.Client().ApplyURI(os.Getenv("MONGO_URL")).SetMaxPoolSize(10)
	// Connect to MongoDB
	ctx, _ := context.WithTimeout(context.Background(), 20*time.Second)
	//defer cancel()
	mongoClient, err := mongo.Connect(ctx, clientOptions)

	if err != nil {
		panic(err)
	}
	// Check the connection
	err = mongoClient.Ping(ctx, nil)

	if err != nil {
		panic(err)
	}
	mongoDB := mongoClient.Database(os.Getenv("MONGO_DB"))
	fmt.Println(mongoDB)
	fmt.Println(mongoClient)
	defer cleanup(mongoDB)
	//defer client.Disconnect(ctx)
	os.Exit(m.Run())
}

func TestSetGetVersion(t *testing.T) {
	defer cleanup(mongoDB)
	migrate := NewMigrate(mongoDB)
	if err := migrate.SetVersion(1, "hello"); err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	version, description, err := migrate.Version()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	if version != 1 || description != "hello" {
		t.Errorf("Unexpected version/description %v %v", version, description)
		return
	}

	if err := migrate.SetVersion(2, "world"); err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	version, description, err = migrate.Version()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	if version != 2 || description != "world" {
		t.Errorf("Unexpected version/description %v %v", version, description)
		return
	}

	if err := migrate.SetVersion(1, "hello"); err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	version, description, err = migrate.Version()
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
	defer cleanup(mongoDB)
	migrate := NewMigrate(mongoDB)
	version, _, err := migrate.Version()
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
	defer cleanup(mongoDB)
	migrate := NewMigrate(mongoDB,
		Migration{Version: 1, Description: "hello", Up: func(db *mongo.Database) error {
			_, err := db.Collection(testCollection).InsertOne(context.Background(),bson.M{"hello": "world"})
			return err
		}},
		Migration{Version: 2, Description: "world", Up: func(db *mongo.Database) error {
			coll := db.Collection(testCollection)
			indexView := coll.Indexes()
			_, err := indexView.CreateOne(context.Background(), mongo.IndexModel{
				Keys: bsonx.Doc{{"hello", bsonx.String("test_idx")}},
			})
			return err
		}},
	)
	if err := migrate.Up(AllAvailable); err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	version, description, err := migrate.Version()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	if version != 2 || description != "world" {
		t.Errorf("Unexpected version/description %v %v", version, description)
		return
	}
	doc := bson.M{}
	if err := mongoDB.Collection(testCollection).FindOne(context.Background(),bson.M{"hello": "world"}).Decode(&doc); err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	if doc["hello"].(string) != "world" {
		t.Errorf("Unexpected data")
		return
	}
	indexes, err := mongoDB.Collection(testCollection).Indexes().List(context.Background())
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	for indexes.Next(context.Background()) {
		next := &bsonx.Doc{}
		err = indexes.Decode(next)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
			return
		}
		elem, err := next.LookupErr("name")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
			return
		}
		elemName := elem.StringValue()
		if elemName == "test_idx" {
			return
		}
	}
	t.Errorf("Expected index not found")
}

func TestDownMigrations(t *testing.T) {
	defer cleanup(mongoDB)
	migrate := NewMigrate(mongoDB,
		Migration{Version: 1, Description: "hello", Up: func(db *mongo.Database) error {
			_, err :=  db.Collection(testCollection).InsertOne(context.Background(), bson.M{"hello": "world"})
			return err
		}, Down: func(db *mongo.Database) error {
			_, err := db.Collection(testCollection).DeleteOne(context.Background(),bson.M{"hello": "world"})
			return err
		}},
		Migration{Version: 2, Description: "world", Up: func(db *mongo.Database) error {
			coll := db.Collection(testCollection)
			indexView := coll.Indexes()
			_, err := indexView.CreateOne(context.Background(), mongo.IndexModel{
				Keys: bsonx.Doc{{"hello", bsonx.String("test_idx")}},
			})
			return err
		}, Down: func(db *mongo.Database) error {
			coll := db.Collection(testCollection)
			indexView := coll.Indexes()
			_, err :=indexView.DropOne(context.Background(),"test_idx")
			return err
		}},
	)
	if err := migrate.Up(AllAvailable); err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	if err := migrate.Down(AllAvailable); err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	version, _, err := migrate.Version()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	if version != 0 {
		t.Errorf("Unexpected version: %v", version)
		return
	}
	err = mongoDB.Collection(testCollection).FindOne(context.Background(),bson.M{"hello": "world"}).Decode(&bson.M{})
	if err != mongo.ErrNoDocuments {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	indexes, err := mongoDB.Collection(testCollection).Indexes().List(context.Background())
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	for indexes.Next(context.Background()) {
		next := &bsonx.Doc{}
		err = indexes.Decode(next)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
			return
		}
		elem, err := next.LookupErr("name")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
			return
		}
		elemName := elem.StringValue()
		if elemName == "test_idx" {
			t.Errorf("Index unexpectedly found")
			return
		}
	}
}

func TestPartialUpMigrations(t *testing.T) {
	defer cleanup(mongoDB)
	migrate := NewMigrate(mongoDB,
		Migration{Version: 1, Description: "hello", Up: func(db *mongo.Database) error {
			_, err := db.Collection(testCollection).InsertOne(context.Background(),bson.M{"hello": "world"})
			return err
		}},
		Migration{Version: 2, Description: "world", Up: func(db *mongo.Database) error {
			coll := db.Collection(testCollection)
			indexView := coll.Indexes()
			_, err := indexView.CreateOne(context.Background(), mongo.IndexModel{
				Keys: bsonx.Doc{{"hello", bsonx.String("test_idx")}},
			})
			return err
		}},
		Migration{Version: 3, Description: "shouldn`t be applied", Up: func(db *mongo.Database) error {
			_, err := db.Collection(testCollection).InsertOne(context.Background(),bson.M{"a": "b"})
			return err
		}},
	)
	if err := migrate.Up(2); err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	version, description, err := migrate.Version()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	if version != 2 || description != "world" {
		t.Errorf("Unexpected version/description %v %v", version, description)
		return
	}
	doc := bson.M{}
	if err := mongoDB.Collection(testCollection).FindOne(context.Background(),bson.M{"hello": "world"}).Decode(&doc); err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	if doc["hello"].(string) != "world" {
		t.Errorf("Unexpected data")
		return
	}
	indexes, err := mongoDB.Collection(testCollection).Indexes().List(context.Background())
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	for indexes.Next(context.Background()) {
		next := &bsonx.Doc{}
		err = indexes.Decode(next)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
			return
		}
		elem, err := next.LookupErr("name")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
			return
		}
		elemName := elem.StringValue()
		if elemName == "test_idx" {
			goto okIndex
		}
	}
	t.Errorf("Expected index not found")
okIndex:
	err = mongoDB.Collection(testCollection).FindOne(context.Background(),bson.M{"a": "b"}).Decode(&bson.M{})
	if err != mongo.ErrNoDocuments {
		t.Errorf("Unexpectedly found data from non-applied migration")
		return
	}
}

func TestPartialDownMigrations(t *testing.T) {
	defer cleanup(mongoDB)
	migrate := NewMigrate(mongoDB,
		Migration{Version: 1, Description: "hello", Up: func(db *mongo.Database) error {
			_, err := db.Collection(testCollection).InsertOne(context.Background(),bson.M{"hello": "world"})
			return err
		}, Down: func(db *mongo.Database) error {
			_, err := db.Collection(testCollection).DeleteOne(context.Background(),  bson.M{"hello": "world"})
			return err
		}},
		Migration{Version: 2, Description: "world", Up: func(db *mongo.Database) error {
			coll := db.Collection(testCollection)
			indexView := coll.Indexes()
			_, err := indexView.CreateOne(context.Background(), mongo.IndexModel{
				Keys: bsonx.Doc{{"hello", bsonx.String("test_idx")}},
			})
			return err
		}, Down: func(db *mongo.Database) error {
			coll := db.Collection(testCollection)
			indexView := coll.Indexes()
			_, err :=indexView.DropOne(context.Background(),"test_idx")
			return err

		}},
		Migration{Version: 3, Description: "next", Up: func(db *mongo.Database) error {
			_, err :=  db.Collection(testCollection).InsertOne(context.Background(), bson.M{"a": "b"})
			return err
		}, Down: func(db *mongo.Database) error {
			_, err := db.Collection(testCollection).DeleteOne(context.Background(), bson.M{"a": "b"})
			return err
		}},
	)
	if err := migrate.Up(AllAvailable); err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	err := mongoDB.Collection(testCollection).FindOne(context.Background(), bson.M{"a": "b"}).Decode(&bson.M{})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	if err := migrate.Down(1); err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	version, description, err := migrate.Version()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	if version != 2 || description != "world" {
		t.Errorf("Unexpected version/description: %v %v", version, description)
		return
	}
	err = mongoDB.Collection(testCollection).FindOne(context.Background(), bson.M{"a": "b"}).Decode(&bson.M{})
	if err != mongo.ErrNoDocuments {
		t.Errorf("Unexpected error: %v", err)
		return
	}
}

func TestUpMigrationWithErrors(t *testing.T) {
	defer cleanup(mongoDB)
	expectedErr := errors.New("normal error")
	migrate := NewMigrate(mongoDB,
		Migration{Version: 1, Description: "hello", Up: func(db *mongo.Database) error {
			return expectedErr
		}},
	)
	if err := migrate.Up(AllAvailable); err != expectedErr {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	version, _, err := migrate.Version()
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
	defer cleanup(mongoDB)
	expectedErr := errors.New("normal error")
	migrate := NewMigrate(mongoDB,
		Migration{Version: 1, Description: "hello", Up: func(db *mongo.Database) error {
			_, err := db.Collection(testCollection).InsertOne(context.Background(), bson.M{"hello": "world"})
			return err
		}, Down: func(db *mongo.Database) error {
			return expectedErr
		}},
	)
	if err := migrate.Up(AllAvailable); err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	if err := migrate.Down(AllAvailable); err != expectedErr {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	version, _, err := migrate.Version()
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
	defer cleanup(mongoDB)
	var cnt int
	migrate := NewMigrate(mongoDB,
		Migration{Version: 1, Description: "hello", Up: func(db *mongo.Database) error {
			cnt++
			return nil
		}},
		Migration{Version: 2, Description: "world", Up: func(db *mongo.Database) error {
			cnt++
			return nil
		}},
	)
	if err := migrate.Up(AllAvailable); err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	if err := migrate.Up(AllAvailable); err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	version, description, err := migrate.Version()
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
	defer cleanup(mongoDB)
	var cnt int
	migrate := NewMigrate(mongoDB,
		Migration{Version: 1, Description: "hello", Up: func(db *mongo.Database) error {
			return nil
		}, Down: func(db *mongo.Database) error {
			cnt++
			return nil
		}},
		Migration{Version: 2, Description: "world", Up: func(db *mongo.Database) error {
			return nil
		}, Down: func(db *mongo.Database) error {
			cnt++
			return nil
		}},
	)
	if err := migrate.Up(AllAvailable); err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	if err := migrate.Down(AllAvailable); err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	if err := migrate.Down(AllAvailable); err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	version, _, err := migrate.Version()
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
