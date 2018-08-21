// +build integration

package migrate

import (
	"errors"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
)

const testCollection = "test"

func cleanup(db *mgo.Database) {
	for _, collection := range []string{testCollection, defaultMigrationsCollection} {
		db.C(collection).DropAllIndexes()
		db.C(collection).DropCollection()
	}
}

var mongo *mgo.Database

func TestMain(m *testing.M) {
	addr, err := url.Parse(os.Getenv("MONGO_URL"))
	session, err := mgo.Dial(addr.String())
	if err != nil {
		panic(err)
	}
	defer session.Close()
	mongo = session.DB(strings.TrimLeft(addr.Path, "/"))
	defer cleanup(mongo)
	os.Exit(m.Run())
}

func TestSetGetVersion(t *testing.T) {
	defer cleanup(mongo)
	migrate := NewMigrate(mongo)
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
	defer cleanup(mongo)
	migrate := NewMigrate(mongo)
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
	defer cleanup(mongo)
	migrate := NewMigrate(mongo,
		Migration{Version: 1, Description: "hello", Up: func(db *mgo.Database) error {
			return db.C(testCollection).Insert(bson.M{"hello": "world"})
		}},
		Migration{Version: 2, Description: "world", Up: func(db *mgo.Database) error {
			return db.C(testCollection).EnsureIndex(mgo.Index{Name: "test_idx", Key: []string{"hello"}})
		}},
	)
	if err := migrate.Up(-1); err != nil {
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
	if err := mongo.C(testCollection).Find(bson.M{"hello": "world"}).One(&doc); err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	if doc["hello"].(string) != "world" {
		t.Errorf("Unexpected data")
		return
	}
	indexes, err := mongo.C(testCollection).Indexes()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	for _, index := range indexes {
		if index.Name == "test_idx" {
			return
		}
	}
	t.Errorf("Expected index not found")
}

func TestDownMigrations(t *testing.T) {
	defer cleanup(mongo)
	migrate := NewMigrate(mongo,
		Migration{Version: 1, Description: "hello", Up: func(db *mgo.Database) error {
			return db.C(testCollection).Insert(bson.M{"hello": "world"})
		}, Down: func(db *mgo.Database) error {
			return db.C(testCollection).Remove(bson.M{"hello": "world"})
		}},
		Migration{Version: 2, Description: "world", Up: func(db *mgo.Database) error {
			return db.C(testCollection).EnsureIndex(mgo.Index{Name: "test_idx", Key: []string{"hello"}})
		}, Down: func(db *mgo.Database) error {
			return db.C(testCollection).DropIndexName("test_idx")
		}},
	)
	if err := migrate.Up(-1); err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	if err := migrate.Down(-1); err != nil {
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
	err = mongo.C(testCollection).Find(bson.M{"hello": "world"}).One(&bson.M{})
	if err != mgo.ErrNotFound {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	indexes, err := mongo.C(testCollection).Indexes()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	for _, index := range indexes {
		if index.Name == "test_idx" {
			t.Errorf("Index unexpectedly found")
			return
		}
	}
}

func TestPartialUpMigrations(t *testing.T) {
	defer cleanup(mongo)
	migrate := NewMigrate(mongo,
		Migration{Version: 1, Description: "hello", Up: func(db *mgo.Database) error {
			return db.C(testCollection).Insert(bson.M{"hello": "world"})
		}},
		Migration{Version: 2, Description: "world", Up: func(db *mgo.Database) error {
			return db.C(testCollection).EnsureIndex(mgo.Index{Name: "test_idx", Key: []string{"hello"}})
		}},
		Migration{Version: 3, Description: "shouldn`t be applied", Up: func(db *mgo.Database) error {
			return db.C(testCollection).Insert(bson.M{"a": "b"})
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
	if err := mongo.C(testCollection).Find(bson.M{"hello": "world"}).One(&doc); err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	if doc["hello"].(string) != "world" {
		t.Errorf("Unexpected data")
		return
	}
	indexes, err := mongo.C(testCollection).Indexes()
	if err != nil {
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
	err = mongo.C(testCollection).Find(bson.M{"a": "b"}).One(&bson.M{})
	if err != mgo.ErrNotFound {
		t.Errorf("Unexpectedly found data from non-applied migration")
		return
	}
}

func TestPartialDownMigrations(t *testing.T) {
	defer cleanup(mongo)
	migrate := NewMigrate(mongo,
		Migration{Version: 1, Description: "hello", Up: func(db *mgo.Database) error {
			return db.C(testCollection).Insert(bson.M{"hello": "world"})
		}, Down: func(db *mgo.Database) error {
			return db.C(testCollection).Remove(bson.M{"hello": "world"})
		}},
		Migration{Version: 2, Description: "world", Up: func(db *mgo.Database) error {
			return db.C(testCollection).EnsureIndex(mgo.Index{Name: "test_idx", Key: []string{"hello"}})
		}, Down: func(db *mgo.Database) error {
			return db.C(testCollection).DropIndexName("test_idx")
		}},
		Migration{Version: 3, Description: "next", Up: func(db *mgo.Database) error {
			return db.C(testCollection).Insert(bson.M{"a": "b"})
		}, Down: func(db *mgo.Database) error {
			return db.C(testCollection).Remove(bson.M{"a": "b"})
		}},
	)
	if err := migrate.Up(-1); err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	err := mongo.C(testCollection).Find(bson.M{"a": "b"}).One(&bson.M{})
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
	err = mongo.C(testCollection).Find(bson.M{"a": "b"}).One(&bson.M{})
	if err != mgo.ErrNotFound {
		t.Errorf("Unexpected error: %v", err)
		return
	}
}

func TestUpMigrationWithErrors(t *testing.T) {
	defer cleanup(mongo)
	expectedErr := errors.New("normal error")
	migrate := NewMigrate(mongo,
		Migration{Version: 1, Description: "hello", Up: func(db *mgo.Database) error {
			return expectedErr
		}},
	)
	if err := migrate.Up(-1); err != expectedErr {
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
	defer cleanup(mongo)
	expectedErr := errors.New("normal error")
	migrate := NewMigrate(mongo,
		Migration{Version: 1, Description: "hello", Up: func(db *mgo.Database) error {
			return db.C(testCollection).Insert(bson.M{"hello": "world"})
		}, Down: func(db *mgo.Database) error {
			return expectedErr
		}},
	)
	if err := migrate.Up(-1); err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	if err := migrate.Down(-1); err != expectedErr {
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
	defer cleanup(mongo)
	var cnt int
	migrate := NewMigrate(mongo,
		Migration{Version: 1, Description: "hello", Up: func(db *mgo.Database) error {
			cnt++
			return nil
		}},
		Migration{Version: 2, Description: "world", Up: func(db *mgo.Database) error {
			cnt++
			return nil
		}},
	)
	if err := migrate.Up(-1); err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	if err := migrate.Up(-1); err != nil {
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
	defer cleanup(mongo)
	var cnt int
	migrate := NewMigrate(mongo,
		Migration{Version: 1, Description: "hello", Up: func(db *mgo.Database) error {
			return nil
		}, Down: func(db *mgo.Database) error {
			cnt++
			return nil
		}},
		Migration{Version: 2, Description: "world", Up: func(db *mgo.Database) error {
			return nil
		}, Down: func(db *mgo.Database) error {
			cnt++
			return nil
		}},
	)
	if err := migrate.Up(-1); err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	if err := migrate.Down(-1); err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	if err := migrate.Down(-1); err != nil {
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
