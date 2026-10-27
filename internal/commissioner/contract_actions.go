package commissioner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

const ContractActionSet = "contract.action.set"

func contractActionFields() map[string]leaguerules.FieldPath {
	return map[string]leaguerules.FieldPath{
		"tag": "contracts.tag", "extension": "contracts.extension",
		"restructure": "contracts.restructure", "buyout": "contracts.buyout",
		"rfa_tender": "contracts.tender", "resign": "contracts.reSign",
		"retire": "contracts.retirement", "death": "contracts.death", "cap_relief": "contracts.capRelief",
	}
}

func (a RawContractAction) field() (leaguerules.FieldPath, error) {
	field, ok := contractActionFields()[a.Action]
	if !ok {
		return "", fmt.Errorf("choose a recognised contract action")
	}
	return field, nil
}

func (a RawArgs) validateContractAction() error {
	other := a
	other.ContractAction = nil
	if a.ContractAction == nil || !reflect.DeepEqual(other, RawArgs{}) {
		return fmt.Errorf("send only the contract action, its on or off choice and its parameters")
	}
	_, _, err := a.ContractAction.ruleValue()
	return err
}

func (a RawContractAction) ruleValue() (leaguerules.FieldPath, json.RawMessage, error) {
	field, err := a.field()
	if err != nil {
		return "", nil, err
	}
	if a.On == nil {
		return "", nil, fmt.Errorf("choose whether this contract action is on or off")
	}
	params, err := actionParams(a.Params)
	if err != nil {
		return "", nil, err
	}
	params["Enabled"] = json.RawMessage("false")
	if *a.On {
		params["Enabled"] = json.RawMessage("true")
	}
	body, err := json.Marshal(params)
	if err != nil {
		return "", nil, fmt.Errorf("read contract action parameters: %w", err)
	}
	if _, err := leaguerules.DecodeSetting(field, body); err != nil {
		return "", nil, fmt.Errorf("contract action parameters: %w", err)
	}
	return field, body, nil
}

func actionParams(body json.RawMessage) (map[string]json.RawMessage, error) {
	decoder := json.NewDecoder(bytes.NewReader(body))
	token, err := decoder.Token()
	if err != nil || token != json.Delim('{') {
		return nil, fmt.Errorf("send the contract action's named parameters")
	}
	params, seen := map[string]json.RawMessage{}, map[string]bool{}
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return nil, fmt.Errorf("read contract parameter name: %w", err)
		}
		name, ok := token.(string)
		if !ok || seen[strings.ToLower(name)] {
			return nil, fmt.Errorf("send each contract parameter once")
		}
		if strings.EqualFold(name, "Enabled") {
			return nil, fmt.Errorf("use the on or off choice, not a second switch in the parameters")
		}
		seen[strings.ToLower(name)] = true
		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return nil, fmt.Errorf("read contract parameter: %w", err)
		}
		params[name] = value
	}
	if _, err := decoder.Token(); err != nil {
		return nil, fmt.Errorf("read contract parameters: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return nil, fmt.Errorf("send one set of contract parameters")
	}
	return params, nil
}

// ruleRecord projects the named door onto the exact rule value the general rules door writes.
// It never changes the stored command, receipt, compensation links or retry identity.
func ruleRecord(r Record) (Record, error) {
	if r.Command.Command != ContractActionSet {
		return r, nil
	}
	if r.Command.Args.ContractAction == nil {
		return Record{}, fmt.Errorf("the recorded contract action has no details")
	}
	field, value, err := r.Command.Args.ContractAction.ruleValue()
	if err != nil {
		return Record{}, err
	}
	r.Command.Command = RulesSet
	r.Command.Args = RawArgs{Field: string(field), Value: value}
	return r, nil
}
