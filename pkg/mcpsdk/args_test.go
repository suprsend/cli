package mcpsdk

import (
	"encoding/json"
	"testing"
)

func TestArgs_EmptyAndNull(t *testing.T) {
	cases := []json.RawMessage{nil, []byte(""), []byte("null")}
	for _, raw := range cases {
		a, err := NewArgs(raw)
		if err != nil {
			t.Fatalf("NewArgs(%q) returned error: %v", string(raw), err)
		}
		if got := a.GetString("anything", "fallback"); got != "fallback" {
			t.Errorf("GetString on empty args = %q, want %q", got, "fallback")
		}
	}
}

func TestArgs_InvalidJSON(t *testing.T) {
	if _, err := NewArgs([]byte("not json")); err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestArgs_RequireString(t *testing.T) {
	a, err := NewArgs([]byte(`{"name": "alice", "count": 3}`))
	if err != nil {
		t.Fatal(err)
	}
	got, err := a.RequireString("name")
	if err != nil || got != "alice" {
		t.Errorf("RequireString(name) = %q, %v; want alice, nil", got, err)
	}
	if _, err := a.RequireString("missing"); err == nil {
		t.Error("RequireString(missing) returned nil error")
	}
	if _, err := a.RequireString("count"); err == nil {
		t.Error("RequireString(count) on number returned nil error")
	}
}

func TestArgs_GetIntAndBool(t *testing.T) {
	a, err := NewArgs([]byte(`{"limit": 25, "active": true}`))
	if err != nil {
		t.Fatal(err)
	}
	if got := a.GetInt("limit", 10); got != 25 {
		t.Errorf("GetInt(limit) = %d, want 25", got)
	}
	if got := a.GetInt("missing", 7); got != 7 {
		t.Errorf("GetInt default = %d, want 7", got)
	}
	if got := a.GetBool("active", false); got != true {
		t.Errorf("GetBool(active) = %v, want true", got)
	}
}

func TestArgs_GetMapAndArray(t *testing.T) {
	a, err := NewArgs([]byte(`{"obj": {"k": "v"}, "arr": [1, 2, 3]}`))
	if err != nil {
		t.Fatal(err)
	}
	m, ok := a.GetMap("obj")
	if !ok || m["k"] != "v" {
		t.Errorf("GetMap(obj) = %v, %v; want {k:v}, true", m, ok)
	}
	arr, ok := a.GetArray("arr")
	if !ok || len(arr) != 3 {
		t.Errorf("GetArray(arr) = %v, %v; want length 3, true", arr, ok)
	}
	if _, ok := a.GetMap("missing"); ok {
		t.Error("GetMap(missing) returned ok=true")
	}
}
