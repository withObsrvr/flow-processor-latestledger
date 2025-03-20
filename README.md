# Latest Ledger Processor Plugin for Obsrvr Flow

This processor plugin analyzes Stellar ledger data and provides metrics for each closed ledger, including:
- Basic ledger information (sequence, hash, close time)
- Transaction counts and success rates
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
- Go 1.21
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
```

### Queries

```graphql
latestLedger: LatestLedger
ledgerBySequence(sequence: Int!): LatestLedger
```

## Dependencies

All dependencies are managed through the `flake.nix` file when using Nix, including:
- github.com/stellar/go
- github.com/withObsrvr/pluginapi

## License

[Specify your license here]