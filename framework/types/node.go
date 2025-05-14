package types

import (
	"context"
	"github.com/celestiaorg/go-square/v2/share"
)

type P2PInfo struct {
	PeerID    string   `json:"ID"`
	Addresses []string `json:"Addrs"`
}

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
	Start(ctx context.Context, opts DANodeStartOptions) error
	// Stop stops the node.
	Stop(ctx context.Context) error
	// GetType returns the type of node. E.g. "bridge" / "light" / "full"
	GetType() DANodeType
	// GetHeader returns a header at a specified height.
	GetHeader(ctx context.Context, height uint64) (Header, error)
	GetAllBlobs(ctx context.Context, height uint64, namespaces []share.Namespace) ([]Blob, error)
	// GetHostRPCAddress returns the externally resolvable RPC address of the node.
	GetHostRPCAddress() string
	GetP2PInfo(ctx context.Context) (P2PInfo, error)
}

type Blob struct {
	Namespace    string `json:"namespace"`
	Data         string `json:"data"`
	ShareVersion int    `json:"share_version"`
	Commitment   string `json:"commitment"`
	Index        int    `json:"index"`
}

type DANodeStartOptions struct {
	P2PAddress       string
	GenesisBlockHash string
	CoreIP           string
}
