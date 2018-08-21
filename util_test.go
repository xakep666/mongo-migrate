package migrate

import "testing"

func TestExtractVersionDescription(t *testing.T) {
	version, description, err := extractVersionDescription("1_test.go")
	if err != nil {
		t.Error(err)
	}
	if version != 1 || description != "test" {
		t.Errorf("Bad version/description: %v %v", version, description)
	}

	_, _, err = extractVersionDescription("test.go")
	if err == nil {
		t.Errorf("Unexpected nil error")
	}

	_, _, err = extractVersionDescription("test")
	if err == nil {
		t.Errorf("Unexpected nil error")
	}

	_, _, err = extractVersionDescription("test_test.go")
	if err == nil {
		t.Errorf("Unexpected nil error")
	}
}
