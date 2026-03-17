//go:build linux && arm64

package utils

import _ "embed"

//go:embed embedded-binaries/type-morph-linux-arm64
var TypeMorphBin []byte
