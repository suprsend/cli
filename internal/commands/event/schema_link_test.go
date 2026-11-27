package event

import "testing"

// The API stores version_no: null when the file leaves it out.
func TestFingerprintSchemaLink(t *testing.T) {
	local := map[string]any{"name": "order_placed", "payload_schema": map[string]any{"schema": "Order-Placed"}}
	stored := map[string]any{"name": "ORDER_PLACED", "description": nil, "payload_schema": map[string]any{"schema": "order-placed", "version_no": nil}}
	if fingerprint(local) != fingerprint(stored) {
		t.Error("an unpinned schema link should match what the API stores")
	}
	noSchema := map[string]any{"name": "order_placed", "payload_schema": map[string]any{"schema": nil, "version_no": nil}}
	if fingerprint(noSchema) != fingerprint(map[string]any{"name": "ORDER_PLACED"}) {
		t.Error("a null schema should match no payload_schema")
	}
	if fingerprint(local) == fingerprint(noSchema) {
		t.Error("removing the schema should be a change")
	}
}
