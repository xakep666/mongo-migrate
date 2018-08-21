package mongo_migrate

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

func extractVersionDescription(name string) (uint64, string, error) {
	base := filepath.Base(name)

	if ext := filepath.Ext(base); ext != ".go" {
		return 0, "", fmt.Errorf("can not extract version from %q", base)
	}

	idx := strings.IndexByte(base, '_')
	if idx == -1 {
		return 0, "", fmt.Errorf("can not extract version from %q", base)
	}

	version, err := strconv.ParseUint(base[:idx], 10, 64)
	if err != nil {
		return 0, "", err
	}

	description := base[idx : len(base)-len(".go")-1]

	return version, description, nil
}
