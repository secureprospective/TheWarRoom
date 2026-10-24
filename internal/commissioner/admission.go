package commissioner

import (
	"context"
	"fmt"
)

// Access describes the actual operation, not a caller's claimed permission. Raw retains all
// references (including bid/trade IDs); Records is the current transaction snapshot on append.
type Access struct {
	Principal      string
	League         string
	Operation      string
	Mode           string
	Franchise      string
	Player         string
	SubjectCommand string
	DraftPick      *DraftPickInput
	Raw            RawCommand
	Records        []Record
}

// CredentialAdmission is independent of domain legality. It must check the credential's
// current revocation, issuer rights and argument scope on every call, without writing or
// reentering the ledger. Records and Raw must not be mutated or retained.
type CredentialAdmission interface {
	Admit(context.Context, Access) error
}

type admissionKey struct{}
type admissionContext struct{ admission CredentialAdmission }

// WithCredentialAdmission marks a hosted request, even when admission is nil. The trusted
// transport supplies this dependency; it is never decoded from a request body.
func WithCredentialAdmission(ctx context.Context, admission CredentialAdmission) context.Context {
	return context.WithValue(ctx, admissionKey{}, admissionContext{admission: admission})
}

// AdmitCredential leaves local desktop calls alone; a hosted request never falls back to
// local authority. Writers call it again inside the transaction, before retry lookup.
func AdmitCredential(ctx context.Context, access Access) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("credential admission: %w", err)
	}
	held, hosted := ctx.Value(admissionKey{}).(admissionContext)
	if !hosted {
		return nil
	}
	if held.admission == nil || access.Principal == "" {
		return Refused(fmt.Errorf("your session is unavailable"))
	}
	if err := held.admission.Admit(ctx, access); err != nil {
		return fmt.Errorf("credential admission: %w", Refused(err))
	}
	return nil
}
