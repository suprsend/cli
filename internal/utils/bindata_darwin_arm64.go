//go:build darwin && arm64

package utils

import _ "embed"

//go:embed embedded-binaries/type-morph-darwin-arm64
var TypeMorphBin []byte
