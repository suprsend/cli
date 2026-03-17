//go:build windows && amd64

package utils

import _ "embed"

//go:embed embedded-binaries/type-morph-windows-amd64.exe
var TypeMorphBin []byte
