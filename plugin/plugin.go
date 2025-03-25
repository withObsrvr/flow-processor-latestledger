//go:build goplugin
// +build goplugin

// Package plugin contains the Go plugin implementation
package plugin

import (
	"context"
	"fmt"
	"log"

	"github.com/stellar/go/xdr"
	"github.com/withObsrvr/pluginapi"

	"github.com/withObsrvr/flow-processor-latestledger/core"
)

// PluginAdapter adapts our core processor to the pluginapi.Plugin interface
type PluginAdapter struct {
	processor         *core.Processor
	networkPassphrase string
	consumers         []pluginapi.Consumer  // downstream consumers
	processors        []pluginapi.Processor // downstream processors
}

// New creates a new instance of the plugin adapter
func New() pluginapi.Plugin {
	return &PluginAdapter{
		consumers:  make([]pluginapi.Consumer, 0),
		processors: make([]pluginapi.Processor, 0),
	}
}

// Name returns the plugin's name following the naming convention
func (p *PluginAdapter) Name() string {
	return "flow/processor/latest-ledger"
}

// Version returns the plugin's version
func (p *PluginAdapter) Version() string {
	return "1.0.0"
}

// Type returns the plugin type
func (p *PluginAdapter) Type() pluginapi.PluginType {
	return pluginapi.ProcessorPlugin
}

// Initialize configures the processor using the provided config map
func (p *PluginAdapter) Initialize(config map[string]interface{}) error {
	networkPassphrase, ok := config["network_passphrase"].(string)
	if !ok {
		return fmt.Errorf("missing network_passphrase in config")
	}

	p.networkPassphrase = networkPassphrase

	// Create the core processor
	p.processor = core.NewProcessor(core.Config{
		NetworkPassphrase: networkPassphrase,
	})

	return nil
}

// Process implements the core logic
func (p *PluginAdapter) Process(ctx context.Context, msg pluginapi.Message) error {
	log.Printf("LatestLedgerProcessor: Processing message with %d consumers and %d processors",
		len(p.consumers), len(p.processors))

	// Convert pluginapi message to core format
	ledgerCloseMeta, ok := msg.Payload.(xdr.LedgerCloseMeta)
	if !ok {
		return fmt.Errorf("expected xdr.LedgerCloseMeta, got %T", msg.Payload)
	}

	// Process using core processor
	metrics, err := p.processor.ProcessLedger(ledgerCloseMeta)
	if err != nil {
		return err
	}

	// Convert results to JSON
	jsonBytes, err := metrics.ToJSON()
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

// RegisterConsumer registers a downstream consumer
func (p *PluginAdapter) RegisterConsumer(consumer pluginapi.Consumer) {
	log.Printf("LatestLedgerProcessor: Registering consumer %s", consumer.Name())
	p.consumers = append(p.consumers, consumer)
}

// Subscribe registers a downstream processor
func (p *PluginAdapter) Subscribe(proc pluginapi.Processor) {
	log.Printf("LatestLedgerProcessor: Registering processor %s", proc.Name())
	p.processors = append(p.processors, proc)
}

// GetSchemaDefinition returns GraphQL type definitions for this plugin
func (p *PluginAdapter) GetSchemaDefinition() string {
	return p.processor.GetSchemaDefinition()
}

// GetQueryDefinitions returns GraphQL query definitions for this plugin
func (p *PluginAdapter) GetQueryDefinitions() string {
	return p.processor.GetQueryDefinitions()
}
