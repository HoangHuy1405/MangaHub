package cliclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// HTTPClient wraps http.Client with auth and base URL awareness.
type HTTPClient struct {
	cfg    *CLIConfig
	client *http.Client
}

// NewHTTPClient creates an HTTPClient backed by the provided CLIConfig.
func NewHTTPClient(cfg *CLIConfig) *HTTPClient {
	return &HTTPClient{
		cfg:    cfg,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// APIResponse mirrors the standard APIResponse envelope used by the server.
type APIResponse struct {
	Success bool            `json:"success"`
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

// Get performs a GET request and returns the parsed APIResponse.
func (h *HTTPClient) Get(path string) (*APIResponse, error) {
	req, err := http.NewRequest("GET", h.cfg.HTTPURL()+path, nil)
	if err != nil {
		return nil, err
	}
	if h.cfg.Token != "" {
		req.Header.Set("Authorization", "Bearer "+h.cfg.Token)
	}
	return h.do(req)
}

// Post performs a POST request with a JSON body.
func (h *HTTPClient) Post(path string, body any) (*APIResponse, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", h.cfg.HTTPURL()+path, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if h.cfg.Token != "" {
		req.Header.Set("Authorization", "Bearer "+h.cfg.Token)
	}
	return h.do(req)
}

// Put performs a PUT request with a JSON body.
func (h *HTTPClient) Put(path string, body any) (*APIResponse, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("PUT", h.cfg.HTTPURL()+path, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if h.cfg.Token != "" {
		req.Header.Set("Authorization", "Bearer "+h.cfg.Token)
	}
	return h.do(req)
}

// Delete performs a DELETE request.
func (h *HTTPClient) Delete(path string) (*APIResponse, error) {
	req, err := http.NewRequest("DELETE", h.cfg.HTTPURL()+path, nil)
	if err != nil {
		return nil, err
	}
	if h.cfg.Token != "" {
		req.Header.Set("Authorization", "Bearer "+h.cfg.Token)
	}
	return h.do(req)
}

func (h *HTTPClient) do(req *http.Request) (*APIResponse, error) {
	resp, err := h.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("server unreachable at %s — is the server running?", h.cfg.HTTPURL())
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var ar APIResponse
	if err := json.Unmarshal(raw, &ar); err != nil {
		return nil, fmt.Errorf("unexpected response from server: %s", string(raw))
	}
	return &ar, nil
}

// Decode unmarshals the Data field of an APIResponse into dest.
func Decode(ar *APIResponse, dest any) error {
	return json.Unmarshal(ar.Data, dest)
}
