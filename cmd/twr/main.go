// Command twr lets a person or an AI helper work a league from a terminal. It uses a connection
// token a signed-in commissioner made in The War Room, over the same HTTPS dispatch as the
// desktop. Changes preview by default; bigger changes still need that person to confirm them.
//
//	twr connect --address https://league.example --league native.example   (paste the twr1 line)
//	twr preview change.json      twr execute change.json      twr mcp
package main

import (
	"bufio"
	"context"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/serverclient"
	"github.com/spf13/cobra"
)

// connection is what twr remembers between runs. The token itself lives only in the keychain.
type connection struct {
	Address string `json:"address"`
	League  string `json:"league"`
	Helper  string `json:"helper"` // the token's public audit ID, named in confirm requests
}

// env is everything twr touches outside itself, so tests can point it at a throwaway setup.
type env struct {
	configDir string
	roots     *x509.CertPool
	keychain  func(origin, league string) tokenStore
	in        io.Reader
	out       io.Writer
}

type tokenStore interface {
	Get() (string, error)
	Set(secret string) error
	Delete() error
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	dir, err := os.UserConfigDir()
	if err == nil {
		e := env{configDir: filepath.Join(dir, "TheWarRoom"), in: os.Stdin, out: os.Stdout,
			keychain: func(origin, league string) tokenStore { return serverclient.NewAgentKeychain(origin, league) }}
		err = e.root().ExecuteContext(ctx)
	}
	stop()
	if err != nil {
		fmt.Fprintln(os.Stderr, "twr:", err)
		os.Exit(1)
	}
}

func (e env) root() *cobra.Command {
	root := &cobra.Command{Use: "twr", Short: "Work a War Room league from a terminal or an AI helper",
		SilenceUsage: true, SilenceErrors: true}
	root.SetIn(e.in)
	root.SetOut(e.out)
	root.AddCommand(e.connectCommand(), e.ledgerCommand(), e.changeCommand("preview"), e.changeCommand("execute"),
		e.disconnectCommand(), e.mcpCommand(), e.transferCommand())
	return root
}

func (e env) connectCommand() *cobra.Command {
	var c connection
	command := &cobra.Command{Use: "connect", Short: "Save a connection token from The War Room", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			fmt.Fprintln(e.out, "Paste the connection line from The War Room, then press Enter:")
			line, err := bufio.NewReader(e.in).ReadString('\n')
			if err != nil && !errors.Is(err, io.EOF) {
				return fmt.Errorf("read the connection line: %w", err)
			}
			id, token, err := serverclient.ParseHelperLink(line)
			if err != nil {
				return err //nolint:wrapcheck // already a plain message for the person
			}
			c.Helper = id
			agent, err := serverclient.NewAgent(c.Address, c.League, token, e.roots)
			if err != nil {
				return err //nolint:wrapcheck // already a plain message for the person
			}
			if _, err := agent.Ledger(cmd.Context()); err != nil {
				return err //nolint:wrapcheck // already a plain message for the person
			}
			if err := e.keychain(c.Address, c.League).Set(token); err != nil {
				return err //nolint:wrapcheck // already a plain message for the person
			}
			if err := e.save(c); err != nil {
				return err
			}
			fmt.Fprintf(e.out, "Connected to %s.\n", c.League)
			return nil
		}}
	command.Flags().StringVar(&c.Address, "address", "", "the league's https:// address")
	command.Flags().StringVar(&c.League, "league", "", "the league ID (native.…)")
	return command
}

func (e env) ledgerCommand() *cobra.Command {
	return &cobra.Command{Use: "ledger", Short: "Print the league's recorded changes", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			agent, _, err := e.agent()
			if err != nil {
				return err
			}
			ledger, err := agent.Ledger(cmd.Context())
			if err != nil {
				return err //nolint:wrapcheck // already a plain message for the person
			}
			return e.print(ledger)
		}}
}

// changeCommand reads one change as JSON from a file, or from stdin when the file is "-".
func (e env) changeCommand(mode string) *cobra.Command {
	var confirmation string
	short := "Show what a change would do, without recording it"
	if mode == "execute" {
		short = "Record a change; bigger changes need the commissioner's confirmation code"
	}
	command := &cobra.Command{Use: mode + " CHANGE.json", Short: short, Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			raw, err := e.readChange(args[0])
			if err != nil {
				return err
			}
			reply, err := e.change(cmd.Context(), raw, mode, confirmation)
			if err != nil {
				return err
			}
			return e.print(reply)
		}}
	if mode == "execute" {
		command.Flags().StringVar(&confirmation, "confirmation", "", "the code the commissioner confirmed in The War Room")
	}
	return command
}

