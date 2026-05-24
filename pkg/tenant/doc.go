// Package tenant carries per-request SuprSend credentials on a context.Context.
//
// Both the CLI (single tenant populated at startup) and multi-tenant HTTP
// servers using pkg/mcpserver (one tenant per session populated by the
// resolver) use this package. Tool handlers and
// utils.GetSuprSendWorkspaceClient read from it.
//
// Credentials carry the service token. The Credentials type implements
// fmt.Stringer (String()) and json.Marshaler (MarshalJSON) such that the
// service token is REDACTED in default print/marshal output — so accidental
// fmt.Sprintf("%+v", creds) or json.Marshal(creds) cannot leak the token to
// logs. Code that genuinely needs the raw token must read the field
// directly: creds.ServiceToken.
//
// Stability: this package is part of the v1 stability commitment for
// github.com/suprsend/cli. See CHANGELOG.md at the repo root for release notes.
package tenant
