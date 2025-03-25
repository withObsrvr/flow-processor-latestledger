# Latest Ledger Processor for Obsrvr Flow (Dual-Target)

This processor plugin analyzes Stellar ledger data and provides metrics for each closed ledger, including:
- Basic ledger information (sequence, hash, close time)
- Transaction counts and success rates
- Operation counts (both total submitted and successful)
- Transactions per second (calculated from successful operations)
- Fee metrics
- Soroban transaction metrics

## Dual-Target Architecture

This project supports building two different types of modules from the same codebase:

1. **Go Plugin** (.so) - For direct integration with the Flow application using Go's native plugin system
2. **WebAssembly Module** (.wasm) - For sandboxed execution using the WebAssembly System Interface (WASI)

The architecture is designed with:
- Shared core business logic in the `core` package
- Target-specific interfaces in the `plugin` and `wasm` packages
- Build-time selection using Go build tags

## Building with Nix

This project uses Nix for reproducible builds of both targets.

### Prerequisites

- [Nix package manager](https://nixos.org/download.html) with flakes enabled

### Building the Go Plugin (Default)

```bash
git clone <repository-url>
cd flow-processor-latestledger
nix build
```

The built plugin will be available at `./result/lib/flow-latest-ledger.so`.

### Building the WebAssembly Module

```bash
nix build .#wasm
```

The built WebAssembly module will be available at `./result/lib/flow-latest-ledger.wasm`.

## Development Environments

### Default Go Plugin Development

```bash
nix develop
```

This provides a shell with:
- Go 1.24.1
- All development tools
- CGO enabled for plugin development

### WebAssembly Development

```bash
nix develop .#wasm
```

This provides a shell with:
- Go 1.24.1
- Wasmtime for testing WebAssembly modules
- Development tools

## Plugin Configuration

When configuring either module type, you need to provide the network passphrase:

### Go Plugin Configuration (in Flow):

```json
{
  "plugins": [
    {
      "path": "/path/to/flow-latest-ledger.so",
      "config": {
        "network_passphrase": "Public Global Stellar Network ; September 2015"
      }
    }
  ]
}
```

### WebAssembly Module Configuration (example using Wazero):

```go
// Initialize the processor
configJSON := `{"network_passphrase":"Public Global Stellar Network ; September 2015"}`
initFn := wasmModule.ExportedFunction("initialize")
initFn.Call(ctx, api.EncodeString(configJSON))

// Process a ledger
processFn := wasmModule.ExportedFunction("processLedger")
result, _ := processFn.Call(ctx, api.EncodeString(ledgerDataJSON))
metricsJSON := api.DecodeString(result[0])
```

## API Reference

### Go Plugin API
- Implements the `pluginapi.Plugin` interface
- Directly integrates with the Flow plugin system
- Communicates using Go objects

### WebAssembly Exports
The WebAssembly module exports these functions:

| Function | Parameters | Returns | Description |
|----------|------------|---------|-------------|
| `initialize` | `configJSON: string` | `statusCode: int32` | Initializes the processor |
| `processLedger` | `ledgerJSON: string` | `metricsJSON: string` | Processes a ledger and returns metrics |
| `getSchemaDefinition` | none | `schema: string` | Returns GraphQL schema |
| `getQueryDefinitions` | none | `queries: string` | Returns GraphQL queries |
| `version` | none | `version: string` | Returns the module version |
| `name` | none | `name: string` | Returns the module name |

## GraphQL Schema

Both module types provide the same GraphQL schema:

```graphql
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
```

## Understanding the Metrics

This processor tracks several important metrics:

- **txSetOperationCount**: Total number of operations in all transactions submitted to the ledger (successful and failed)
- **successfulOperationCount**: Number of operations from successful transactions only
- **transactionsPerSecond**: The rate of successful operations per second (this is equivalent to what other blockchains call "transactions per second")

In Stellar, a transaction can contain multiple operations, and each operation is an atomic unit of work (payment, account creation, etc.). What other blockchains call a "transaction" is more equivalent to a Stellar "operation" in terms of functionality, which is why our TPS metric is based on operations rather than transactions.

## Comparison of Module Types

| Feature | Go Plugin (.so) | WebAssembly Module (.wasm) |
|---------|----------------|-----------------------------|
| **Integration** | Direct memory access | Sandbox isolation |
| **Performance** | Native speed | Near-native speed |
| **Security** | Less isolation | Strong isolation |
| **Portability** | Linux only | Cross-platform |
| **Dependencies** | Needs matching Go version | Self-contained |
| **API Style** | Native Go objects | JSON string exchange |
| **Build Size** | 64MB+ | Smaller (20-30MB) |

## License

[Specify your license here]