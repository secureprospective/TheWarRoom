package catalog

import "strings"

// BrowserCommand describes a local browser action. Its typed browser handler, not a remote
// transport, owns the argument values; none of these declarations grants server authority.
type BrowserCommand struct {
	Arguments  Schema       `json:"-"`
	Searchable bool         `json:"-"`
	ID         string       `json:"id"`
	Label      string       `json:"label"`
	Aliases    []string     `json:"aliases"`
	Roles      []string     `json:"roles"`
	Gravity    string       `json:"gravity"`
	Undo       string       `json:"undo"`
	Args       []string     `json:"args"`
	Keys       []BrowserKey `json:"keys,omitempty"`
}

type BrowserKey struct {
	Key  string            `json:"key"`
	Args map[string]string `json:"args"`
}

func BrowserCommands() []BrowserCommand {
	out := append(browserWorkCommands(), browserShellCommands()...)
	for i := range out {
		c := &out[i]
		if c.Arguments.ClientType == "" && len(c.Args) == 0 {
			c.Arguments = browserType("Record<string, never>")
		}
		c.Searchable = c.ID != "commandbar.open" && c.ID != "commandbar.close" && c.ID != "overlays.close" && c.ID != "surface.close"
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
		{ID: "watch.add", Arguments: browserType("import('../personal/contract').ListChange"), Label: "Watch player", Args: strings.Fields("id entity action text")},
		{ID: "bait.set", Arguments: browserType("import('../personal/contract').ListChange"), Label: "Mark trade bait", Args: strings.Fields("id entity action text")},
		{ID: "note.add", Arguments: browserType("import('../personal/contract').ListChange"), Label: "Keep note", Args: strings.Fields("id entity action text")},
		{ID: "draft.list", Arguments: browserType("import('../personal/contract').ListChange"), Label: "Change draft list", Args: strings.Fields("id entity action text")},
		{ID: "personal.handoff", Arguments: browserType("{ kind: import('../personal/contract').ListKind | 'ir' }"), Label: "Open MFL list", Args: strings.Fields("kind")},
		{ID: "window.set", Arguments: browserType("{ kind: import('../data/contract').WindowKind; at: string }"), Label: "Commissioner: set a league date", Args: strings.Fields("kind at"), Gravity: "G2", Undo: "reversible"},
		{ID: "window.edit", Arguments: browserType("{ kind: import('../data/contract').WindowKind }"), Label: "Commissioner: choose a league date", Args: strings.Fields("kind")},
		{ID: "phase.goto", Arguments: browserType("{ phase: 'OFFSEASON' | 'DRAFT' | 'REGULAR_SEASON' | 'PLAYOFFS' }"), Label: "Open phase workspace", Args: strings.Fields("phase")},
		{ID: "inspector.pin", Arguments: browserType("{ entity: import('../shell/state').EntitySubject | import('../shell/state').PlayerSubject }"), Label: "Inspect entity", Args: strings.Fields("entity")},
		{ID: "refresh.run", Arguments: browserType("{ scope: string }"), Label: "Refresh data", Args: strings.Fields("scope"), Gravity: "G1"},
		{ID: "cache.run", Arguments: browserType("{ scope: string }"), Label: "Read data again", Args: strings.Fields("scope"), Gravity: "G1"},
		{ID: "data.run", Arguments: browserType("import('../datasources/state').DataAction"), Label: "Data settings", Args: strings.Fields("kind"), Gravity: "G1"},
		{ID: "data.parameter", Arguments: browserType("import('../datasources/state').ParamAction"), Label: "Change engine parameter", Args: strings.Fields("key position value"), Roles: []string{"admin"}, Gravity: "G1"},
		{ID: "contracts.set", Arguments: browserType("import('../contracts/state').ContractsAction"), Label: "Filter or sort contracts", Args: strings.Fields("field value")},
		{ID: "market", Arguments: browserType("import('../market/contract').MarketFilter"), Label: "Market", Args: strings.Fields("field value")},
		{ID: "trade.builder", Arguments: browserType("import('../shell/tradeBuilderState').BuilderAction"), Label: "Trade builder", Args: strings.Fields("action"), Gravity: "G3", Undo: "irreversible"},
		{ID: "acquisition", Arguments: browserType("import('../contracts/acquisition').AcquisitionRequest"), Label: "What does the add cost me?", Args: strings.Fields("mine")},
		{ID: "trade.accept.plan", Arguments: browserType("{ tradeId: string }"), Label: "Plan accept", Args: strings.Fields("tradeId"), Roles: []string{"gm"}, Gravity: "G2", Undo: "reversible"},
		{ID: "lineup.edit", Arguments: browserType("{ franchiseId: string }"), Label: "Edit lineup", Args: strings.Fields("franchiseId")},
		{ID: "lineup.toggle", Arguments: browserType("{ playerId: string }"), Label: "Toggle starter", Args: strings.Fields("playerId")},
		{ID: "lineup.reset", Arguments: browserType("Record<string, never>"), Label: "Reset"},
		{ID: "lineup.cancel", Arguments: browserType("Record<string, never>"), Label: "Cancel"},
		{ID: "lineup.draft", Arguments: browserType("Record<string, never>"), Label: "Check and save plan", Gravity: "G1", Undo: "reversible"},
		{ID: "move.withdraw", Arguments: browserType("{ correlationId: string }"), Label: "Remove roster plan", Args: strings.Fields("correlationId")},
		{ID: "move.handoff", Arguments: browserType("{ correlationId: string }"), Label: "Open MFL page", Args: strings.Fields("correlationId"), Gravity: "G1", Undo: "reversible"},
		{ID: "mflkey.connect", Arguments: browserType("Record<string, never>"), Label: "Connect MFL", Aliases: []string{"mfl", "api key"}, Gravity: "G1", Undo: "reversible"},
		{ID: "mflkey.forget", Arguments: browserType("Record<string, never>"), Label: "Forget MFL key", Gravity: "G1", Undo: "reversible"},
		{ID: "roster.ir", Arguments: browserType("{ subject: import('../shell/state').PlayerSubject }"), Label: "Draft IR move", Args: strings.Fields("subject"), Roles: []string{"gm"}, Gravity: "G2", Undo: "reversible"},
		{ID: "roster.taxi", Arguments: browserType("{ subject: import('../shell/state').PlayerSubject }"), Label: "Draft move to taxi squad", Args: strings.Fields("subject"), Roles: []string{"gm"}, Gravity: "G2", Undo: "reversible"},
		{ID: "surface.open", Arguments: browserType("{ place: import('../registry').Surface }"), Label: "Open shell surface", Args: strings.Fields("place"), Aliases: []string{"surface"}},
		{ID: "surface.close", Arguments: browserType("Record<string, never>"), Label: "Close endpoint surface"},
		{ID: "report.sort", Arguments: browserType("{ id: string; column: number }"), Label: "Sort report", Args: strings.Fields("id column")},
		{ID: "market.reports", Arguments: browserType("{ action: 'refresh' }"), Label: "Player market reports", Args: strings.Fields("action")},
		{ID: "comms.read", Arguments: browserType("{ action: 'refresh' | 'team' | 'kind' | 'more' | 'open'; value: string }"), Label: "League communications", Args: strings.Fields("action value")},
		{ID: "backup.now", Arguments: browserType("Record<string, never>"), Label: "Back up now", Aliases: []string{"backup", "save a backup"}, Gravity: "G1"},
		{ID: "commissioner.surface", Arguments: browserType("import('../commissioner/surfaceActions').SurfaceAction"), Label: "Change the league view", Args: strings.Fields("screen action value")},
		{ID: "commissioner.read", Arguments: browserType("{ league: string }"), Label: "Read the league ledger", Args: strings.Fields("league")},
		{ID: "commissioner.draft", Arguments: browserType("{ action: 'add' | 'remove'; path: string; index: number }"), Label: "Edit a rule value", Args: strings.Fields("action path index")},
		{ID: "commissioner.execute", Arguments: browserType("{ body: string }"), Label: "Record a league change", Args: strings.Fields("body")},
		{ID: "control.read", Arguments: browserType("{ action: 'league' | 'news' | 'search' | 'open-accounting'; value: string }"), Label: "Control Room league and site news", Args: strings.Fields("action value")},
		{ID: "research.read", Arguments: browserType("{ action: string; value: string }"), Label: "Player research", Args: strings.Fields("action value")},
		{ID: "race.reports", Arguments: browserType("{ action: 'refresh' | 'week' | 'team'; value: string }"), Label: "League race reports", Args: strings.Fields("action value")},
		{ID: "game.follow", Arguments: browserType("{ id: string }"), Label: "Follow NFL game", Args: strings.Fields("id")},
		{ID: "now.reports", Arguments: browserType("{ action: 'refresh' | 'week' | 'reports-week'; value: string }"), Label: "This week in the league", Args: strings.Fields("action value")},
		{ID: "draft.reports", Arguments: browserType("{ action: 'refresh' | 'mfl' }"), Label: "Draft reports", Args: strings.Fields("action")},
	}
}

