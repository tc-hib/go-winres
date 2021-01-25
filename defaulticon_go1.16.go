// +build go1.16

package main

import _ "embed"

//go:embed icon.png
var initIcon []byte

//go:embed icon16.png
var initIcon16 []byte
