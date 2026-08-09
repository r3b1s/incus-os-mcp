package server

import (
	"encoding/json"
	"testing"
)

func TestListOutputMarshalsAsObjectEnvelope(t *testing.T) {
	t.Parallel()

	data, err := json.Marshal(ListOutput[string]{Items: []string{"one", "two"}})
	if err != nil {
		t.Fatalf("marshal ListOutput: %v", err)
	}

	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatalf("unmarshal ListOutput: %v", err)
	}
	object, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("collection result must be an object, got %T", value)
	}
	items, ok := object["items"].([]any)
	if !ok || len(items) != 2 {
		t.Fatalf("object envelope must contain two items, got %#v", object["items"])
	}
}

func TestValidateProjectScope(t *testing.T) {
	t.Parallel()

	if err := validateProjectScope("lab", true); err == nil {
		t.Fatal("project combined with all_projects must be rejected")
	}
	for _, scope := range []struct {
		project string
		all     bool
	}{
		{},
		{project: "lab"},
		{all: true},
	} {
		if err := validateProjectScope(scope.project, scope.all); err != nil {
			t.Fatalf("valid scope project=%q all=%t: %v", scope.project, scope.all, err)
		}
	}
}
