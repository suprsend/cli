package event

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/suprsend/cli/internal/utils"
	"github.com/suprsend/cli/mgmnt"
)

// Only new or changed events are sent; names match case-insensitively,
// since the API stores them in upper case.
func TestPlanEvents(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("offset") != "0" {
			_, _ = w.Write([]byte(`{"results":[],"meta":{"count":2}}`))
			return
		}
		_, _ = w.Write([]byte(`{"results":[
			{"name":"ORDER_PLACED","description":"placed","payload_schema":null,"created_at":"2026-01-01"},
			{"name":"ORDER_SHIPPED","description":"shipped","payload_schema":{"schema":"order","version_no":1}}
		],"meta":{"count":2}}`))
	}))
	defer srv.Close()
	client := mgmnt.NewClientWithUrls("token", srv.URL+"/", srv.URL+"/", false)

	events := []any{
		map[string]any{"name": "ORDER_PLACED", "description": "placed", "payload_schema": nil, "created_at": "2026-02-02"},
		map[string]any{"name": "order_shipped", "description": "shipped!", "payload_schema": map[string]any{"schema": "order", "version_no": float64(1)}},
		map[string]any{"name": "ORDER_RETURNED", "description": "returned"},
	}
	statuses, toSend, err := planEvents(client, "staging", events)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{utils.StatusUnchanged, utils.StatusUpdated, utils.StatusCreated}
	if strings.Join(statuses, ",") != strings.Join(want, ",") {
		t.Errorf("statuses = %v, want %v", statuses, want)
	}
	if len(toSend) != 2 || EventName(toSend, 0) != "order_shipped" || EventName(toSend, 1) != "ORDER_RETURNED" {
		t.Errorf("toSend = %v, want order_shipped and ORDER_RETURNED", toSend)
	}
}
