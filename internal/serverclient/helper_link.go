package serverclient

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

// A helper link is the one line a commissioner copies from The War Room into twr connect: the
// token's public audit ID and its secret. A confirm request is the one line the helper hands back
// when a bigger change needs that person's say-so. Both are plain text a person can paste.
const (
	helperLinkPrefix     = "twr1."
	confirmRequestPrefix = "twrc1."
)

func HelperLink(id, bearer string) string { return helperLinkPrefix + id + "." + bearer }

func ParseHelperLink(text string) (id, bearer string, err error) {
	parts := strings.Split(strings.TrimPrefix(strings.TrimSpace(text), helperLinkPrefix), ".")
	if !strings.HasPrefix(strings.TrimSpace(text), helperLinkPrefix) || len(parts) != 2 ||
		(accounts.RawSession{Bearer: parts[0]}).Validate() != nil || (accounts.RawSession{Bearer: parts[1]}).Validate() != nil {
		return "", "", fmt.Errorf("paste the whole connection line from The War Room; it starts with twr1")
	}
	return parts[0], parts[1], nil
}

type confirmRequest struct {
	TokenID string                  `json:"tokenId"`
	Change  commissioner.RawCommand `json:"change"`
}

func EncodeConfirmRequest(id string, raw commissioner.RawCommand) (string, error) {
	data, err := json.Marshal(confirmRequest{TokenID: id, Change: raw})
	if err != nil {
		return "", fmt.Errorf("encode the confirm request: %w", err)
	}
	return confirmRequestPrefix + base64.RawURLEncoding.EncodeToString(data), nil
}

func DecodeConfirmRequest(text string) (string, commissioner.RawCommand, error) {
	refused := fmt.Errorf("paste the whole confirm request from your AI helper; it starts with twrc1")
	encoded, ok := strings.CutPrefix(strings.TrimSpace(text), confirmRequestPrefix)
	if !ok || len(encoded) > 48<<10 {
		return "", commissioner.RawCommand{}, refused
	}
	data, err := base64.RawURLEncoding.Strict().DecodeString(encoded)
	if err != nil {
		return "", commissioner.RawCommand{}, refused
	}
	var request confirmRequest
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&request) != nil || (accounts.RawSession{Bearer: request.TokenID}).Validate() != nil ||
		request.Change.ValidatePreview() != nil {
		return "", commissioner.RawCommand{}, refused
	}
	return request.TokenID, request.Change, nil
}
