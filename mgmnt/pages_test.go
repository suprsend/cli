package mgmnt

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

// workspaceServer serves n workspaces, at most pageCap per page whatever
// limit asks for, with meta.count set to n.
func workspaceServer(t *testing.T, n, pageCap int) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		limit = min(limit, pageCap)
		results := []Workspace{}
		for i := offset; i < n && i < offset+limit; i++ {
			results = append(results, Workspace{Slug: fmt.Sprintf("ws-%d", i)})
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"results": results, "meta": map[string]int{"count": n}})
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestListWorkspacesPages(t *testing.T) {
	tests := []struct {
		name                 string
		total, pageCap       int
		limit, offset, wantN int
	}{
		// limit 0 lists everything, past the 20 that used to be the default.
		{"all", 38, 50, 0, 0, 38},
		// An API that returns fewer items than asked for doesn't end the list early.
		{"capped pages", 38, 10, 0, 0, 38},
		{"limit", 38, 50, 5, 0, 5},
		{"limit and offset", 38, 10, 20, 30, 8},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := workspaceServer(t, tt.total, tt.pageCap)
			c := NewClientWithUrls("token", srv.URL+"/", srv.URL+"/", false)
			resp, err := c.ListWorkspaces(tt.limit, tt.offset)
			if err != nil {
				t.Fatal(err)
			}
			if len(resp.Results) != tt.wantN || resp.Meta.Count != tt.total {
				t.Fatalf("got %d workspaces of %d, want %d of %d", len(resp.Results), resp.Meta.Count, tt.wantN, tt.total)
			}
			if tt.wantN > 0 && resp.Results[0].Slug != fmt.Sprintf("ws-%d", tt.offset) {
				t.Errorf("first workspace = %s, want ws-%d", resp.Results[0].Slug, tt.offset)
			}
		})
	}
}
