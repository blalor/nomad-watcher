// +build tools

package main

/*
    This file exists to capture dependencies on runtime tools that are not
    actually used in any source.

    Go is ridiculously picky about the +build and package lines; really needs to
    be exactly as above!

    ref: https://github.com/golang/go/issues/25922#issuecomment-412992431
    ref: https://github.com/golang/go/wiki/Modules#how-can-i-track-tool-dependencies-for-a-module
*/

import (
    _ "github.com/onsi/ginkgo/ginkgo"
    _ "github.com/vektra/mockery"
)
