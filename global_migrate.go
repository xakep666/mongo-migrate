package mongo_migrate

import (
	"fmt"
	"runtime"

	"github.com/globalsign/mgo"
)

var globalMigrate = NewMigrate(nil)

func internalRegister(up, down MigrationFunc, skip int) error {
	_, file, _, _ := runtime.Caller(skip)
	version, description, err := extractVersionDescription(file)
	if err != nil {
		return err
	}
	if hasVersion(globalMigrate.migrations, version) {
		return fmt.Errorf("migration with version %v already registered", version)
	}
	globalMigrate.migrations = append(globalMigrate.migrations, Migration{
		Version:     version,
		Description: description,
		Up:          up,
		Down:        down,
	})
	return nil
}

func Register(up, down MigrationFunc) error {
	return internalRegister(up, down, 2)
}

func MustRegister(up, down MigrationFunc) {
	if err := internalRegister(up, down, 2); err != nil {
		panic(err)
	}
}

func RegisteredMigrations() []Migration {
	ret := make([]Migration, len(globalMigrate.migrations))
	copy(ret, globalMigrate.migrations)
	return ret
}

func SetDatabase(db *mgo.Database) {
	globalMigrate.db = db
}

func SetMigrationsCollection(name string) {
	globalMigrate.SetMigrationsCollection(name)
}

func Version() (uint64, string, error) {
	return globalMigrate.Version()
}

func Up(n int) error {
	return globalMigrate.Up(n)
}

func Down(n int) error {
	return globalMigrate.Down(n)
}
