package commissioner

import (
	"context"
	"fmt"
)

// ReadingViewKey names what a reading made under ctx may see, so two identical readings share one
// computation only when their viewers would be shown the same thing. Shareable is false for a hosted
// request carrying credential admission or a token audit: those readings always run on their own.
func ReadingViewKey(ctx context.Context) (key string, shareable bool) {
	if _, hosted := ctx.Value(admissionKey{}).(admissionContext); hosted {
		return "", false
	}
	if id, _ := ctx.Value(tokenAuditKey{}).(string); id != "" {
		return "", false
	}
	key = "local"
	if view, ok := ctx.Value(recipientKey{}).(recipientView); ok {
		key = fmt.Sprintf("recipient:%q@%d", view.principal, view.at.UnixNano())
	}
	if principals := ctx.Value(principalsKey{}); principals != nil {
		// Identity maps are compared by pointer: two maps are one view only if they are the same map.
		key += fmt.Sprintf("|principals:%p", principals)
	}
	return key, true
}
