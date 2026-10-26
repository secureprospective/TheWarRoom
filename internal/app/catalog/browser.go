package catalog

import "strings"

// BrowserCommand describes a local browser action. Its typed browser handler, not a remote
// transport, owns the argument values; none of these declarations grants server authority.
type BrowserCommand struct {
	ID      string       `json:"id"`
	Label   string       `json:"label"`
	Aliases []string     `json:"aliases"`
	Roles   []string     `json:"roles"`
	Gravity string       `json:"gravity"`
	Undo    string       `json:"undo"`
	Args    []string     `json:"args"`
	Keys    []BrowserKey `json:"keys,omitempty"`
}

type BrowserKey struct {
	Key  string            `json:"key"`
	Args map[string]string `json:"args"`
}

func BrowserCommands() []BrowserCommand {
	out := append(browserWorkCommands(), browserShellCommands()...)
	for i := range out {
		c := &out[i]
		if c.Roles == nil {
			c.Roles = []string{"gm", "commish", "admin"}
		}
		if c.Aliases == nil {
			c.Aliases = []string{}
		}
		if c.Args == nil {
			c.Args = []string{}
		}
		if c.Gravity == "" {
			c.Gravity = "G0"
		}
		if c.Undo == "" {
			c.Undo = "instant"
		}
	}
	return out
}

func browserWorkCommands() []BrowserCommand {
	return []BrowserCommand{
		{ID: "watch.add", Label: "Watch player", Args: strings.Fields("id entity action text")},
		{ID: "bait.set", Label: "Mark trade bait", Args: strings.Fields("id entity action text")},
		{ID: "note.add", Label: "Keep note", Args: strings.Fields("id entity action text")},
		{ID: "draft.list", Label: "Change draft list", Args: strings.Fields("id entity action text")},
		{ID: "personal.handoff", Label: "Open MFL list", Args: strings.Fields("kind")},
		{ID: "window.set", Label: "Commissioner: set a league date", Args: strings.Fields("kind at"), Gravity: "G2", Undo: "reversible"},
		{ID: "window.edit", Label: "Commissioner: choose a league date", Args: strings.Fields("kind")},
		{ID: "phase.goto", Label: "Open phase workspace", Args: strings.Fields("phase")},
		{ID: "inspector.pin", Label: "Inspect entity", Args: strings.Fields("entity")},
		{ID: "refresh.run", Label: "Refresh data", Args: strings.Fields("scope"), Gravity: "G1"},
		{ID: "cache.run", Label: "Read data again", Args: strings.Fields("scope"), Gravity: "G1"},
		{ID: "data.run", Label: "Data settings", Args: strings.Fields("kind"), Gravity: "G1"},
		{ID: "data.parameter", Label: "Change engine parameter", Args: strings.Fields("key position value"), Roles: []string{"admin"}, Gravity: "G1"},
		{ID: "contracts.set", Label: "Filter or sort contracts", Args: strings.Fields("field value")},
		{ID: "offers.filter", Label: "Filter Offers", Args: strings.Fields("field value")},
		{ID: "market", Label: "Market", Args: strings.Fields("field value")},
		{ID: "trade.builder", Label: "Trade builder", Args: strings.Fields("action"), Gravity: "G3", Undo: "irreversible"},
		{ID: "acquisition", Label: "What does the add cost me?", Args: strings.Fields("mine")},
		{ID: "trade.accept.plan", Label: "Plan accept", Args: strings.Fields("tradeId"), Roles: []string{"gm"}, Gravity: "G2", Undo: "reversible"},
		{ID: "lineup.edit", Label: "Edit lineup", Args: strings.Fields("franchiseId")},
		{ID: "lineup.toggle", Label: "Toggle starter", Args: strings.Fields("playerId")},
		{ID: "lineup.reset", Label: "Reset"},
		{ID: "lineup.cancel", Label: "Cancel"},
		{ID: "lineup.draft", Label: "Check and save plan", Gravity: "G1", Undo: "reversible"},
		{ID: "move.withdraw", Label: "Remove roster plan", Args: strings.Fields("correlationId")},
		{ID: "move.handoff", Label: "Open MFL page", Args: strings.Fields("correlationId"), Gravity: "G1", Undo: "reversible"},
		{ID: "mflkey.connect", Label: "Connect MFL", Aliases: []string{"mfl", "api key"}, Gravity: "G1", Undo: "reversible"},
		{ID: "mflkey.forget", Label: "Forget MFL key", Gravity: "G1", Undo: "reversible"},
		{ID: "roster.ir", Label: "Draft IR move", Args: strings.Fields("subject"), Roles: []string{"gm"}, Gravity: "G2", Undo: "reversible"},
		{ID: "roster.taxi", Label: "Draft move to taxi squad", Args: strings.Fields("subject"), Roles: []string{"gm"}, Gravity: "G2", Undo: "reversible"},
		{ID: "surface.open", Label: "Open shell surface", Args: strings.Fields("place"), Aliases: []string{"surface"}},
		{ID: "surface.close", Label: "Close endpoint surface"},
		{ID: "report.sort", Label: "Sort report", Args: strings.Fields("id column")},
		{ID: "market.reports", Label: "Player market reports", Args: strings.Fields("action")},
		{ID: "comms.read", Label: "League communications", Args: strings.Fields("action value")},
		{ID: "backup.now", Label: "Back up now", Aliases: []string{"backup", "save a backup"}, Gravity: "G1"},
		{ID: "commissioner.surface", Label: "Change the league view", Args: strings.Fields("screen action value")},
		{ID: "commissioner.read", Label: "Read the league ledger", Args: strings.Fields("league")},
		{ID: "commissioner.draft", Label: "Edit a rule value", Args: strings.Fields("action path index")},
		{ID: "commissioner.execute", Label: "Record a league change", Args: strings.Fields("body")},
		{ID: "control.read", Label: "Control Room league and site news", Args: strings.Fields("action value")},
		{ID: "research.read", Label: "Player research", Args: strings.Fields("action value")},
		{ID: "archive.reports", Label: "League archive reports", Args: strings.Fields("action value")},
		{ID: "race.reports", Label: "League race reports", Args: strings.Fields("action value")},
		{ID: "game.follow", Label: "Follow NFL game", Args: strings.Fields("id")},
		{ID: "now.reports", Label: "This week in the league", Args: strings.Fields("action value")},
		{ID: "draft.reports", Label: "Draft reports", Args: strings.Fields("action")},
	}
}

