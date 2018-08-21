// +build integration

package migrate

import (
	"github.com/globalsign/mgo"
)

const globalTestIndexName = "test_idx_2"

func init() {
	Register(func(db *mgo.Database) error {
		return db.C(globalTestCollection).EnsureIndex(mgo.Index{Name: globalTestIndexName, Key: []string{"a"}})
	}, func(db *mgo.Database) error {
		return db.C(globalTestCollection).DropIndexName(globalTestIndexName)
	})
}
