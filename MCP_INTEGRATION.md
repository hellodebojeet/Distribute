# MCP Integration Guide

This document describes the integration of Modular Capability Providers (MCPs) into the distributed filesystem to evolve it into an IPFS + NFT-compatible system.

## Overview

The following MCPs have been integrated:

1. **P2P Networking** - libp2p
2. **DHT** - Kademlia DHT
3. **Content Addressing** - CID + Multihash
4. **Merkle DAG** - IPLD
5. **Block Exchange** - Bitswap
6. **CLI** - Cobra
7. **Observability** - Zap + Prometheus

## 1. P2P Networking (libp2p)

### Purpose
Provides production-grade peer-to-peer networking with NAT traversal, secure connections, and multiple transport protocols.

### Library Used
`github.com/libp2p/go-libp2p`

### Interface Definition
```go
type Host interface {
    ID() peer.ID
    Addrs() []ma.Multiaddr
    Connect(ctx context.Context, pi peer.AddrInfo) error
    Disconnect(pid peer.ID) error
    SetStreamHandler(pid protocol.ID, handler network.StreamHandler)
    NewStream(ctx context.Context, p peer.ID, pids ...protocol.ID) (network.Stream, error)
    Close() error
    Peerstore() peerstore.Peerstore
    Network() network.Network
}
```

### Integration Points
- `internal/p2p/host.go` - Core P2P node implementation
- `internal/p2p/discovery.go` - Peer discovery mechanisms
- `internal/p2p/streams.go` - Stream management for file transfer

### Code Snippet
```go
// Create a new libp2p host
host, privKey, err := p2p.NewHostWithKey("/ip4/0.0.0.0/tcp/4001")
if err != nil {
    log.Fatal(err)
}
defer host.Close()

// Set stream handler
host.SetStreamHandler("/distribute/file-transfer/1.0.0", func(s network.Stream) {
    // Handle incoming file transfer
})
```

### Concurrency Concerns
- All libp2p operations are goroutine-safe
- Stream handlers must be thread-safe
- Use proper synchronization for shared state

### Failure Scenarios
- Network partitions
- NAT traversal failures
- Peer disconnections
- Invalid message formats

## 2. DHT (Kademlia)

### Purpose
Provides decentralized peer discovery and content routing.

### Library Used
`github.com/libp2p/go-libp2p-kad-dht`

### Interface Definition
```go
type DHT interface {
    Bootstrap(ctx context.Context) error
    PutValue(ctx context.Context, key string, value []byte) error
    GetValue(ctx context.Context, key string) ([]byte, error)
    FindPeer(ctx context.Context, peerID peer.ID) (peer.AddrInfo, error)
    RoutingTable() interface{}
    Close() error
}
```

### Integration Points
- `internal/dht/dht.go` - DHT wrapper implementation
- Integrated with P2P host for peer discovery

### Code Snippet
```go
// Create DHT with the P2P host
dhtConfig := dht.DHTConfig{
    Host: host,
    Mode: dht.ModeAutoServer,
}
d, err := dht.NewDHT(dhtConfig)
if err != nil {
    log.Fatal(err)
}
defer d.Close()

// Bootstrap the DHT
if err := d.Bootstrap(ctx); err != nil {
    log.Fatal(err)
}
```

### Edge Cases
- Bootstrap failures
- DHT pollution attacks
- Routing table inconsistencies

## 3. Content Addressing (CID + Multihash)

### Purpose
Provides content-addressable storage with cryptographic hashing.

### Libraries Used
- `github.com/ipfs/go-cid`
- `github.com/multiformats/go-multihash`

### Interface Definition
```go
type CID interface {
    String() string
    Bytes() []byte
    Hash() mh.Multihash
    Version() uint64
    Codec() uint64
    Equals(other CID) bool
    Validate() error
}
```

### Integration Points
- `internal/cid/cid.go` - CID wrapper implementation
- Used by blockstore and DAG for content addressing

### Code Snippet
```go
// Create a CID from data
config := cid.DefaultCIDConfig()
c, err := cid.NewCID([]byte("hello world"), config)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("CID: %s\n", c.String())
```

