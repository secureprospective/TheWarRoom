package leaguetransfer

import (
	"context"
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// Move is a local commissioner operation over two explicitly selected databases. Filesystem
// custody is required; do not expose it as a remote endpoint accepting a claimed principal.
// Reverse uses the same operation, with the current authority as source, never a stale copy.
func Move(ctx context.Context, source, target *history.Store, raw RawExport, action, principal string, confirmed bool) error {
	if !confirmed {
		return fmt.Errorf("the commissioner must explicitly confirm this league move action")
	}
	if action != "seal" && action != "activate" && action != "abort" {
		return fmt.Errorf("choose seal, activate or abort")
	}
	if err := raw.Validate(ctx); err != nil {
		return err
	}
	if err := authorizeMove(ctx, source, raw, principal); err != nil {
		return err
	}
	f, err := moveFence(ctx, source, target, raw)
	if err != nil {
		return err
	}
	if action == "seal" {
		if err := target.CheckTransferDestination(ctx, f); err != nil {
			return fmt.Errorf("prepare destination before sealing: %w", err)
		}
		_, err := source.SealTransfer(ctx, f, raw.Events, raw.Checkpoint)
		if err != nil {
			return fmt.Errorf("seal source: %w", err)
		}
		return nil
	}
	return decideMove(ctx, source, target, raw, action, f)
}
func decideMove(ctx context.Context, source, target *history.Store, raw RawExport, action string, f history.TransferFence) error {
	held, err := source.TransferFence(ctx, raw.League)
	if err != nil {
		return fmt.Errorf("read sealed move: %w", err)
	}
	if held.League != f.League || held.Epoch != f.Epoch || held.Content != f.Content || held.Peer != f.Peer {
		return fmt.Errorf("the selected file and destination do not match the sealed move")
	}
	activate := action == "activate"
	if err := source.DecideTransfer(ctx, held, activate, func() error {
		if err := target.DecideDestination(ctx, held, activate, raw.Events, raw.Blobs); err != nil {
			return fmt.Errorf("decide destination: %w", err)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("decide league move: %w", err)
	}
	return nil
}

func authorizeMove(ctx context.Context, source *history.Store, raw RawExport, principal string) error {
	if err := source.CheckTransferCommissioner(ctx, raw.League, principal); err != nil {
		return fmt.Errorf("check move commissioner: %w", err)
	}
	return nil
}

func moveFence(ctx context.Context, source, target *history.Store, raw RawExport) (history.TransferFence, error) {
	peer, err := target.TransferIdentity(ctx)
	if err != nil {
		return history.TransferFence{}, fmt.Errorf("identify destination: %w", err)
	}
	own, err := source.TransferIdentity(ctx)
	if err != nil {
		return history.TransferFence{}, fmt.Errorf("identify source: %w", err)
	}
	if peer == own {
		return history.TransferFence{}, fmt.Errorf("choose a different destination database")
	}
	return history.TransferFence{League: raw.League, Epoch: raw.SourceEpoch, Content: raw.Content, Peer: peer}, nil
}
