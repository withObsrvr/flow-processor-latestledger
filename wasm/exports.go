//go:build wasmmodule
// +build wasmmodule

// Package wasm contains the WebAssembly module exports
package wasm

import (
	"encoding/json"
	"log"

	"github.com/stellar/go/xdr"

	"github.com/withObsrvr/flow-processor-latestledger/core"
)

// Status codes for error handling
const (
	STATUS_OK               int32 = 0
	STATUS_ERROR            int32 = 1
	STATUS_INVALID_CONFIG   int32 = 2
	STATUS_INVALID_LEDGER   int32 = 3
	STATUS_PROCESSING_ERROR int32 = 4
)

// Global processor instance
var processor *core.Processor

// ErrorMessage represents an error response
type ErrorMessage struct {
	Error   string `json:"error"`
	Code    int32  `json:"code"`
	Details string `json:"details,omitempty"`
}

//go:wasmexport initialize
func initialize(configJSON string) int32 {
	log.Printf("Initializing latest-ledger processor with config: %s", configJSON)

	// Parse configuration
	config, err := core.ParseConfig(configJSON)
	if err != nil {
		log.Printf("Failed to parse config: %v", err)
		return STATUS_INVALID_CONFIG
	}

	// Create processor instance
	processor = core.NewProcessor(config)
	return STATUS_OK
}

//go:wasmexport processLedger
func processLedger(ledgerJSON string) string {
	if processor == nil {
		errorMsg := ErrorMessage{
			Error: "Processor not initialized",
			Code:  STATUS_ERROR,
		}
		result, _ := json.Marshal(errorMsg)
		return string(result)
	}

	// Parse ledger data
	var ledgerData xdr.LedgerCloseMeta
	err := json.Unmarshal([]byte(ledgerJSON), &ledgerData)
	if err != nil {
		errorMsg := ErrorMessage{
			Error:   "Invalid ledger data",
			Code:    STATUS_INVALID_LEDGER,
			Details: err.Error(),
		}
		result, _ := json.Marshal(errorMsg)
		return string(result)
	}

	// Process ledger
	metrics, err := processor.ProcessLedger(ledgerData)
	if err != nil {
		errorMsg := ErrorMessage{
			Error:   "Failed to process ledger",
			Code:    STATUS_PROCESSING_ERROR,
			Details: err.Error(),
		}
		result, _ := json.Marshal(errorMsg)
		return string(result)
	}

	// Convert metrics to JSON
	result, err := metrics.ToJSON()
	if err != nil {
		errorMsg := ErrorMessage{
			Error:   "Failed to serialize metrics",
			Code:    STATUS_ERROR,
			Details: err.Error(),
		}
		errResult, _ := json.Marshal(errorMsg)
		return string(errResult)
	}

	return string(result)
}

//go:wasmexport getSchemaDefinition
func getSchemaDefinition() string {
	if processor == nil {
		return ""
	}
	return processor.GetSchemaDefinition()
}

//go:wasmexport getQueryDefinitions
func getQueryDefinitions() string {
	if processor == nil {
		return ""
	}
	return processor.GetQueryDefinitions()
}

//go:wasmexport version
func version() string {
	return "1.0.0" // Match the plugin version
}

//go:wasmexport name
func name() string {
	return "flow/processor/latest-ledger" // Match the plugin name
}