### Edge Cases
- Hash collisions
- Invalid CID formats
- Version compatibility

## 4. Merkle DAG (IPLD)

### Purpose
Provides Merkle DAG structure for representing linked data.

### Library Used
`github.com/ipld/go-ipld-prime`

### Interface Definition
```go
type Node interface {
    CID() cid.Cid
    RawData() []byte
    Links() []Link
    AddLink(name string, target cid.Cid) error
    RemoveLink(name string) error
}

type Builder interface {
    BuildNode(data []byte) (Node, error)
    BuildNodeWithLinks(data []byte, links []Link) (Node, error)
    BuildTree(nodes []Node) (Node, error)
}
```

### Integration Points
- `internal/dag/builder.go` - DAG builder implementation
- Used for file and directory structures

### Code Snippet
```go
// Create a DAG builder
builder := dag.NewBuilder(ipld.LinkSystem{})

// Build a node
node, err := builder.BuildNode([]byte("file content"))
if err != nil {
    log.Fatal(err)
}

// Add a link
node.AddLink("child", childCID)
```

### Edge Cases
- Circular references
- Missing linked nodes
- Large DAG traversal

## 5. Block Exchange (Bitswap)

### Purpose
Provides efficient block exchange protocol for distributing data.

### Library Used
`github.com/ipfs/go-bitswap` (simplified implementation)

### Interface Definition
```go
type Bitswap interface {
    GetBlock(ctx context.Context, c cid.Cid) (*blockstore.Block, error)
    HasBlock(c cid.Cid)
    WantBlock(c cid.Cid)
    Close() error
}

type Exchange interface {
    FetchBlock(ctx context.Context, c cid.Cid) (*blockstore.Block, error)
    ProvideBlock(ctx context.Context, b *blockstore.Block) error
    RegisterProvider(c cid.Cid, provider peer.ID)
    GetProviders(c cid.Cid) []peer.ID
}
```

### Integration Points
- `internal/bitswap/bitswap.go` - Bitswap implementation
- Works with blockstore for local storage

### Code Snippet
```go
// Create bitswap
bs, err := bitswap.NewBitswap(bitswap.BitswapConfig{
    Host:       host,
    Blockstore: blockstore,
})
if err != nil {
    log.Fatal(err)
}
defer bs.Close()

// Request a block
block, err := bs.GetBlock(ctx, cid)
if err != nil {
    log.Fatal(err)
}
```

### Edge Cases
- Network timeouts
- Block validation
- Provider discovery

## 6. CLI (Cobra)

### Purpose
Provides a professional command-line interface.

### Library Used
`github.com/spf13/cobra`

### Integration Points
- `cmd/cli/main.go` - CLI implementation

### Code Snippet
```go
var nodeCmd = &cobra.Command{
    Use:   "node",
    Short: "Start a storage node",
    RunE:  runNode,
}

func runNode(cmd *cobra.Command, args []string) error {
    listen, _ := cmd.Flags().GetString("listen")
    // Start node...
    return nil
}
```

## 7. Observability (Zap + Prometheus)

### Purpose
Provides structured logging and metrics collection.

### Libraries Used
- `go.uber.org/zap` - Structured logging
- `prometheus/client_golang` - Metrics

### Interface Definition
```go
type Logger interface {
    Debug(msg string, fields ...Field)
    Info(msg string, fields ...Field)
    Warn(msg string, fields ...Field)
    Error(msg string, fields ...Field)
    Fatal(msg string, fields ...Field)
    With(fields ...Field) Logger
    Sync() error
}

type Metrics interface {
    Counter(name string, labels map[string]string) Counter
    Gauge(name string, labels map[string]string) Gauge
    Histogram(name string, labels map[string]string) Histogram
    StartServer(addr string) error
    StopServer() error
}
```

### Integration Points
- `internal/observability/logging.go` - Logger implementation
- `internal/observability/metrics.go` - Metrics implementation

