// +build integration

package migrate

import (
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
)

const globalTestCollection = "test-global"

func init() {
	Register(func(db *mgo.Database) error {
		return db.C(globalTestCollection).Insert(bson.M{"a": "b"})
	}, func(db *mgo.Database) error {
		return db.C(globalTestCollection).Remove(bson.M{"a": "b"})
	})
}