func (e env) disconnectCommand() *cobra.Command {
	return &cobra.Command{Use: "disconnect", Short: "Forget the saved connection on this computer", Args: cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			c, err := e.load()
			if err != nil {
				return err
			}
			if err := e.keychain(c.Address, c.League).Delete(); err != nil {
				return err //nolint:wrapcheck // already a plain message for the person
			}
			if err := os.Remove(e.configPath()); err != nil && !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("remove the saved connection: %w", err)
			}
			fmt.Fprintln(e.out, "Disconnected. Ask the commissioner to revoke the token in The War Room if it is no longer needed.")
			return nil
		}}
}

func (e env) agent() (*serverclient.Agent, connection, error) {
	c, err := e.load()
	if err != nil {
		return nil, c, err
	}
	token, err := e.keychain(c.Address, c.League).Get()
	if err != nil {
		return nil, c, err //nolint:wrapcheck // already a plain message for the person
	}
	agent, err := serverclient.NewAgent(c.Address, c.League, token, e.roots)
	return agent, c, err //nolint:wrapcheck // plain message
}

// changeReply is a preview or receipt. A preview of a bigger change also carries the one line the
// commissioner pastes into The War Room to confirm it.
type changeReply struct {
	Reply          json.RawMessage `json:"reply"`
	ConfirmRequest string          `json:"confirmRequest,omitempty"`
	Next           string          `json:"next,omitempty"`
}

func (e env) change(ctx context.Context, raw commissioner.RawCommand, mode, confirmation string) (changeReply, error) {
	agent, c, err := e.agent()
	if err != nil {
		return changeReply{}, err
	}
	raw, err = agent.Prepare(ctx, raw)
	if err != nil {
		return changeReply{}, err //nolint:wrapcheck // already a plain message for the person
	}
	reply, err := agent.Dispatch(ctx, raw, mode, confirmation)
	if err != nil {
		return changeReply{}, err //nolint:wrapcheck // already a plain message for the person
	}
	out := changeReply{Reply: reply}
	definition, err := raw.EffectiveDefinition()
	if mode == "preview" && err == nil && (definition.Gravity == "G2" || definition.Gravity == "G3") {
		if out.ConfirmRequest, err = serverclient.EncodeConfirmRequest(c.Helper, raw); err != nil {
			return out, err //nolint:wrapcheck // plain message
		}
		out.Next = "This change needs the commissioner's confirmation. Ask them to paste confirmRequest into The War Room under AI helpers, then run execute with the code they give you."
	}
	return out, nil
}

func (e env) readChange(name string) (commissioner.RawCommand, error) {
	var raw commissioner.RawCommand
	var data []byte
	var err error
	if name == "-" {
		data, err = io.ReadAll(io.LimitReader(e.in, 1<<20))
	} else {
		data, err = os.ReadFile(filepath.Clean(name))
	}
	if err != nil {
		return raw, fmt.Errorf("read the change: %w", err)
	}
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&raw); err != nil {
		return raw, fmt.Errorf("the change is not valid JSON for a league change: %w", err)
	}
	return raw, nil
}

func (e env) configPath() string { return filepath.Join(e.configDir, "twr.json") }

func (e env) load() (connection, error) {
	var c connection
	data, err := os.ReadFile(e.configPath())
	if err != nil {
		return c, fmt.Errorf("no saved connection; run twr connect first")
	}
	if err := json.Unmarshal(data, &c); err != nil {
		return c, fmt.Errorf("the saved connection is unreadable; run twr connect again")
	}
	return c, nil
}

func (e env) save(c connection) error {
	if err := os.MkdirAll(e.configDir, 0o700); err != nil {
		return fmt.Errorf("create the settings folder: %w", err)
	}
	data, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("encode the connection: %w", err)
	}
	if err := os.WriteFile(e.configPath(), data, 0o600); err != nil {
		return fmt.Errorf("save the connection: %w", err)
	}
	return nil
}

func (e env) print(value any) error {
	encoder := json.NewEncoder(e.out)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		return fmt.Errorf("print the reply: %w", err)
	}
	return nil
}
