package codeversion

import (
	"fmt"
	"runtime"
)

var Version string

func GetGoRuntimeVersion() string {
	return fmt.Sprintf("Go version: %s %s/%s", runtime.Version(), runtime.GOOS, runtime.GOARCH)
}

func GetCodeVersion(app string) string {
	if app == "" {
		app = "App"
	}
	return fmt.Sprintf("%s version: %s", app, Version)
}
