//go:build darwin && amd64

package utils

import _ "embed"

//go:embed embedded-binaries/type-morph-darwin-amd64
var TypeMorphBin []byte
