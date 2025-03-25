//go:build wasmmodule
// +build wasmmodule

// Package main is the entry point for the latest-ledger processor WebAssembly module.
package main

// No imports needed for the WebAssembly main, as exports are defined in the wasm package

// Empty main function for the WebAssembly module
// The actual functionality is exposed via exported functions in the wasm package
func main() {
	// For WebAssembly modules, this function is not used
	// The module exports its own functions using go:wasmexport
}
