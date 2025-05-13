package docker

import (
	"context"
	"fmt"
	"github.com/chatton/celestia-test/framework/docker/consts"
	"github.com/chatton/celestia-test/framework/testutil/toml"
	"github.com/chatton/celestia-test/framework/types"
	volumetypes "github.com/docker/docker/api/types/volume"
	"github.com/docker/go-connections/nat"
	"go.uber.org/zap"
)

var _ types.DANode = &DANode{}

const (
	bridgeNodeRPCPort = "26658/tcp"
	bridgeNodeP2PPort = "2121/tcp"
)

// bridgeNodePorts defines the default port mappings for the DANode's RPC and P2P communication.
var bridgeNodePorts = nat.PortMap{
	nat.Port(bridgeNodeRPCPort): {},
	nat.Port(bridgeNodeP2PPort): {},
}

// newDANode initializes and returns a new DANode instance using the provided context, test name, and configuration.
func newDANode(ctx context.Context, testName string, cfg Config, nodeType types.DANodeType) (types.DANode, error) {
	if cfg.DANodeConfig == nil {
		return nil, fmt.Errorf("bridge node config is nil")
	}

	image := cfg.DANodeConfig.Images[0]

	bn := &DANode{
		nodeType: nodeType,
		cfg:      *cfg.DANodeConfig,
		log: cfg.Logger.With(
			zap.String("node_type", nodeType.String()),
		),
		node: newNode(cfg.DockerNetworkID, cfg.DockerClient, testName, image, "/home/celestia", "bridge"),
	}

	bn.containerLifecycle = NewContainerLifecycle(cfg.Logger, cfg.DockerClient, bn.Name())

	v, err := cfg.DockerClient.VolumeCreate(ctx, volumetypes.CreateOptions{
		Labels: map[string]string{
			consts.CleanupLabel:   testName,
			consts.NodeOwnerLabel: bn.Name(),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("creating volume for chain node: %w", err)
	}
	bn.VolumeName = v.Name

	if err := SetVolumeOwner(ctx, VolumeOwnerOptions{
		Log:        bn.log,
		Client:     cfg.DockerClient,
		VolumeName: v.Name,
		ImageRef:   image.Ref(),
		TestName:   testName,
		UidGid:     image.UIDGID,
	}); err != nil {
		return nil, fmt.Errorf("set volume owner: %w", err)
	}

	return bn, nil
}

// DANode is a docker implementation of a celestia bridge node.
type DANode struct {
	*node
	nodeType types.DANodeType
	cfg      DANodeConfig
	log      *zap.Logger
	testName string
	// ports that are resolvable from the test runners themselves.
	hostRPCPort string
	hostP2PPort string
}

func (n *DANode) GetType() types.DANodeType {
	return n.nodeType
}

// GetHostRPCAddress returns the externally resolvable RPC address of the bridge node.
func (n *DANode) GetHostRPCAddress() string {
	return n.hostRPCPort
}

// Stop terminates the DANode by removing its associated container gracefully using the provided context.
func (n *DANode) Stop(ctx context.Context) error {
	return n.RemoveContainer(ctx)
}

// Start initializes and starts the DANode with the provided core IP and genesis hash in the given context.
// It returns an error if the node initialization or startup fails.
func (n *DANode) Start(ctx context.Context, coreIp string, genesisHash string) error {
	if err := n.initNode(ctx); err != nil {
		return fmt.Errorf("failed to initialize p2p network: %w", err)
	}

	if err := n.startBridgeNode(ctx, coreIp, genesisHash); err != nil {
		return fmt.Errorf("failed to start bridge node: %w", err)
	}

	return nil
}

// disableRPCAuth disables RPC authentication so that the tests can use the endpoints without configuring auth.
func (n *DANode) disableRPCAuth(ctx context.Context) error {
	modifications := make(toml.Toml)
	rpc := make(toml.Toml)
	rpc["SkipAuth"] = true
	modifications["RPC"] = rpc

	return ModifyConfigFile(
		ctx,
		n.log,
		n.DockerClient,
		n.TestName,
		n.VolumeName,
		"config.toml",
		modifications,
	)
}

func (n *DANode) startBridgeNode(ctx context.Context, coreIp, genesisHash string) error {
	env := []string{
		fmt.Sprintf("P2P_NETWORK=%s", n.cfg.ChainID),
		fmt.Sprintf("CELESTIA_CUSTOM=%s:%s", n.cfg.ChainID, genesisHash),
	}

	if err := n.CreateNodeContainer(ctx, coreIp, env); err != nil {
		return fmt.Errorf("failed to create container: %w", err)
	}

	// TODO: eventually re-enable and test with auth.
	if err := n.disableRPCAuth(ctx); err != nil {
		return fmt.Errorf("failed to disable RPC auth: %w", err)
	}

	if err := n.containerLifecycle.StartContainer(ctx); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	// Set the host ports once since they will not change after the container has started.
	hostPorts, err := n.containerLifecycle.GetHostPorts(ctx, bridgeNodeRPCPort, bridgeNodeP2PPort)
	if err != nil {
		return err
	}

	n.hostRPCPort, n.hostP2PPort = hostPorts[0], hostPorts[1]
	return nil
}

func (n *DANode) initNode(ctx context.Context) error {
	// note: my_celes_key is the default key name for the bridge node.
	cmd := []string{"celestia", n.nodeType.String(), "init", "--p2p.network", n.cfg.ChainID, "--keyring.keyname", "my_celes_key", "--node.store", n.homeDir}
	_, _, err := n.exec(ctx, n.log, cmd, nil)
	return err
}

func (n *DANode) CreateNodeContainer(ctx context.Context, coreIp string, env []string) error {
	cmd := []string{"celestia", n.nodeType.String(), "start", "--p2p.network", n.cfg.ChainID, "--core.ip", coreIp, "--rpc.addr", "0.0.0.0", "--rpc.port", "26658", "--keyring.keyname", "my_celes_key", "--node.store", n.homeDir}

	usingPorts := nat.PortMap{}
	for k, v := range bridgeNodePorts {
		usingPorts[k] = v
	}

	return n.containerLifecycle.CreateContainer(ctx, n.TestName, n.NetworkID, n.Image, usingPorts, "", n.bind(), nil, n.HostName(), cmd, env, []string{})
}
