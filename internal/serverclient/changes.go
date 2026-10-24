package serverclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
)

func (c *Client) FixtureChange(ctx context.Context, command commissioner.RawCommand) (commissioner.Receipt, error) {
	raw := serverhttp.RawFixtureChange{Protocol: 1, Command: command}
	if err := raw.Validate(); err != nil {
		return commissioner.Receipt{}, fmt.Errorf("fixture change input: %w", err)
	}
	body, err := json.Marshal(raw)
	if err != nil {
		return commissioner.Receipt{}, fmt.Errorf("encode fixture change: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+serverhttp.FixtureChangePath, bytes.NewReader(body))
	if err != nil {
		return commissioner.Receipt{}, fmt.Errorf("create fixture change: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := c.http.Do(request)
	if err != nil {
		return commissioner.Receipt{}, fmt.Errorf("request fixture change: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK || response.Header.Get("Content-Type") != "application/json" {
		return commissioner.Receipt{}, fmt.Errorf("fixture change refused (HTTP %d)", response.StatusCode)
	}
	body, err = io.ReadAll(io.LimitReader(response.Body, (256<<10)+1))
	if err != nil {
		return commissioner.Receipt{}, fmt.Errorf("read fixture receipt: %w", err)
	}
	if len(body) > 256<<10 {
		return commissioner.Receipt{}, fmt.Errorf("fixture receipt too large")
	}
	var receipt serverhttp.RawReceiptResponse
	if err := json.Unmarshal(body, &receipt); err != nil {
		return commissioner.Receipt{}, fmt.Errorf("decode fixture receipt: %w", err)
	}
	if err := receipt.Validate(); err != nil {
		return commissioner.Receipt{}, fmt.Errorf("validate fixture receipt: %w", err)
	}
	return receipt.Receipt, nil
}
