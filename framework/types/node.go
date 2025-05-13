package types

import "context"

type Header struct {
	Height uint64 `json:"height"`
}

type DANodeType int

const (
	BridgeNode DANodeType = iota
	LightNode
	FullNode
)

func (n DANodeType) String() string {
	return nodeStrings[n]
}

var nodeStrings = [...]string{
	"bridge",
	"light",
	"full",
}

type DANode interface {
	// Start starts the node.
	Start(ctx context.Context, coreIp, genesisBlockHash string) error
	// Stop stops the node.
	Stop(ctx context.Context) error
	// GetType returns the type of node. E.g. "bridge" / "light" / "full"
	GetType() DANodeType
	// GetHeader returns a header at a specified height.
	GetHeader(ctx context.Context, height uint64) (Header, error)
	// GetHostRPCAddress returns the externally resolvable RPC address of the node.
	GetHostRPCAddress() string
}
