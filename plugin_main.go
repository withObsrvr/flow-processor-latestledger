//go:build goplugin
// +build goplugin

// Package main is the entry point for the latest-ledger processor Go plugin.
package main

import (
	"github.com/withObsrvr/flow-processor-latestledger/plugin"
	"github.com/withObsrvr/pluginapi"
)

// For Go plugins, this function is called by the plugin loader
func main() {
	// Empty main function
	// Go plugins need a main package, but don't use main()
}

// New is exported and called by the plugin loader to create a new instance
func New() pluginapi.Plugin {
	return plugin.New()
}