### Code Snippet
```go
// Create logger
logger, err := observability.NewLogger(observability.LoggerConfig{
    Level:  "info",
    Format: "json",
})
if err != nil {
    log.Fatal(err)
}
defer logger.Sync()

// Log with fields
logger.Info("server started",
    observability.StringField("addr", ":4001"),
    observability.IntField("peers", 10),
)

// Create metrics
metrics := observability.NewMetrics(observability.MetricsConfig{
    Namespace: "distribute",
})

// Start metrics server
metrics.StartServer(":9090")
```

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         CLI (Cobra)                          │
├─────────────────────────────────────────────────────────────┤
│                    Distributed Filesystem                    │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │
│  │   P2P Node  │  │    DHT      │  │  Bitswap    │         │
│  │  (libp2p)   │  │ (Kademlia)  │  │             │         │
│  └─────────────┘  └─────────────┘  └─────────────┘         │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │
│  │    CID      │  │  Blockstore │  │    DAG      │         │
│  │ (Multihash) │  │             │  │   (IPLD)    │         │
│  └─────────────┘  └─────────────┘  └─────────────┘         │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐                          │
│  │   Logger    │  │   Metrics   │                          │
│  │   (Zap)     │  │ (Prometheus)│                          │
│  └─────────────┘  └─────────────┘                          │
└─────────────────────────────────────────────────────────────┘
```

## Usage Example

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/hellodebojeet/Distribute/internal/p2p"
    "github.com/hellodebojeet/Distribute/internal/dht"
    "github.com/hellodebojeet/Distribute/internal/observability"
)

func main() {
    // Create logger
    logger := observability.NewDefaultLogger()
    defer logger.Sync()

    // Create P2P host
    host, _, err := p2p.NewHostWithKey("/ip4/0.0.0.0/tcp/4001")
    if err != nil {
        log.Fatal(err)
    }
    defer host.Close()

    logger.Info("host created",
        observability.StringField("id", host.ID().String()),
    )

    // Create DHT
    dhtConfig := dht.DHTConfig{
        Host: host.(*p2p.Host),
    }
    d, err := dht.NewDHT(dhtConfig)
    if err != nil {
        log.Fatal(err)
    }
    defer d.Close()

    // Bootstrap DHT
    if err := d.Bootstrap(context.Background()); err != nil {
        logger.Error("bootstrap failed", observability.ErrorField(err))
    }

    logger.Info("node started successfully")
}
```

## Configuration

All MCPs support configuration through their respective config structs:

```go
// P2P configuration
p2pConfig := p2p.HostConfig{
    ListenAddrs: []string{"/ip4/0.0.0.0/tcp/4001"},
    NATManager:  true,
    Relay:       true,
}

// DHT configuration
dhtConfig := dht.DHTConfig{
    Mode:        dht.ModeAutoServer,
    BucketSize:  20,
    Concurrency: 3,
}

// Logger configuration
loggerConfig := observability.LoggerConfig{
    Level:      "info",
    Format:     "json",
    OutputPath: "/var/log/distribute.log",
}

// Metrics configuration
metricsConfig := observability.MetricsConfig{
    Namespace: "distribute",
}
```

## Testing

All interfaces are designed to be mockable for testing:

```go
type MockHost struct {
    mock.Mock
}

func (m *MockHost) Connect(ctx context.Context, pi peer.AddrInfo) error {
    args := m.Called(ctx, pi)
    return args.Error(0)
}
```

## Performance Considerations

1. **P2P**: Connection pooling and stream multiplexing
2. **DHT**: Caching and query optimization
3. **Blockstore**: LRU caching for frequently accessed blocks
4. **DAG**: Lazy loading of child nodes
5. **Bitswap**: Batch requests and priority queuing

## Security

1. **Transport Security**: TLS 1.3 for all P2P connections
2. **Content Verification**: CID-based content addressing ensures integrity
3. **Access Control**: Interface-based authorization
4. **Audit Logging**: Structured logging for all operations

## Future Enhancements

1. **NFT Support**: Add ERC-721 metadata support
2. **Smart Contracts**: Integration with blockchain for access control
3. **Advanced DAG**: Support for UnixFS and other DAG formats
4. **Performance**: GPU acceleration for hashing
5. **Monitoring**: Distributed tracing with OpenTelemetry