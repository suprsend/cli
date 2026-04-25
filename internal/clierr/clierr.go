package clierr

import (
	"encoding/json"
	"errors"
)

// Error code constants — stable taxonomy from the CLI spec (§8).
const (
	CodeAuthMissingToken  = "auth-missing-token"
	CodeAuthInvalidToken  = "auth-invalid-token"
	CodeAuthForbidden     = "auth-forbidden"
	CodeConfigMissing     = "config-missing"
	CodeConfigInvalid     = "config-invalid"
	CodeSchemaValidation  = "schema-validation-failed"
	CodeFileNotFound      = "file-not-found"
	CodeFileParseFailed   = "file-parse-failed"
	CodeAPIConflict       = "api-conflict"
	CodeAPINotFound       = "api-not-found"
	CodeAPIRateLimited    = "api-rate-limited"
	CodeAPIInternal       = "api-internal"
	CodeDanglingReference = "dangling-reference"
	CodeDryRunOnly        = "dry-run-only"
	CodeInvalidUsage      = "invalid-usage"
	CodeUnknown           = "unknown"
)

// CLIError is a structured, machine-readable error whose JSON shape matches the spec:
//
//	{"error": "...", "code": "...", "resource": "...", "details": {...}, "hint": "..."}
type CLIError struct {
	Message  string         `json:"error"`
	Code     string         `json:"code"`
	Resource string         `json:"resource,omitempty"`
	Details  map[string]any `json:"details,omitempty"`
	Hint     string         `json:"hint,omitempty"`
}

// Error implements the error interface.
func (e *CLIError) Error() string { return e.Message }

// JSON returns the error marshalled to JSON bytes.
func (e *CLIError) JSON() []byte {
	b, _ := json.Marshal(e)
	return b
}

// WithResource sets the resource field (e.g. "workflows/welcome") and returns e.
func (e *CLIError) WithResource(resource string) *CLIError {
	e.Resource = resource
	return e
}

// WithDetails sets the details field and returns e.
func (e *CLIError) WithDetails(details map[string]any) *CLIError {
	e.Details = details
	return e
}

// WithHint sets the hint field and returns e.
func (e *CLIError) WithHint(hint string) *CLIError {
	e.Hint = hint
	return e
}

// New creates a CLIError with the given message and code.
func New(message, code string) *CLIError {
	return &CLIError{Message: message, Code: code}
}

// Wrap promotes a plain error into a CLIError with the given code and hint.
// If err already contains a *CLIError anywhere in its chain, that value is
// returned unchanged — preserving the original classification over the caller's.
func Wrap(err error, code, hint string) *CLIError {
	var ce *CLIError
	if errors.As(err, &ce) {
		return ce
	}
	return &CLIError{Message: err.Error(), Code: code, Hint: hint}
}
