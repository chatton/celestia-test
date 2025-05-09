package docker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/chatton/celestia-test/framework/types"
	"io"
	"net/http"
	"strconv"
)

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (b *BridgeNode) GetHeader(ctx context.Context, height uint64) (types.Header, error) {
	url := fmt.Sprintf("http://%s", b.hostRPCPort)

	type headerResp struct {
		Header struct {
			Height string `json:"height"`
		} `json:"header"`
	}

	resp, err := callRPC[headerResp](ctx, url, "header.GetByHeight", []uint64{height})
	if err != nil {
		return types.Header{}, err
	}

	h, err := strconv.Atoi(resp.Header.Height)
	if err != nil {
		return types.Header{}, fmt.Errorf("failed to parse height: %w", err)
	}

	return types.Header{Height: uint64(h)}, nil
}

func (b *BridgeNode) GetAllBlobs(ctx context.Context, height uint64) ([]types.Blob, error) {
	url := fmt.Sprintf("http://%s", b.hostRPCPort)
	return callRPC[[]types.Blob](ctx, url, "blob.GetAll", []interface{}{height, nil})
}

// callRPC sends an HTTP POST request with an RPC payload to the given URL and decodes the response to the specified type.
// this can be used to interact with the jsonrpc of the bridge node directly instead of importing the client from celestia-node
// to avoid a circular dependency.
func callRPC[T any](ctx context.Context, url, method string, params interface{}) (T, error) {
	var result T

	payload := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  method,
		"params":  params,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return result, fmt.Errorf("failed to marshal JSON payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return result, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return result, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return result, fmt.Errorf("unexpected status code: %s, body: %s", resp.Status, respBody)
	}

	var rpcResp struct {
		Result T         `json:"result"`
		Error  *RPCError `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return result, fmt.Errorf("failed to decode JSON response: %w", err)
	}

	if rpcResp.Error != nil {
		return result, fmt.Errorf("rpc error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	return rpcResp.Result, nil
}