func browserShellCommands() []BrowserCommand {
	return []BrowserCommand{
		{ID: "endpoint.open", Label: "Open endpoint", Args: strings.Fields("id"), Aliases: []string{"endpoint"}},
		{ID: "franchise.set", Label: "View a franchise", Args: strings.Fields("franchiseId"), Aliases: []string{"view team"}},
		{ID: "preset.apply", Label: "Apply preset", Args: strings.Fields("preset"), Aliases: []string{"view"}},
		{ID: "inspector.open", Label: "Inspect player", Args: strings.Fields("subject"), Aliases: []string{"player"}},
		{ID: "inspector.collapse", Label: "Collapse inspector", Aliases: []string{"collapse"}},
		{ID: "commandbar.close", Label: "Close command bar"},
		{ID: "nav.open", Label: "Open workspace", Args: strings.Fields("node workspace"), Aliases: []string{"goto", "navigate"}},
		{ID: "density.set", Label: "Set density", Args: strings.Fields("density"), Aliases: []string{"density"}, Keys: []BrowserKey{{Key: "N", Args: map[string]string{"density": "narrative"}}, {Key: "T", Args: map[string]string{"density": "tactical"}}, {Key: "M", Args: map[string]string{"density": "matrix"}}, {Key: "1", Args: map[string]string{"density": "narrative"}}, {Key: "2", Args: map[string]string{"density": "tactical"}}, {Key: "3", Args: map[string]string{"density": "matrix"}}}},
		{ID: "inspector.expand", Label: "Expand inspector", Aliases: []string{"expand"}},
		{ID: "inspector.toggle", Label: "Toggle inspector", Aliases: []string{"inspect"}, Keys: []BrowserKey{{Key: "I", Args: map[string]string{}}}},
		{ID: "inspector.close", Label: "Close inspector", Aliases: []string{"close inspector"}},
		{ID: "calendar.summon", Label: "Summon calendar", Aliases: []string{"calendar"}},
		{ID: "comms.toggle", Label: "Toggle comms", Aliases: []string{"comms", "chat"}},
		{ID: "overlays.close", Label: "Close overlays", Aliases: []string{"escape"}, Keys: []BrowserKey{{Key: "Escape", Args: map[string]string{}}}},
		{ID: "commandbar.open", Label: "Type a command", Aliases: []string{"command", "/"}, Keys: []BrowserKey{{Key: "/", Args: map[string]string{}}, {Key: "Ctrl+K", Args: map[string]string{}}}},
	}
}
