// +build integration

package migrate

import (
	"testing"
)

func TestGlobalMigrateUp(t *testing.T) {
	defer cleanup(db)
	SetDatabase(db)

	if err := Up(-1); err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	version, description, err := Version()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	if version != 2 || description != "sample_index_test" {
		t.Errorf("Unexpected version/description: %v %v", version, description)
		return
	}
}

func TestGlobalMigrateDown(t *testing.T) {
	defer cleanup(db)
	SetDatabase(db)

	if err := Up(-1); err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	if err := Down(-1); err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	version, _, err := Version()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	if version != 0 {
		t.Errorf("Unexpected version: %v", version)
		return
	}
}
