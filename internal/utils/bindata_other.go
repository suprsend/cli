//go:build !(linux && amd64) && !(linux && arm64) && !(darwin && amd64) && !(darwin && arm64) && !(windows && amd64)

package utils

// TypeMorphBin is nil on unsupported platforms.
var TypeMorphBin []byte
