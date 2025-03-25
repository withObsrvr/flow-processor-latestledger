# Latest Ledger Processor Plugin for Obsrvr Flow

This processor plugin analyzes Stellar ledger data and provides metrics for each closed ledger, including:
- Basic ledger information (sequence, hash, close time)
- Transaction counts and success rates
- Operation counts (both total submitted and successful)
- Transactions per second (calculated from successful operations)
- Fee metrics
- Soroban transaction metrics

## Building with Nix

This project uses Nix for reproducible builds.

### Prerequisites

- [Nix package manager](https://nixos.org/download.html) with flakes enabled

### Building

1. Clone the repository:
```bash
git clone <repository-url>
cd flow-processor-latestledger
```

2. Build with Nix:
```bash
nix build
```

The built plugin will be available at `./result/lib/flow-latest-ledger.so`.

### Development

To enter a development shell with all dependencies:

```bash
nix develop
```

This will provide a shell with all the necessary dependencies, including:
- Go 1.24.1
- Development tools (gopls, delve)

From within the development shell, you can build the plugin manually:
```bash
go build -buildmode=plugin -o flow-latest-ledger.so .
```

## Plugin Configuration

When configuring this plugin, you need to provide the network passphrase in your Flow configuration:

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

## GraphQL Schema

This plugin provides the following GraphQL types and queries:

### Types

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

### Queries

```graphql
latestLedger: LatestLedger
ledgerBySequence(sequence: Int!): LatestLedger
```

## Understanding the Metrics

This plugin tracks several important metrics:

- **txSetOperationCount**: Total number of operations in all transactions submitted to the ledger (successful and failed)
- **successfulOperationCount**: Number of operations from successful transactions only
- **transactionsPerSecond**: The rate of successful operations per second (this is equivalent to what other blockchains call "transactions per second")

In Stellar, a transaction can contain multiple operations, and each operation is an atomic unit of work (payment, account creation, etc.). What other blockchains call a "transaction" is more equivalent to a Stellar "operation" in terms of functionality, which is why our TPS metric is based on operations rather than transactions.

## Dependencies

All dependencies are managed through the `flake.nix` file when using Nix, including:
- github.com/stellar/go
- github.com/withObsrvr/pluginapi

## License

[Specify your license here]