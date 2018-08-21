package migrate

import (
	"sort"
	"testing"
)

func TestMigrationSort(t *testing.T) {
	migrations := []Migration{
		{Version: 10, Description: "10"},
		{Version: 2, Description: "2"},
		{Version: 4, Description: "4"},
		{Version: 8, Description: "8"},
	}
	migrationSort(migrations)
	if !sort.SliceIsSorted(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	}) {
		t.Errorf("Unexpected unsorted array")
	}
}

func TestHasVersion(t *testing.T) {
	migrations := []Migration{
		{Version: 10, Description: "10"},
		{Version: 2, Description: "2"},
		{Version: 4, Description: "4"},
		{Version: 8, Description: "8"},
	}
	if !hasVersion(migrations, 2) {
		t.Errorf("Unexpectedly not found version")
	}
	if hasVersion(migrations, 3) {
		t.Errorf("Unexpectedly found version")
	}
}
