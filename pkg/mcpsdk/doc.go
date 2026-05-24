// Package mcpsdk defines an SDK-agnostic representation of an MCP tool.
//
// The closed-source hosted MCP server (and any future OSS-library consumer)
// constructs *Tool values here; internal adapters convert them to the runtime
// types of the official modelcontextprotocol/go-sdk.
//
// Stability: this package is part of the v1 stability commitment for
// github.com/suprsend/cli. Breaking changes require a v2 major bump (which
// would change the import path to github.com/suprsend/cli/v2/pkg/mcpsdk).
// See CHANGELOG.md at the repo root for release notes.
package mcpsdk
