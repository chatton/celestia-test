package types

import "context"

type Header struct {
	Height uint64 `json:"height"`
}

type Blob struct {
	Namespace    string `json:"namespace"`
	Data         string `json:"data"`
	ShareVersion int    `json:"share_version"`
	Commitment   string `json:"commitment"`
	Index        int    `json:"index"`
}

type Node interface {
	// Start starts the node.
	Start(ctx context.Context, coreIp, genesisBlockHash string) error
	// Stop stops the node.
	Stop(ctx context.Context) error
	// GetType returns the type of node. E.g. "bridge" / "light"
	GetType() string
	// GetHeader returns a header at a specified height.
	GetHeader(ctx context.Context, height uint64) (Header, error)
	// GetRPCAddress returns an RPC address resolvable by the test runner.
	GetRPCAddress() string
	GetAllBlobs(ctx context.Context, height uint64) ([]Blob, error)
}
