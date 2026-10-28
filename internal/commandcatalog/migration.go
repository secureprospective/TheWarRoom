package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/secureprospective/TheWarRoom/internal/leaguemigration"
)

// AssistDoor is the move from MFL's door: the AI assist and the chat drive the move through it.
const AssistDoor Door = "assist"

// MigrationInputs are the move's ports: the service, and who the commissioner is. Nil leaves the move out.
type MigrationInputs struct {
	Service   *leaguemigration.Service
	Principal func() (string, error)
}

type migrationCommand struct {
	id, label, gravity string
	mode               Mode
	run                func(context.Context, *leaguemigration.Service, leaguemigration.RawStep, string) (any, error)
}

func migrationCommands() []migrationCommand {
	return []migrationCommand{
		{leaguemigration.CommandPrepare, "Read an MFL league and say what moving it would land", "G0", Read,
			func(ctx context.Context, s *leaguemigration.Service, r leaguemigration.RawStep, _ string) (any, error) {
				return s.Prepare(ctx, r.MFLLeague)
			}},
		{leaguemigration.CommandImport, "Import an MFL league into the app", "G3", Execute,
			func(ctx context.Context, s *leaguemigration.Service, r leaguemigration.RawStep, who string) (any, error) {
				return s.Import(ctx, r.MFLLeague, r.Reason, who)
			}},
		{leaguemigration.CommandReview, "Compare the imported league with MFL", "G0", Read,
			func(ctx context.Context, s *leaguemigration.Service, r leaguemigration.RawStep, _ string) (any, error) {
				return s.Review(ctx, request(r))
			}},
		{leaguemigration.CommandCutover, "Switch the imported league over from MFL", "G3", Execute,
			func(ctx context.Context, s *leaguemigration.Service, r leaguemigration.RawStep, who string) (any, error) {
				return s.Cutover(ctx, request(r), r.Digest, r.Reason, who)
			}},
		{leaguemigration.CommandInvites, "Prepare the owner invitations", "G0", Read,
			func(ctx context.Context, s *leaguemigration.Service, r leaguemigration.RawStep, _ string) (any, error) {
				return s.Invites(ctx, request(r))
			}},
		{leaguemigration.CommandSendInvites, "Send the owner invitations", "G2", Execute,
			func(ctx context.Context, s *leaguemigration.Service, r leaguemigration.RawStep, who string) (any, error) {
				return s.SendInvites(ctx, request(r), r.Digest, r.Reason, who)
			}},
	}
}

func request(r leaguemigration.RawStep) leaguemigration.Request {
	return leaguemigration.Request{MFLLeague: r.MFLLeague, League: r.League}
}

// MigrationEntries are the move's registry entries, each with its one handler on the assist door.
func MigrationEntries(in MigrationInputs) []Entry {
	out := make([]Entry, 0, len(migrationCommands()))
	for _, c := range migrationCommands() {
		definition := Command{ID: c.id, Method: c.id, Version: "1", Role: "commish", Gravity: c.gravity,
			Availability: "desktop", Permission: "command:" + c.id, Modes: []Mode{c.mode},
			Arguments: Shape(reflect.TypeFor[leaguemigration.RawStep](), "leaguemigration.RawStep.Validate"),
			Undo:      "none; MFL is never changed, so a move given up is imported again",
			Facts:     "mfl-read-only,ledger", Safeguards: "typed reason; digest of the review or list read"}
		out = append(out, Entry{ID: c.id, Label: c.label, Definition: definition,
			Assist: map[Mode]Handler{c.mode: migrationHandler(c, in)}})
	}
	return out
}

func migrationHandler(c migrationCommand, in MigrationInputs) Handler {
	return func(ctx context.Context, body json.RawMessage) (json.RawMessage, error) {
		var raw leaguemigration.RawStep
		if err := decodeJSON(body, &raw); err != nil {
			return nil, err
		}
		if err := raw.Validate(); err != nil {
			return nil, fmt.Errorf("check the move step: %w", err)
		}
		who, err := in.Principal()
		if err != nil {
			return nil, fmt.Errorf("the commissioner is not signed in: %w", err)
		}
		return result(c.run(ctx, in.Service, raw, who))
	}
}
