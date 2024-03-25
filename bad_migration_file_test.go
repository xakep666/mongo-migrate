package migrate

import (
	"context"
	"testing"

	"go.mongodb.org/mongo-driver/mongo"
)

func TestBadMigrationFile(t *testing.T) {
	oldMigrate := globalMigrate
	defer func() {
		globalMigrate = oldMigrate
	}()
	globalMigrate = NewMigrate(nil)

	err := Register(func(ctx context.Context, db *mongo.Database) error {
		return nil
	}, func(ctx context.Context, db *mongo.Database) error {
		return nil
	})
	if err == nil {
		t.Errorf("Unexpected nil error")
	}
}

func TestBadMigrationFilePanic(t *testing.T) {
	oldMigrate := globalMigrate
	defer func() {
		globalMigrate = oldMigrate
		if r := recover(); r == nil {
			t.Errorf("Unexpectedly no panic recovered")
		}
	}()
	globalMigrate = NewMigrate(nil)
	MustRegister(func(ctx context.Context, db *mongo.Database) error {
		return nil
	}, func(ctx context.Context, db *mongo.Database) error {
		return nil
	})
}
