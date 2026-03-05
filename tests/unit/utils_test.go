package unit_test

import (
	"testing"

	"github.com/suprsend/cli/internal/utils"
)

func TestGetExtensionForLanguage(t *testing.T) {
	tests := []struct {
		lang string
		want string
	}{
		{"python", ".py"},
		{"java", ".java"},
		{"typescript", ".ts"},
		{"typescript-zod", ".ts"},
		{"go", ".go"},
		{"kotlin", ".kt"},
		{"swift", ".swift"},
		{"dart", ".dart"},
		{"unknown", ""},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.lang, func(t *testing.T) {
			got := utils.GetExtensionForLanguage(tt.lang)
			if got != tt.want {
				t.Errorf("GetExtensionForLanguage(%q) = %q, want %q", tt.lang, got, tt.want)
			}
		})
	}
}

func TestGetStringPtr(t *testing.T) {
	t.Run("key exists string", func(t *testing.T) {
		m := map[string]any{"name": "alice"}
		got := utils.GetStringPtr(m, "name")
		if got == nil || *got != "alice" {
			t.Errorf("expected pointer to 'alice', got %v", got)
		}
	})
	t.Run("key exists non-string", func(t *testing.T) {
		m := map[string]any{"count": 42}
		got := utils.GetStringPtr(m, "count")
		if got != nil {
			t.Errorf("expected nil, got %v", *got)
		}
	})
	t.Run("missing key", func(t *testing.T) {
		m := map[string]any{"name": "alice"}
		got := utils.GetStringPtr(m, "missing")
		if got != nil {
			t.Errorf("expected nil, got %v", *got)
		}
	})
}

func TestGetMap(t *testing.T) {
	t.Run("key exists map", func(t *testing.T) {
		inner := map[string]any{"a": 1}
		m := map[string]any{"data": inner}
		got := utils.GetMap(m, "data")
		if got == nil {
			t.Fatal("expected map, got nil")
		}
		if got["a"] != 1 {
			t.Errorf("expected a=1, got %v", got["a"])
		}
	})
	t.Run("key exists non-map", func(t *testing.T) {
		m := map[string]any{"data": "not a map"}
		got := utils.GetMap(m, "data")
		if got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})
	t.Run("missing key", func(t *testing.T) {
		m := map[string]any{}
		got := utils.GetMap(m, "missing")
		if got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})
}

func TestStringSchema(t *testing.T) {
	result := utils.StringSchema("a description")
	if result["type"] != "string" {
		t.Errorf("type = %v, want 'string'", result["type"])
	}
	if result["description"] != "a description" {
		t.Errorf("description = %v, want 'a description'", result["description"])
	}
}

func TestBoolSchema(t *testing.T) {
	result := utils.BoolSchema("a bool desc")
	if result["type"] != "boolean" {
		t.Errorf("type = %v, want 'boolean'", result["type"])
	}
	if result["description"] != "a bool desc" {
		t.Errorf("description = %v, want 'a bool desc'", result["description"])
	}
}

func TestArraySchema(t *testing.T) {
	result := utils.ArraySchema("an array desc")
	if result["type"] != "array" {
		t.Errorf("type = %v, want 'array'", result["type"])
	}
	if result["description"] != "an array desc" {
		t.Errorf("description = %v, want 'an array desc'", result["description"])
	}
	items, ok := result["items"].(map[string]any)
	if !ok {
		t.Fatal("items should be a map")
	}
	if items["type"] != "string" {
		t.Errorf("items.type = %v, want 'string'", items["type"])
	}
}

func TestRequiresKey(t *testing.T) {
	trueActions := []string{"set", "append", "increment", "unset", "remove"}
	for _, action := range trueActions {
		if !utils.RequiresKey(action) {
			t.Errorf("RequiresKey(%q) = false, want true", action)
		}
	}
	falseActions := []string{"add_email", "upsert", "add_sms", "remove_email", ""}
	for _, action := range falseActions {
		if utils.RequiresKey(action) {
			t.Errorf("RequiresKey(%q) = true, want false", action)
		}
	}
}

func TestRequiresValue(t *testing.T) {
	trueActions := []string{"set", "append", "increment"}
	for _, action := range trueActions {
		if !utils.RequiresValue(action) {
			t.Errorf("RequiresValue(%q) = false, want true", action)
		}
	}
	falseActions := []string{"unset", "remove", "add_email", ""}
	for _, action := range falseActions {
		if utils.RequiresValue(action) {
			t.Errorf("RequiresValue(%q) = true, want false", action)
		}
	}
}

func TestToStringSlice(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		input := []any{"a", "b", "c"}
		result, err := utils.ToStringSlice(input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 3 || result[0] != "a" || result[1] != "b" || result[2] != "c" {
			t.Errorf("result = %v, want [a b c]", result)
		}
	})
	t.Run("mixed types", func(t *testing.T) {
		input := []any{"a", 42}
		_, err := utils.ToStringSlice(input)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
	t.Run("empty", func(t *testing.T) {
		input := []any{}
		result, err := utils.ToStringSlice(input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 0 {
			t.Errorf("result = %v, want empty", result)
		}
	})
}

func TestRemarshal(t *testing.T) {
	t.Run("map to struct", func(t *testing.T) {
		type Target struct {
			Name string `json:"name"`
			Age  int    `json:"age"`
		}
		src := map[string]any{"name": "Alice", "age": float64(30)}
		var dst Target
		if err := utils.Remarshal(src, &dst); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if dst.Name != "Alice" || dst.Age != 30 {
			t.Errorf("dst = %+v, want {Alice 30}", dst)
		}
	})
	t.Run("struct to struct", func(t *testing.T) {
		type Src struct {
			Name string `json:"name"`
		}
		type Dst struct {
			Name string `json:"name"`
		}
		var dst Dst
		if err := utils.Remarshal(Src{Name: "Bob"}, &dst); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if dst.Name != "Bob" {
			t.Errorf("name = %q, want %q", dst.Name, "Bob")
		}
	})
}
