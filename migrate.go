package mongo_migrate

import (
	"time"

	"github.com/globalsign/mgo"
)

type versionRecord struct {
	Version     uint64
	Description string
	Timestamp   time.Time
}

type Migrate struct {
	db                   *mgo.Database
	migrations           []Migration
	migrationsCollection string
}

func NewMigrate(db *mgo.Database, migrations ...Migration) *Migrate {
	return &Migrate{
		db:                   db,
		migrations:           migrations,
		migrationsCollection: "migrations",
	}
}

func (m *Migrate) SetMigrationsCollection(name string) {
	m.migrationsCollection = name
}

func (m *Migrate) Version() (uint64, string, error) {
	var rec versionRecord
	// find record with greatest id (assuming it`s latest also)
	err := m.db.C(m.migrationsCollection).Find(nil).Sort("-_id").One(&rec)
	if err != nil {
		return 0, "", err
	}
	return rec.Version, rec.Description, nil
}

func (m *Migrate) SetVersion(version uint64, description string) error {
	return m.db.C(m.migrationsCollection).Insert(versionRecord{
		Version:     version,
		Timestamp:   time.Now().UTC(),
		Description: description,
	})
}

func (m *Migrate) Up(n int) error {
	currentVersion, _, err := m.Version()
	if err != nil {
		return err
	}
	if n <= 0 || n > len(m.migrations) {
		n = len(m.migrations)
	}
	migrationSort(m.migrations)

	for i := 0; i < n; i++ {
		migration := m.migrations[i]
		if migration.Version <= currentVersion || migration.Up == nil {
			continue
		}
		if err := migration.Up(m.db); err != nil {
			return err
		}
		if err := m.SetVersion(migration.Version, migration.Description); err != nil {
			return err
		}

	}
	return nil
}

func (m *Migrate) Down(n int) error {
	currentVersion, _, err := m.Version()
	if err != nil {
		return err
	}
	migrationSort(m.migrations)

	currentVersionIndex := len(m.migrations) - 1
	for i := len(m.migrations) - 1; i >= 0; i-- {
		currentVersionIndex = i
		if m.migrations[i].Version < currentVersion {
			break
		}
	}

	if n < 0 || n > currentVersionIndex {
		n = currentVersionIndex
	}

	for i := n; i >= 0; i-- {
		migration := m.migrations[i]
		if migration.Down == nil {
			continue
		}
		if err := migration.Down(m.db); err != nil {
			return err
		}
		if err := m.SetVersion(migration.Version, migration.Description); err != nil {
			return err
		}
	}
	return nil
}
