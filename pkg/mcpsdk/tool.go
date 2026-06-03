// Package mcpsdk defines an SDK-agnostic representation of an MCP tool.
// Adapters in subpackages register these against a specific MCP runtime
// (currently only the official modelcontextprotocol/go-sdk in mcpsdk/official).
// Tool definitions and handlers live in internal/tools and depend only on this
// package, so they remain single-sourced across runtimes.
package mcpsdk

import (
	"context"

	"github.com/google/jsonschema-go/jsonschema"
)

// Tool is a runtime-agnostic MCP tool definition.
type Tool struct {
	Name        string
	Description string
	InputSchema *jsonschema.Schema
	Annotations Annotations
	// Meta is passed through to the protocol as the tool's `_meta` field.
	// Used by MCP Apps to attach `_meta.ui.resourceUri` etc.
	Meta    map[string]any
	Handler ToolHandler
}

// Annotations mirrors the MCP tool annotations vocabulary. Adapters translate
// these into the runtime-specific annotation type.
//
// DestructiveHint and OpenWorldHint are *bool to mirror the MCP spec's
// tri-state: nil means "unset" and the consumer applies the spec default
// (both default to true). A plain false would otherwise be indistinguishable
// from "author left it unset", incorrectly overriding the spec default. Set
// them via BoolPtr. ReadOnlyHint and IdempotentHint are plain bool because the
// spec defaults those to false, which matches Go's zero value.
type Annotations struct {
	Title           string
	ReadOnlyHint    bool
	IdempotentHint  bool
	DestructiveHint *bool // nil = unset (MCP spec default: true)
	OpenWorldHint   *bool // nil = unset (MCP spec default: true)
}

// BoolPtr returns a pointer to b. Convenience for setting the *bool annotation
// hints (DestructiveHint, OpenWorldHint) on a Tool.
func BoolPtr(b bool) *bool { return &b }

// ToolHandler is the SDK-independent handler signature. The handler receives a
// context (which carries tenant credentials when the request was authenticated
// — see pkg/tenant) and a parsed Args wrapper. Handlers return a Result
// or a Go error; adapters translate either into the runtime-specific result
// type. A returned error is treated as a protocol-level error; tool-side
// failures the LLM should see go in Result with IsError=true.
type ToolHandler func(ctx context.Context, args Args) (Result, error)

// Result is the runtime-agnostic shape of a tool response. Text is the
// primary unstructured content; Structured is optional structured output
// (becomes the tool result's structuredContent field).
type Result struct {
	Text       string
	Structured any
	IsError    bool
}
