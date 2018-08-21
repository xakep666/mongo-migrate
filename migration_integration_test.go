// +build integration

package migrate

import (
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/globalsign/mgo"
)

func cleanup(db *mgo.Database) {
	collections, err := db.CollectionNames()
	if err != nil {
		panic(err)
	}
	for _, collection := range collections {
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
	m.Run()
}

func TestSetGetVersion(t *testing.T) {
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
}
