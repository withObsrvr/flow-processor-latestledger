// plugin_latestledger.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/stellar/go/ingest"
	"github.com/stellar/go/ingest/ledger"
	"github.com/stellar/go/xdr"

	// Import the core plugin API.
	"github.com/withObsrvr/pluginapi"
)

// LatestLedger holds metrics extracted from a ledger.
type LatestLedger struct {
	Sequence          uint32    `json:"sequence"`
	Hash              string    `json:"hash"`
	TransactionCount  int       `json:"transaction_count"`
	OperationCount    int       `json:"operation_count"`
	SuccessfulTxCount int       `json:"successful_tx_count"`
	FailedTxCount     int       `json:"failed_tx_count"`
	TotalFeeCharged   int64     `json:"total_fee_charged"`
	ClosedAt          time.Time `json:"closed_at"`
	BaseFee           uint32    `json:"base_fee"`

	// Soroban metrics
	SorobanTxCount            int    `json:"soroban_tx_count"`
	TotalSorobanFees          int64  `json:"total_soroban_fees"`
	TotalResourceInstructions uint64 `json:"total_resource_instructions"`

	SkippedTxCount int `json:"skipped_tx_count"`
	UnknownTxCount int `json:"unknown_tx_count"`
}

// LatestLedgerProcessor implements both pluginapi.Processor and pluginapi.ConsumerRegistry
type LatestLedgerProcessor struct {
	networkPassphrase string
	consumers         []pluginapi.Consumer  // downstream consumers
	processors        []pluginapi.Processor // downstream processors
}

// GetSchemaDefinition returns GraphQL type definitions for this plugin
func (p *LatestLedgerProcessor) GetSchemaDefinition() string {
	return `
type LatestLedger {
    sequence: Int!
    hash: String!
    transactionCount: Int!
    operationCount: Int!
    successfulTxCount: Int!
    failedTxCount: Int!
    totalFeeCharged: String!
    closedAt: String!
    baseFee: Int!
    sorobanTxCount: Int!
    totalSorobanFees: String!
    totalResourceInstructions: String!
    skippedTxCount: Int!
    unknownTxCount: Int!
}
`
}

// GetQueryDefinitions returns GraphQL query definitions for this plugin
func (p *LatestLedgerProcessor) GetQueryDefinitions() string {
	return `
    latestLedger: LatestLedger
    ledgerBySequence(sequence: Int!): LatestLedger
`
}

// RegisterConsumer registers a downstream consumer
func (p *LatestLedgerProcessor) RegisterConsumer(consumer pluginapi.Consumer) {
	log.Printf("LatestLedgerProcessor: Registering consumer %s", consumer.Name())
	p.consumers = append(p.consumers, consumer)
}

// Subscribe registers a downstream processor (keeping existing method for compatibility)
func (p *LatestLedgerProcessor) Subscribe(proc pluginapi.Processor) {
	log.Printf("LatestLedgerProcessor: Registering processor %s", proc.Name())
	p.processors = append(p.processors, proc)
}

// Process implements the core logic
func (p *LatestLedgerProcessor) Process(ctx context.Context, msg pluginapi.Message) error {
	log.Printf("LatestLedgerProcessor: Processing message with %d consumers and %d processors",
		len(p.consumers), len(p.processors))

	ledgerCloseMeta, ok := msg.Payload.(xdr.LedgerCloseMeta)
	if !ok {
		return fmt.Errorf("expected xdr.LedgerCloseMeta, got %T", msg.Payload)
	}

	// Create a transaction reader using the network passphrase.
	txReader, err := ingest.NewLedgerTransactionReaderFromLedgerCloseMeta(
		p.networkPassphrase,
		ledgerCloseMeta,
	)
	if err != nil {
		return fmt.Errorf("error creating transaction reader: %v", err)
	}
	defer txReader.Close()

	// Extract basic ledger metrics.
	metrics := LatestLedger{
		Sequence: ledger.Sequence(ledgerCloseMeta),
		Hash:     ledger.Hash(ledgerCloseMeta),
		BaseFee:  ledger.BaseFee(ledgerCloseMeta),
		ClosedAt: ledger.ClosedAt(ledgerCloseMeta),
	}

	// Process each transaction. Skip transactions with "unknown tx hash" errors.
	for {
		tx, err := txReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			if strings.Contains(err.Error(), "unknown tx hash") {
				log.Printf("Warning: transaction with unknown hash found in ledger %d", metrics.Sequence)
				// Still increment transaction count even for unknown transactions
				metrics.TransactionCount++
				metrics.FailedTxCount++
				continue
			}
			return fmt.Errorf("error reading transaction: %v", err)
		}

		metrics.TransactionCount++
		metrics.OperationCount += len(tx.Envelope.Operations())
		metrics.TotalFeeCharged += int64(tx.Result.Result.FeeCharged)

		if tx.Result.Successful() {
			metrics.SuccessfulTxCount++
		} else {
			metrics.FailedTxCount++
		}

		// Process Soroban metrics, if present.
		if hasSorobanTransaction(tx) {
			metrics.SorobanTxCount++
			sMetrics := getSorobanMetrics(tx)
			metrics.TotalSorobanFees += sMetrics.resourceFee
			metrics.TotalResourceInstructions += uint64(sMetrics.instructions)
		}
	}

	// Calculate success rate safely to avoid division by zero
	var successRate float64
	if metrics.TransactionCount > 0 {
		successRate = (float64(metrics.SuccessfulTxCount) / float64(metrics.TransactionCount)) * 100
	}

	log.Printf("Latest ledger: %d (Transactions: %d, Operations: %d, Success Rate: %.2f%%)",
		metrics.Sequence,
		metrics.TransactionCount,
		metrics.OperationCount,
		successRate,
	)

	// Marshal metrics to JSON.
	jsonBytes, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("error marshaling latest ledger: %w", err)
	}

	// Create forward message
	forwardMsg := pluginapi.Message{
		Payload:   jsonBytes,
		Timestamp: msg.Timestamp,
		Metadata: map[string]interface{}{
			"ledger_sequence": metrics.Sequence,
			"source":          "latest-ledger-processor",
			"data_type":       "latest_ledger",
		},
	}

	// Forward to consumers
	for i, consumer := range p.consumers {
		log.Printf("LatestLedgerProcessor: Forwarding to consumer %d: %s", i, consumer.Name())
		if err := consumer.Process(ctx, forwardMsg); err != nil {
			log.Printf("Error in consumer %s: %v", consumer.Name(), err)
		}
	}

	// Forward to processors
	for i, proc := range p.processors {
		log.Printf("LatestLedgerProcessor: Forwarding to processor %d: %s", i, proc.Name())
		if err := proc.Process(ctx, forwardMsg); err != nil {
			log.Printf("Error in processor %s: %v", proc.Name(), err)
		}
	}

	return nil
}

