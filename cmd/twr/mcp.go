package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
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

// readInput is one read-only league command. Scope is free-form here so the tool does not demand a
// league: twr fills in the saved league itself. read.go still decodes it strictly.
type readInput struct {
	Command string         `json:"command" jsonschema:"a read-only league command whose name ends in .read, such as standings.read"`
	Scope   map[string]any `json:"scope,omitempty" jsonschema:"optional filters for the read, such as season, week or franchise"`
}

func (e env) mcpCommand() *cobra.Command {
	return &cobra.Command{Use: "mcp", Short: "Serve the saved connection to an AI helper over MCP (stdio)", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := e.mcpServer().Run(cmd.Context(), &mcp.StdioTransport{}); err != nil {
				return fmt.Errorf("AI helper connection ended: %w", err)
			}
			return nil
		}}
}

// mcpServer offers the same things the CLI does. Every call reads the saved connection
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
	mcp.AddTool(server, &mcp.Tool{Name: "league_read", Description: "Read one read-only league answer, such as standings.read. It never changes the league."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in readInput) (*mcp.CallToolResult, any, error) {
			scope, err := scopeFrom(in.Scope)
			if err != nil {
				return failed(err), nil, nil
			}
			reply, err := e.read(ctx, in.Command, scope)
			if err != nil {
				return failed(err), nil, nil
			}
			return replied(reply)
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

// scopeFrom turns the tool's scope object into the same strict scope the CLI reads from a file.
func scopeFrom(scope map[string]any) (serverhttp.RawWorkflowScope, error) {
	if scope == nil {
		return serverhttp.RawWorkflowScope{}, nil
	}
	data, err := json.Marshal(scope)
	if err != nil {
		return serverhttp.RawWorkflowScope{}, fmt.Errorf("encode the scope: %w", err)
	}
	return decodeScope(data)
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
