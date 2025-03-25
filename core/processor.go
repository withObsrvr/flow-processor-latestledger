// Package core contains the shared business logic for the latest ledger processor
package core

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/stellar/go/ingest"
	"github.com/stellar/go/ingest/ledger"
	"github.com/stellar/go/xdr"
)

// LatestLedger holds metrics extracted from a ledger.
type LatestLedger struct {
	Sequence                 uint32    `json:"sequence"`
	Hash                     string    `json:"hash"`
	TransactionCount         int       `json:"transaction_count"`
	TxSetOperationCount      int       `json:"tx_set_operation_count"`     // All operations submitted to the ledger
	SuccessfulOperationCount int       `json:"successful_operation_count"` // Operations from successful transactions
	SuccessfulTxCount        int       `json:"successful_tx_count"`
	FailedTxCount            int       `json:"failed_tx_count"`
	TotalFeeCharged          int64     `json:"total_fee_charged"`
	ClosedAt                 time.Time `json:"closed_at"`
	BaseFee                  uint32    `json:"base_fee"`

	// Operations per second (called transactions per second in other blockchains)
	TransactionsPerSecond float64 `json:"transactions_per_second"`

	// Soroban metrics
	SorobanTxCount            int    `json:"soroban_tx_count"`
	TotalSorobanFees          int64  `json:"total_soroban_fees"`
	TotalResourceInstructions uint64 `json:"total_resource_instructions"`

	SkippedTxCount int `json:"skipped_tx_count"`
	UnknownTxCount int `json:"unknown_tx_count"`
}

// Helper types and functions for Soroban metrics.
type sorobanMetrics struct {
	resourceFee  int64
	instructions uint32
	readBytes    uint32
	writeBytes   uint32
}

// Config holds the configuration for the latest ledger processor.
type Config struct {
	NetworkPassphrase string `json:"network_passphrase"`
}

// Processor contains the core logic for the latest ledger processor.
type Processor struct {
	NetworkPassphrase       string
	PreviousLedgerCloseTime time.Time
}

// NewProcessor creates a new latest ledger processor.
func NewProcessor(config Config) *Processor {
	return &Processor{
		NetworkPassphrase: config.NetworkPassphrase,
	}
}

// ProcessLedger processes a ledger and returns metrics.
func (p *Processor) ProcessLedger(ledgerCloseMeta xdr.LedgerCloseMeta) (*LatestLedger, error) {
	// Create a transaction reader using the network passphrase.
	txReader, err := ingest.NewLedgerTransactionReaderFromLedgerCloseMeta(
		p.NetworkPassphrase,
		ledgerCloseMeta,
	)
	if err != nil {
		return nil, fmt.Errorf("error creating transaction reader: %v", err)
	}
	defer txReader.Close()

	// Extract basic ledger metrics.
	metrics := LatestLedger{
		Sequence: ledger.Sequence(ledgerCloseMeta),
		Hash:     ledger.Hash(ledgerCloseMeta),
		BaseFee:  ledger.BaseFee(ledgerCloseMeta),
		ClosedAt: ledger.ClosedAt(ledgerCloseMeta),
	}

	// Process each transaction.
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
				metrics.UnknownTxCount++
				continue
			}
			return nil, fmt.Errorf("error reading transaction: %v", err)
		}

		metrics.TransactionCount++
		operationCount := len(tx.Envelope.Operations())
		metrics.TxSetOperationCount += operationCount
		metrics.TotalFeeCharged += int64(tx.Result.Result.FeeCharged)

		if tx.Result.Successful() {
			metrics.SuccessfulTxCount++
			metrics.SuccessfulOperationCount += operationCount
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

	// Calculate transactions per second (operations per second in Stellar terms)
	// Using successful operations for TPS calculation as it better represents actual throughput
	if !p.PreviousLedgerCloseTime.IsZero() {
		// Calculate the time difference between the current and previous ledger
		timeDiff := metrics.ClosedAt.Sub(p.PreviousLedgerCloseTime).Seconds()
		if timeDiff > 0 {
			metrics.TransactionsPerSecond = float64(metrics.SuccessfulOperationCount) / timeDiff
		} else {
			// Default fallback - Stellar's target is ~5 second ledger close time
			metrics.TransactionsPerSecond = float64(metrics.SuccessfulOperationCount) / 5.0
		}
	} else {
		// For the first ledger processed, use a reasonable approximation
		// Stellar's historical ledger close time is ~5 seconds
		metrics.TransactionsPerSecond = float64(metrics.SuccessfulOperationCount) / 5.0
	}

	// Update the previous close time for next calculation
	p.PreviousLedgerCloseTime = metrics.ClosedAt

	return &metrics, nil
}

// GetSchemaDefinition returns GraphQL type definitions for this processor.
func (p *Processor) GetSchemaDefinition() string {
	return `
type LatestLedger {
    sequence: Int!
    hash: String!
    transactionCount: Int!
    txSetOperationCount: Int!
    successfulOperationCount: Int!
    successfulTxCount: Int!
    failedTxCount: Int!
    totalFeeCharged: String!
    closedAt: String!
    baseFee: Int!
    transactionsPerSecond: Float!
    sorobanTxCount: Int!
    totalSorobanFees: String!
    totalResourceInstructions: String!
    skippedTxCount: Int!
    unknownTxCount: Int!
}
`
}

// GetQueryDefinitions returns GraphQL query definitions for this processor.
func (p *Processor) GetQueryDefinitions() string {
	return `
    latestLedger: LatestLedger
    ledgerBySequence(sequence: Int!): LatestLedger
`
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

// ToJSON converts the processor metrics to JSON.
func (l *LatestLedger) ToJSON() ([]byte, error) {
	return json.Marshal(l)
}

// ParseConfig parses a JSON configuration string into a Config struct.
func ParseConfig(configJSON string) (Config, error) {
	var config Config
	err := json.Unmarshal([]byte(configJSON), &config)
	return config, err
}