// Helper types and functions for Soroban metrics.
type sorobanMetrics struct {
	resourceFee  int64
	instructions uint32
	readBytes    uint32
	writeBytes   uint32
}

func hasSorobanTransaction(tx ingest.LedgerTransaction) bool {
	switch tx.Envelope.Type {
	case xdr.EnvelopeTypeEnvelopeTypeTx:
		_, has := tx.Envelope.V1.Tx.Ext.GetSorobanData()
		return has
	case xdr.EnvelopeTypeEnvelopeTypeTxFeeBump:
		_, has := tx.Envelope.FeeBump.Tx.InnerTx.V1.Tx.Ext.GetSorobanData()
		return has
	}
	return false
}

func getSorobanMetrics(tx ingest.LedgerTransaction) sorobanMetrics {
	var sorobanData xdr.SorobanTransactionData
	var sMetrics sorobanMetrics

	switch tx.Envelope.Type {
	case xdr.EnvelopeTypeEnvelopeTypeTx:
		sorobanData, _ = tx.Envelope.V1.Tx.Ext.GetSorobanData()
	case xdr.EnvelopeTypeEnvelopeTypeTxFeeBump:
		sorobanData, _ = tx.Envelope.FeeBump.Tx.InnerTx.V1.Tx.Ext.GetSorobanData()
	}

	sMetrics.resourceFee = int64(sorobanData.ResourceFee)
	sMetrics.instructions = uint32(sorobanData.Resources.Instructions)
	sMetrics.readBytes = uint32(sorobanData.Resources.ReadBytes)
	sMetrics.writeBytes = uint32(sorobanData.Resources.WriteBytes)

	return sMetrics
}

// Exported New function to allow dynamic loading.
// When the plugin manager loads the shared object, it calls New() to obtain a new instance.
func New() pluginapi.Plugin {
	return &LatestLedgerProcessor{}
}

// NewLatestLedgerProcessor creates a new LatestLedgerProcessor from configuration.
func NewLatestLedgerProcessor(config map[string]interface{}) (*LatestLedgerProcessor, error) {
	networkPassphrase, ok := config["network_passphrase"].(string)
	if !ok {
		return nil, fmt.Errorf("missing network_passphrase in config")
	}
	return &LatestLedgerProcessor{
		networkPassphrase: networkPassphrase,
		consumers:         make([]pluginapi.Consumer, 0),
		processors:        make([]pluginapi.Processor, 0),
	}, nil
}

// Name returns the plugin's name following the naming convention.
func (p *LatestLedgerProcessor) Name() string {
	return "flow/processor/latest-ledger"
}

// Version returns the plugin's version.
func (p *LatestLedgerProcessor) Version() string {
	return "1.0.0"
}

// Type returns the plugin type.
func (p *LatestLedgerProcessor) Type() pluginapi.PluginType {
	return pluginapi.ProcessorPlugin
}

// Initialize configures the processor using the provided config map.
func (p *LatestLedgerProcessor) Initialize(config map[string]interface{}) error {
	processor, err := NewLatestLedgerProcessor(config)
	if err != nil {
		return err
	}
	*p = *processor
	return nil
}