func browserShellCommands() []BrowserCommand {
	return []BrowserCommand{
		{ID: "endpoint.open", Arguments: browserType("{ id: import('../registry').EndpointId }"), Label: "Open endpoint", Args: strings.Fields("id"), Aliases: []string{"endpoint"}},
		{ID: "franchise.set", Arguments: browserType("{ franchiseId: string }"), Label: "View a franchise", Args: strings.Fields("franchiseId"), Aliases: []string{"view team"}},
		{ID: "preset.apply", Arguments: browserType("{ preset: import('../shell/presets').PresetId }"), Label: "Apply preset", Args: strings.Fields("preset"), Aliases: []string{"view"}},
		{ID: "inspector.open", Arguments: browserType("{ subject: import('../shell/state').PlayerSubject }"), Label: "Inspect player", Args: strings.Fields("subject"), Aliases: []string{"player"}},
		{ID: "inspector.collapse", Arguments: browserType("Record<string, never>"), Label: "Collapse inspector", Aliases: []string{"collapse"}},
		{ID: "commandbar.close", Arguments: browserType("Record<string, never>"), Label: "Close command bar"},
		{ID: "nav.open", Arguments: browserType("import('../shell/nodes').Route | { node: import('../shell/nodes').Node; workspace?: undefined }"), Label: "Open workspace", Args: strings.Fields("node workspace"), Aliases: []string{"goto", "navigate"}},
		{ID: "density.set", Arguments: browserType("{ density: import('../cards/Card').Density }"), Label: "Set density", Args: strings.Fields("density"), Aliases: []string{"density"}, Keys: []BrowserKey{{Key: "N", Args: map[string]string{"density": "narrative"}}, {Key: "T", Args: map[string]string{"density": "tactical"}}, {Key: "M", Args: map[string]string{"density": "matrix"}}, {Key: "1", Args: map[string]string{"density": "narrative"}}, {Key: "2", Args: map[string]string{"density": "tactical"}}, {Key: "3", Args: map[string]string{"density": "matrix"}}}},
		{ID: "inspector.expand", Arguments: browserType("Record<string, never>"), Label: "Expand inspector", Aliases: []string{"expand"}},
		{ID: "inspector.toggle", Arguments: browserType("Record<string, never>"), Label: "Toggle inspector", Aliases: []string{"inspect"}, Keys: []BrowserKey{{Key: "I", Args: map[string]string{}}}},
		{ID: "inspector.close", Arguments: browserType("Record<string, never>"), Label: "Close inspector", Aliases: []string{"close inspector"}},
		{ID: "calendar.summon", Arguments: browserType("Record<string, never>"), Label: "Summon calendar", Aliases: []string{"calendar"}},
		{ID: "comms.toggle", Arguments: browserType("Record<string, never>"), Label: "Toggle comms", Aliases: []string{"comms", "chat"}},
		{ID: "overlays.close", Arguments: browserType("Record<string, never>"), Label: "Close overlays", Aliases: []string{"escape"}, Keys: []BrowserKey{{Key: "Escape", Args: map[string]string{}}}},
		{ID: "commandbar.open", Arguments: browserType("Record<string, never>"), Label: "Type a command", Aliases: []string{"command", "/"}, Keys: []BrowserKey{{Key: "/", Args: map[string]string{}}, {Key: "Ctrl+K", Args: map[string]string{}}}},
	}
}

func browserType(clientType string) Schema {
	return Schema{Type: "json", Validator: "typed local browser handler", ClientType: clientType}
}
