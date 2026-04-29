//go:build linux && amd64

package utils

import _ "embed"

//go:embed embedded-binaries/type-morph-linux-amd64
var TypeMorphBin []byte
