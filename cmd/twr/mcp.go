package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/spf13/cobra"
)

// changeInput is one league change for an AI helper. twr fills in the league and the current
// revision. Execute needs the person's confirmation code for bigger changes; the helper can never
// confirm on its own.
type changeInput struct {
	Command      string               `json:"command" jsonschema:"the league command, as listed by twr preview"`
	Args         commissioner.RawArgs `json:"args,omitzero" jsonschema:"the command's arguments"`
	Reason       string               `json:"reason,omitempty" jsonschema:"why; bigger changes need one"`
	Request      string               `json:"request,omitempty"`
	Effective    string               `json:"effective,omitempty"`
	Visibility   string               `json:"visibility,omitempty"`
	Confirmation string               `json:"confirmation,omitempty" jsonschema:"code the commissioner confirmed in The War Room"`
}

func (in changeInput) raw() commissioner.RawCommand {
	return commissioner.RawCommand{Command: in.Command, Args: in.Args, Reason: in.Reason, Request: in.Request,
		Effective: in.Effective, Visibility: in.Visibility}
}

type noInput struct{}

func (e env) mcpCommand() *cobra.Command {
	return &cobra.Command{Use: "mcp", Short: "Serve the saved connection to an AI helper over MCP (stdio)", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := e.mcpServer().Run(cmd.Context(), &mcp.StdioTransport{}); err != nil {
				return fmt.Errorf("AI helper connection ended: %w", err)
			}
			return nil
		}}
}

// mcpServer offers the same three things the CLI does. Every call reads the saved connection
// fresh, so twr disconnect or a revoked token stops the helper at once.
func (e env) mcpServer() *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: "twr", Version: "1"}, nil)
	mcp.AddTool(server, &mcp.Tool{Name: "league_ledger", Description: "Read the league's recorded changes."},
		func(ctx context.Context, _ *mcp.CallToolRequest, _ noInput) (*mcp.CallToolResult, any, error) {
			agent, _, err := e.agent()
			if err != nil {
				return failed(err), nil, nil
			}
			ledger, err := agent.Ledger(ctx)
			if err != nil {
				return failed(err), nil, nil
			}
			return replied(ledger)
		})
	for _, mode := range []string{"preview", "execute"} {
		description := "Show what a league change would do, without recording it. Always preview first."
		if mode == "execute" {
			description = "Record a previewed league change. Bigger changes need the commissioner's confirmation code."
		}
		mcp.AddTool(server, &mcp.Tool{Name: mode + "_change", Description: description},
			func(ctx context.Context, _ *mcp.CallToolRequest, in changeInput) (*mcp.CallToolResult, any, error) {
				confirmation := ""
				if mode == "execute" {
					confirmation = in.Confirmation
				}
				reply, err := e.change(ctx, in.raw(), mode, confirmation)
				if err != nil {
					return failed(err), nil, nil
				}
				return replied(reply)
			})
	}
	return server
}

func failed(err error) *mcp.CallToolResult {
	return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}}}
}

func replied(value any) (*mcp.CallToolResult, any, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, nil, fmt.Errorf("encode the reply: %w", err)
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(data)}}}, nil, nil
}
