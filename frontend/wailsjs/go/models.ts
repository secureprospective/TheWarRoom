export namespace accounts {
	
	export class SeatStatus {
	    hasAccount: boolean;
	    username?: string;
	
	    static createFrom(source: any = {}) {
	        return new SeatStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.hasAccount = source["hasAccount"];
	        this.username = source["username"];
	    }
	}

}

export namespace archive {
	
	export class ArchiveBoard {
	    board: report.Board;
	    season: number;
	    seasons: report.Choice[];
	    team: string;
	    other: string;
	    teams: report.Choice[];
	
	    static createFrom(source: any = {}) {
	        return new ArchiveBoard(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.board = this.convertValues(source["board"], report.Board);
	        this.season = source["season"];
	        this.seasons = this.convertValues(source["seasons"], report.Choice);
	        this.team = source["team"];
	        this.other = source["other"];
	        this.teams = this.convertValues(source["teams"], report.Choice);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace bid {
	
	export class Result {
	    rule: string;
	    label: string;
	    verdict: string;
	    reason: string;
	    mark: string;
	
	    static createFrom(source: any = {}) {
	        return new Result(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rule = source["rule"];
	        this.label = source["label"];
	        this.verdict = source["verdict"];
	        this.reason = source["reason"];
	        this.mark = source["mark"];
	    }
	}

}

export namespace commissioner {
	
	export class AdjustedRecord {
	    franchise: string;
	    wins: number;
	    losses: number;
	    ties: number;
	    entry: string;
	    reason: string;
	
	    static createFrom(source: any = {}) {
	        return new AdjustedRecord(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.franchise = source["franchise"];
	        this.wins = source["wins"];
	        this.losses = source["losses"];
	        this.ties = source["ties"];
	        this.entry = source["entry"];
	        this.reason = source["reason"];
	    }
	}
	export class BackupResult {
	    set: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new BackupResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.set = source["set"];
	        this.message = source["message"];
	    }
	}
	export class BidResult {
	    id: string;
	    refusal?: string;
	
	    static createFrom(source: any = {}) {
	        return new BidResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.refusal = source["refusal"];
	    }
	}
	export class BidOutcome {
	    player?: string;
	    season: number;
	    points?: number;
	    late?: boolean;
	    winner?: string;
	    winningBid?: string;
	    tied?: string[];
	    results?: BidResult[];
	
	    static createFrom(source: any = {}) {
	        return new BidOutcome(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.player = source["player"];
	        this.season = source["season"];
	        this.points = source["points"];
	        this.late = source["late"];
	        this.winner = source["winner"];
	        this.winningBid = source["winningBid"];
	        this.tied = source["tied"];
	        this.results = this.convertValues(source["results"], BidResult);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class BracketChampion {
	    franchise: string;
	    name: string;
	    label: string;
	    source: string;
	    record: string;
	
	    static createFrom(source: any = {}) {
	        return new BracketChampion(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.franchise = source["franchise"];
	        this.name = source["name"];
	        this.label = source["label"];
	        this.source = source["source"];
	        this.record = source["record"];
	    }
	}
	export class BracketNodeSpec {
	    id: string;
	    week: number;
	    homeSeed?: number;
	    awaySeed?: number;
	    homeFrom?: string;
	    awayFrom?: string;
	
	    static createFrom(source: any = {}) {
	        return new BracketNodeSpec(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.week = source["week"];
	        this.homeSeed = source["homeSeed"];
	        this.awaySeed = source["awaySeed"];
	        this.homeFrom = source["homeFrom"];
	        this.awayFrom = source["awayFrom"];
	    }
	}
	export class BracketTitle {
	    season: number;
	    label: string;
	    default: boolean;
	    entry?: string;
	    reason?: string;
	
	    static createFrom(source: any = {}) {
	        return new BracketTitle(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.season = source["season"];
	        this.label = source["label"];
	        this.default = source["default"];
	        this.entry = source["entry"];
	        this.reason = source["reason"];
	    }
	}
	export class Calendar {
	    league: string;
	    revision: number;
	    clock: leagueclock.ActionCalendar;
	    decision: leagueclock.ActionDecision;
	
	    static createFrom(source: any = {}) {
	        return new Calendar(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.league = source["league"];
	        this.revision = source["revision"];
	        this.clock = this.convertValues(source["clock"], leagueclock.ActionCalendar);
	        this.decision = this.convertValues(source["decision"], leagueclock.ActionDecision);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ControlField {
	    field: string;
	    label: string;
	
	    static createFrom(source: any = {}) {
	        return new ControlField(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.field = source["field"];
	        this.label = source["label"];
	    }
	}
	export class ControlEntry {
	    id: string;
	    built: boolean;
	    note: string;
	    fields: ControlField[];
	
	    static createFrom(source: any = {}) {
	        return new ControlEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.built = source["built"];
	        this.note = source["note"];
	        this.fields = this.convertValues(source["fields"], ControlField);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class ControlGroup {
	    id: string;
	    label: string;
	    controls: ControlEntry[];
	
	    static createFrom(source: any = {}) {
	        return new ControlGroup(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.label = source["label"];
	        this.controls = this.convertValues(source["controls"], ControlEntry);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Controls {
	    groups: ControlGroup[];
	
	    static createFrom(source: any = {}) {
	        return new Controls(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.groups = this.convertValues(source["groups"], ControlGroup);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ExceptionScope {
	    franchise?: string;
	    player?: string;
	    transaction?: string;
	
	    static createFrom(source: any = {}) {
	        return new ExceptionScope(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.franchise = source["franchise"];
	        this.player = source["player"];
	        this.transaction = source["transaction"];
	    }
	}
	export class CorrectionPick {
	    season: number;
	    round: number;
	    original: string;
	
	    static createFrom(source: any = {}) {
	        return new CorrectionPick(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.season = source["season"];
	        this.round = source["round"];
	        this.original = source["original"];
	    }
	}
	export class Correction {
	    entry: string;
	    who: string;
	    when: time.Time;
	    why: string;
	    what: string;
	    player?: string;
	    franchise?: string;
	    pick?: CorrectionPick;
	    rule?: string;
	    scope?: ExceptionScope;
	    until?: string;
	
	    static createFrom(source: any = {}) {
	        return new Correction(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.entry = source["entry"];
	        this.who = source["who"];
	        this.when = this.convertValues(source["when"], time.Time);
	        this.why = source["why"];
	        this.what = source["what"];
	        this.player = source["player"];
	        this.franchise = source["franchise"];
	        this.pick = this.convertValues(source["pick"], CorrectionPick);
	        this.rule = source["rule"];
	        this.scope = this.convertValues(source["scope"], ExceptionScope);
	        this.until = source["until"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class Corrections {
	    league: string;
	    revision: number;
	    items: Correction[];
	
	    static createFrom(source: any = {}) {
	        return new Corrections(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.league = source["league"];
	        this.revision = source["revision"];
	        this.items = this.convertValues(source["items"], Correction);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Digest {
	    id: string;
	    sentence: string;
	    when: time.Time;
	
	    static createFrom(source: any = {}) {
	        return new Digest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.sentence = source["sentence"];
	        this.when = this.convertValues(source["when"], time.Time);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Digests {
	    league: string;
	    digests: Digest[];
	
	    static createFrom(source: any = {}) {
	        return new Digests(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.league = source["league"];
	        this.digests = this.convertValues(source["digests"], Digest);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DraftCheckpoint {
	    entry: string;
	    words: string;
	    at: time.Time;
	
	    static createFrom(source: any = {}) {
	        return new DraftCheckpoint(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.entry = source["entry"];
	        this.words = source["words"];
	        this.at = this.convertValues(source["at"], time.Time);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DraftIssued {
	    season: number;
	    rounds: number;
	    picks: number;
	    tradable: boolean;
	
	    static createFrom(source: any = {}) {
	        return new DraftIssued(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.season = source["season"];
	        this.rounds = source["rounds"];
	        this.picks = source["picks"];
	        this.tradable = source["tradable"];
	    }
	}
	export class DraftPlayer {
	    id: string;
	    name: string;
	    position: string;
	    team: string;
	
	    static createFrom(source: any = {}) {
	        return new DraftPlayer(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.position = source["position"];
	        this.team = source["team"];
	    }
	}
	export class DraftClosed {
	    at: time.Time;
	    automatic: boolean;
	    pool: DraftPlayer[];
	    issued: DraftIssued[];
	
	    static createFrom(source: any = {}) {
	        return new DraftClosed(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.at = this.convertValues(source["at"], time.Time);
	        this.automatic = source["automatic"];
	        this.pool = this.convertValues(source["pool"], DraftPlayer);
	        this.issued = this.convertValues(source["issued"], DraftIssued);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class NativePick {
	    compensatory?: boolean;
	    correction?: string;
	    season: number;
	    round: number;
	    slot: number;
	    original: string;
	    owner: string;
	    player: string;
	    skipped: boolean;
	
	    static createFrom(source: any = {}) {
	        return new NativePick(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.compensatory = source["compensatory"];
	        this.correction = source["correction"];
	        this.season = source["season"];
	        this.round = source["round"];
	        this.slot = source["slot"];
	        this.original = source["original"];
	        this.owner = source["owner"];
	        this.player = source["player"];
	        this.skipped = source["skipped"];
	    }
	}
	export class NativePoolEntry {
	    player: string;
	    name?: string;
	    season: number;
	
	    static createFrom(source: any = {}) {
	        return new NativePoolEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.player = source["player"];
	        this.name = source["name"];
	        this.season = source["season"];
	    }
	}
	export class DraftOutcome {
	    terms?: rookiedraft.Terms;
	    franchise?: string;
	    slot?: number;
	    pool?: NativePoolEntry[];
	    issued?: NativePick[];
	
	    static createFrom(source: any = {}) {
	        return new DraftOutcome(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.terms = this.convertValues(source["terms"], rookiedraft.Terms);
	        this.franchise = source["franchise"];
	        this.slot = source["slot"];
	        this.pool = this.convertValues(source["pool"], NativePoolEntry);
	        this.issued = this.convertValues(source["issued"], NativePick);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DraftOwner {
	    id: string;
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new DraftOwner(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	    }
	}
	export class DraftPickInput {
	    league: string;
	    season: number;
	    franchise: string;
	    player: string;
	    request: string;
	    expected: number;
	
	    static createFrom(source: any = {}) {
	        return new DraftPickInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.league = source["league"];
	        this.season = source["season"];
	        this.franchise = source["franchise"];
	        this.player = source["player"];
	        this.request = source["request"];
	        this.expected = source["expected"];
	    }
	}
	
	export class Entry {
	    id: string;
	    who: string;
	    what: string;
	    why: string;
	    when: time.Time;
	    gravity: string;
	    undo: string;
	    replacedAt?: string;
	
	    static createFrom(source: any = {}) {
	        return new Entry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.who = source["who"];
	        this.what = source["what"];
	        this.why = source["why"];
	        this.when = this.convertValues(source["when"], time.Time);
	        this.gravity = source["gravity"];
	        this.undo = source["undo"];
	        this.replacedAt = source["replacedAt"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class FlippedGame {
	    week: number;
	    home: string;
	    away: string;
	    before: string;
	    after: string;
	    decided?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new FlippedGame(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.week = source["week"];
	        this.home = source["home"];
	        this.away = source["away"];
	        this.before = source["before"];
	        this.after = source["after"];
	        this.decided = source["decided"];
	    }
	}
	export class FranchiseGame {
	    week: number;
	    game?: string;
	    opponent?: string;
	    venue: string;
	    kickoff?: time.Time;
	
	    static createFrom(source: any = {}) {
	        return new FranchiseGame(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.week = source["week"];
	        this.game = source["game"];
	        this.opponent = source["opponent"];
	        this.venue = source["venue"];
	        this.kickoff = this.convertValues(source["kickoff"], time.Time);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class FranchiseSchedule {
	    league: string;
	    franchise: string;
	    season: number;
	    revision: number;
	    message?: string;
	    games: FranchiseGame[];
	
	    static createFrom(source: any = {}) {
	        return new FranchiseSchedule(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.league = source["league"];
	        this.franchise = source["franchise"];
	        this.season = source["season"];
	        this.revision = source["revision"];
	        this.message = source["message"];
	        this.games = this.convertValues(source["games"], FranchiseGame);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ScoreChange {
	    entry: string;
	    what: string;
	    reason: string;
	    status: string;
	
	    static createFrom(source: any = {}) {
	        return new ScoreChange(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.entry = source["entry"];
	        this.what = source["what"];
	        this.reason = source["reason"];
	        this.status = source["status"];
	    }
	}
	export class StarterScore {
	    player: string;
	    name: string;
	    position?: string;
	    words: string;
	    points?: number;
	
	    static createFrom(source: any = {}) {
	        return new StarterScore(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.player = source["player"];
	        this.name = source["name"];
	        this.position = source["position"];
	        this.words = source["words"];
	        this.points = source["points"];
	    }
	}
	export class FranchiseWeekScore {
	    franchise: string;
	    name: string;
	    status: string;
	    points?: number;
	    words: string;
	    starters: StarterScore[];
	    adjustment: number;
	    changes: ScoreChange[];
	    lineup: nativecheck.LineupVerdict;
	    zeroed?: string;
	
	    static createFrom(source: any = {}) {
	        return new FranchiseWeekScore(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.franchise = source["franchise"];
	        this.name = source["name"];
	        this.status = source["status"];
	        this.points = source["points"];
	        this.words = source["words"];
	        this.starters = this.convertValues(source["starters"], StarterScore);
	        this.adjustment = source["adjustment"];
	        this.changes = this.convertValues(source["changes"], ScoreChange);
	        this.lineup = this.convertValues(source["lineup"], nativecheck.LineupVerdict);
	        this.zeroed = source["zeroed"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Governed {
	    entry: string;
	    who: string;
	    when: time.Time;
	    change: string;
	
	    static createFrom(source: any = {}) {
	        return new Governed(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.entry = source["entry"];
	        this.who = source["who"];
	        this.when = this.convertValues(source["when"], time.Time);
	        this.change = source["change"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class HistoryRecord {
	    record: string;
	    franchise: string;
	    name: string;
	    byCommissioner: boolean;
	    reason?: string;
	
	    static createFrom(source: any = {}) {
	        return new HistoryRecord(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.record = source["record"];
	        this.franchise = source["franchise"];
	        this.name = source["name"];
	        this.byCommissioner = source["byCommissioner"];
	        this.reason = source["reason"];
	    }
	}
	export class HouseRule {
	    id: string;
	    text: string;
	
	    static createFrom(source: any = {}) {
	        return new HouseRule(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.text = source["text"];
	    }
	}
	export class Job {
	    id: string;
	    name: string;
	    state: string;
	    words: string;
	    opens: string;
	    closes: string;
	    controls: string[];
	
	    static createFrom(source: any = {}) {
	        return new Job(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.state = source["state"];
	        this.words = source["words"];
	        this.opens = source["opens"];
	        this.closes = source["closes"];
	        this.controls = source["controls"];
	    }
	}
	export class Jobs {
	    league: string;
	    name: string;
	    revision: number;
	    jobs: Job[];
	
	    static createFrom(source: any = {}) {
	        return new Jobs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.league = source["league"];
	        this.name = source["name"];
	        this.revision = source["revision"];
	        this.jobs = this.convertValues(source["jobs"], Job);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class MailLog {
	    words: string;
	    recorded: Note[];
	
	    static createFrom(source: any = {}) {
	        return new MailLog(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.words = source["words"];
	        this.recorded = this.convertValues(source["recorded"], Note);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Release {
	    date: string;
	    title: string;
	    words: string;
	
	    static createFrom(source: any = {}) {
	        return new Release(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.date = source["date"];
	        this.title = source["title"];
	        this.words = source["words"];
	    }
	}
	export class Note {
	    id: string;
	    kind: string;
	    text: string;
	    pinned: boolean;
	    who: string;
	    when: time.Time;
	    withdrawn: boolean;
	    change: string;
	
	    static createFrom(source: any = {}) {
	        return new Note(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.kind = source["kind"];
	        this.text = source["text"];
	        this.pinned = source["pinned"];
	        this.who = source["who"];
	        this.when = this.convertValues(source["when"], time.Time);
	        this.withdrawn = source["withdrawn"];
	        this.change = source["change"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class LeagueNotes {
	    league: string;
	    revision: number;
	    notices: Note[];
	    notes: Note[];
	    releases: Release[];
	    mail: MailLog;
	
	    static createFrom(source: any = {}) {
	        return new LeagueNotes(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.league = source["league"];
	        this.revision = source["revision"];
	        this.notices = this.convertValues(source["notices"], Note);
	        this.notes = this.convertValues(source["notes"], Note);
	        this.releases = this.convertValues(source["releases"], Release);
	        this.mail = this.convertValues(source["mail"], MailLog);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Ledger {
	    league: string;
	    name: string;
	    revision: number;
	    entries: Entry[];
	
	    static createFrom(source: any = {}) {
	        return new Ledger(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.league = source["league"];
	        this.name = source["name"];
	        this.revision = source["revision"];
	        this.entries = this.convertValues(source["entries"], Entry);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class LineupPlayer {
	    player: string;
	    name: string;
	    position: string;
	    team?: string;
	    locked: boolean;
	    locksAt?: time.Time;
	    words: string;
	
	    static createFrom(source: any = {}) {
	        return new LineupPlayer(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.player = source["player"];
	        this.name = source["name"];
	        this.position = source["position"];
	        this.team = source["team"];
	        this.locked = source["locked"];
	        this.locksAt = this.convertValues(source["locksAt"], time.Time);
	        this.words = source["words"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class ManagedSchedule {
	    league: string;
	    revision: number;
	    draft: competition.ScheduleWeek[];
	    published: competition.ScheduleWeek[];
	
	    static createFrom(source: any = {}) {
	        return new ManagedSchedule(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.league = source["league"];
	        this.revision = source["revision"];
	        this.draft = this.convertValues(source["draft"], competition.ScheduleWeek);
	        this.published = this.convertValues(source["published"], competition.ScheduleWeek);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class MissedLineup {
	    franchise: string;
	    name: string;
	    flagged: boolean;
	
	    static createFrom(source: any = {}) {
	        return new MissedLineup(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.franchise = source["franchise"];
	        this.name = source["name"];
	        this.flagged = source["flagged"];
	    }
	}
	export class NativeCharge {
	    kind?: string;
	    season?: number;
	    id: string;
	    franchise: string;
	    amountCents: number;
	    description: string;
	
	    static createFrom(source: any = {}) {
	        return new NativeCharge(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.season = source["season"];
	        this.id = source["id"];
	        this.franchise = source["franchise"];
	        this.amountCents = source["amountCents"];
	        this.description = source["description"];
	    }
	}
	export class NativePlayer {
	    acquiredVia?: string;
	    correction?: string;
	    name?: string;
	    rookieRound?: number;
	    pending?: boolean;
	    restructured?: boolean;
	    id: string;
	    franchise: string;
	    status: string;
	    salaryCents: number;
	    expirationYear: number;
	    contractStatus: string;
	    contractInfo: string;
	
	    static createFrom(source: any = {}) {
	        return new NativePlayer(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.acquiredVia = source["acquiredVia"];
	        this.correction = source["correction"];
	        this.name = source["name"];
	        this.rookieRound = source["rookieRound"];
	        this.pending = source["pending"];
	        this.restructured = source["restructured"];
	        this.id = source["id"];
	        this.franchise = source["franchise"];
	        this.status = source["status"];
	        this.salaryCents = source["salaryCents"];
	        this.expirationYear = source["expirationYear"];
	        this.contractStatus = source["contractStatus"];
	        this.contractInfo = source["contractInfo"];
	    }
	}
	export class NativeFranchise {
	    id: string;
	    heldId: string;
	    name: string;
	    abbreviation: string;
	    division: string;
	
	    static createFrom(source: any = {}) {
	        return new NativeFranchise(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.heldId = source["heldId"];
	        this.name = source["name"];
	        this.abbreviation = source["abbreviation"];
	        this.division = source["division"];
	    }
	}
	export class NativeChampion {
	    season: number;
	    franchise: string;
	
	    static createFrom(source: any = {}) {
	        return new NativeChampion(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.season = source["season"];
	        this.franchise = source["franchise"];
	    }
	}
	export class RawNativeSnapshot {
	    standingsOrder?: string[];
	    season: number;
	    champion?: string;
	    champions?: NativeChampion[];
	    conferences?: Record<string, string>;
	    rosterAt: time.Time;
	    draftAt: time.Time;
	    futureAt: time.Time;
	    franchises: NativeFranchise[];
	    players: NativePlayer[];
	    picks: NativePick[];
	    adjustments: NativeCharge[];
	    pool?: NativePoolEntry[];
	
	    static createFrom(source: any = {}) {
	        return new RawNativeSnapshot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.standingsOrder = source["standingsOrder"];
	        this.season = source["season"];
	        this.champion = source["champion"];
	        this.champions = this.convertValues(source["champions"], NativeChampion);
	        this.conferences = source["conferences"];
	        this.rosterAt = this.convertValues(source["rosterAt"], time.Time);
	        this.draftAt = this.convertValues(source["draftAt"], time.Time);
	        this.futureAt = this.convertValues(source["futureAt"], time.Time);
	        this.franchises = this.convertValues(source["franchises"], NativeFranchise);
	        this.players = this.convertValues(source["players"], NativePlayer);
	        this.picks = this.convertValues(source["picks"], NativePick);
	        this.adjustments = this.convertValues(source["adjustments"], NativeCharge);
	        this.pool = this.convertValues(source["pool"], NativePoolEntry);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class NativeAssets {
	    corrections?: Correction[];
	    league: string;
	    revision: number;
	    seeded: boolean;
	    entry: string;
	    source: string;
	    snapshot?: RawNativeSnapshot;
	
	    static createFrom(source: any = {}) {
	        return new NativeAssets(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.corrections = this.convertValues(source["corrections"], Correction);
	        this.league = source["league"];
	        this.revision = source["revision"];
	        this.seeded = source["seeded"];
	        this.entry = source["entry"];
	        this.source = source["source"];
	        this.snapshot = this.convertValues(source["snapshot"], RawNativeSnapshot);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PlayoffRulesView {
	    words: string;
	    perConference: number;
	    divisionWinners: number;
	    wildCards: number;
	    byeSeeds: number;
	    seeding: string;
	    reseed: boolean;
	    startWeek: number;
	    finalWeek: number;
	    homeBonus: number;
	    shape: BracketNodeSpec[];
	
	    static createFrom(source: any = {}) {
	        return new PlayoffRulesView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.words = source["words"];
	        this.perConference = source["perConference"];
	        this.divisionWinners = source["divisionWinners"];
	        this.wildCards = source["wildCards"];
	        this.byeSeeds = source["byeSeeds"];
	        this.seeding = source["seeding"];
	        this.reseed = source["reseed"];
	        this.startWeek = source["startWeek"];
	        this.finalWeek = source["finalWeek"];
	        this.homeBonus = source["homeBonus"];
	        this.shape = this.convertValues(source["shape"], BracketNodeSpec);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class NativeBracket {
	    league: string;
	    name: string;
	    season: number;
	    revision: number;
	    status: string;
	    words: string;
	    preset: string;
	    presets: string[];
	    rules: PlayoffRulesView;
	    started: boolean;
	    startKnown: boolean;
	    fields: competition.ConferenceField[];
	    rounds: competition.BracketRound[];
	    champion?: BracketChampion;
	    title: BracketTitle;
	    history: HistoryRecord[];
	    changes: ScoreChange[];
	
	    static createFrom(source: any = {}) {
	        return new NativeBracket(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.league = source["league"];
	        this.name = source["name"];
	        this.season = source["season"];
	        this.revision = source["revision"];
	        this.status = source["status"];
	        this.words = source["words"];
	        this.preset = source["preset"];
	        this.presets = source["presets"];
	        this.rules = this.convertValues(source["rules"], PlayoffRulesView);
	        this.started = source["started"];
	        this.startKnown = source["startKnown"];
	        this.fields = this.convertValues(source["fields"], competition.ConferenceField);
	        this.rounds = this.convertValues(source["rounds"], competition.BracketRound);
	        this.champion = this.convertValues(source["champion"], BracketChampion);
	        this.title = this.convertValues(source["title"], BracketTitle);
	        this.history = this.convertValues(source["history"], HistoryRecord);
	        this.changes = this.convertValues(source["changes"], ScoreChange);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class NativeCapCharge {
	    id: string;
	    description: string;
	    amountCents?: number;
	
	    static createFrom(source: any = {}) {
	        return new NativeCapCharge(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.description = source["description"];
	        this.amountCents = source["amountCents"];
	    }
	}
	export class NativeCapFranchise {
	    exception?: string;
	    capCents?: number;
	    salaryCents?: number;
	    activeCents?: number;
	    irCents?: number;
	    taxiCents?: number;
	    adjustmentCents?: number;
	    committedCents?: number;
	    roomCents?: number;
	    quantumCents?: number;
	    mode: string;
	    deadline?: time.Time;
	    words: string;
	    enforcement: string;
	    franchise: string;
	    name: string;
	    abbreviation: string;
	    charges: NativeCapCharge[];
	
	    static createFrom(source: any = {}) {
	        return new NativeCapFranchise(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.exception = source["exception"];
	        this.capCents = source["capCents"];
	        this.salaryCents = source["salaryCents"];
	        this.activeCents = source["activeCents"];
	        this.irCents = source["irCents"];
	        this.taxiCents = source["taxiCents"];
	        this.adjustmentCents = source["adjustmentCents"];
	        this.committedCents = source["committedCents"];
	        this.roomCents = source["roomCents"];
	        this.quantumCents = source["quantumCents"];
	        this.mode = source["mode"];
	        this.deadline = this.convertValues(source["deadline"], time.Time);
	        this.words = source["words"];
	        this.enforcement = source["enforcement"];
	        this.franchise = source["franchise"];
	        this.name = source["name"];
	        this.abbreviation = source["abbreviation"];
	        this.charges = this.convertValues(source["charges"], NativeCapCharge);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class RuleLine {
	    label: string;
	    words: string;
	
	    static createFrom(source: any = {}) {
	        return new RuleLine(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.label = source["label"];
	        this.words = source["words"];
	    }
	}
	export class NativeCapCheck {
	    corrections?: Correction[];
	    league: string;
	    name: string;
	    revision: number;
	    rosterSeason: number;
	    rulesSeason: number;
	    seeded: boolean;
	    source: string;
	    counting: string;
	    rules: RuleLine[];
	    franchises: NativeCapFranchise[];
	
	    static createFrom(source: any = {}) {
	        return new NativeCapCheck(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.corrections = this.convertValues(source["corrections"], Correction);
	        this.league = source["league"];
	        this.name = source["name"];
	        this.revision = source["revision"];
	        this.rosterSeason = source["rosterSeason"];
	        this.rulesSeason = source["rulesSeason"];
	        this.seeded = source["seeded"];
	        this.source = source["source"];
	        this.counting = source["counting"];
	        this.rules = this.convertValues(source["rules"], RuleLine);
	        this.franchises = this.convertValues(source["franchises"], NativeCapFranchise);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	
	export class NativeDirectorySeat {
	    franchise: string;
	    name: string;
	    abbreviation: string;
	    owner: string;
	    caretaker?: string;
	    division: string;
	
	    static createFrom(source: any = {}) {
	        return new NativeDirectorySeat(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.franchise = source["franchise"];
	        this.name = source["name"];
	        this.abbreviation = source["abbreviation"];
	        this.owner = source["owner"];
	        this.caretaker = source["caretaker"];
	        this.division = source["division"];
	    }
	}
	export class NativeDirectory {
	    league: string;
	    name: string;
	    revision: number;
	    rosterSeason: number;
	    source: string;
	    franchises: NativeDirectorySeat[];
	
	    static createFrom(source: any = {}) {
	        return new NativeDirectory(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.league = source["league"];
	        this.name = source["name"];
	        this.revision = source["revision"];
	        this.rosterSeason = source["rosterSeason"];
	        this.source = source["source"];
	        this.franchises = this.convertValues(source["franchises"], NativeDirectorySeat);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class NativeDraft {
	    corrections?: Correction[];
	    league: string;
	    season: number;
	    revision: number;
	    rules: RuleLine[];
	    owners: DraftOwner[];
	    order: rookiedraft.RoundOrder[];
	    words: string;
	    draft: rookiedraft.Draft;
	    nextOwnedPick?: rookiedraft.Pick;
	    closed?: DraftClosed;
	    players: Record<string, DraftPlayer>;
	    available: DraftPlayer[];
	    checkpoints: DraftCheckpoint[];
	
	    static createFrom(source: any = {}) {
	        return new NativeDraft(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.corrections = this.convertValues(source["corrections"], Correction);
	        this.league = source["league"];
	        this.season = source["season"];
	        this.revision = source["revision"];
	        this.rules = this.convertValues(source["rules"], RuleLine);
	        this.owners = this.convertValues(source["owners"], DraftOwner);
	        this.order = this.convertValues(source["order"], rookiedraft.RoundOrder);
	        this.words = source["words"];
	        this.draft = this.convertValues(source["draft"], rookiedraft.Draft);
	        this.nextOwnedPick = this.convertValues(source["nextOwnedPick"], rookiedraft.Pick);
	        this.closed = this.convertValues(source["closed"], DraftClosed);
	        this.players = this.convertValues(source["players"], DraftPlayer, true);
	        this.available = this.convertValues(source["available"], DraftPlayer);
	        this.checkpoints = this.convertValues(source["checkpoints"], DraftCheckpoint);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class NativeGame {
	    week: number;
	    home: string;
	    away: string;
	    homeName: string;
	    awayName: string;
	    status: string;
	    homePoints?: number;
	    awayPoints?: number;
	    winner?: string;
	    words: string;
	    why: string[];
	
	    static createFrom(source: any = {}) {
	        return new NativeGame(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.week = source["week"];
	        this.home = source["home"];
	        this.away = source["away"];
	        this.homeName = source["homeName"];
	        this.awayName = source["awayName"];
	        this.status = source["status"];
	        this.homePoints = source["homePoints"];
	        this.awayPoints = source["awayPoints"];
	        this.winner = source["winner"];
	        this.words = source["words"];
	        this.why = source["why"];
	    }
	}
	export class NativeHouseRules {
	    league: string;
	    name: string;
	    revision: number;
	    season: number;
	    source: string;
	    words: string;
	    rules: HouseRule[];
	
	    static createFrom(source: any = {}) {
	        return new NativeHouseRules(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.league = source["league"];
	        this.name = source["name"];
	        this.revision = source["revision"];
	        this.season = source["season"];
	        this.source = source["source"];
	        this.words = source["words"];
	        this.rules = this.convertValues(source["rules"], HouseRule);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class NativeLeague {
	    id: string;
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new NativeLeague(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	    }
	}
	export class NativeLineup {
	    league: string;
	    name: string;
	    revision: number;
	    season: number;
	    week: number;
	    franchise: string;
	    franchiseName: string;
	    zone: string;
	    set: boolean;
	    corrected: boolean;
	    words: string;
	    reason?: string;
	    starters: LineupPlayer[];
	    bench: LineupPlayer[];
	    verdict: nativecheck.LineupVerdict;
	    rules: RuleLine[];
	
	    static createFrom(source: any = {}) {
	        return new NativeLineup(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.league = source["league"];
	        this.name = source["name"];
	        this.revision = source["revision"];
	        this.season = source["season"];
	        this.week = source["week"];
	        this.franchise = source["franchise"];
	        this.franchiseName = source["franchiseName"];
	        this.zone = source["zone"];
	        this.set = source["set"];
	        this.corrected = source["corrected"];
	        this.words = source["words"];
	        this.reason = source["reason"];
	        this.starters = this.convertValues(source["starters"], LineupPlayer);
	        this.bench = this.convertValues(source["bench"], LineupPlayer);
	        this.verdict = this.convertValues(source["verdict"], nativecheck.LineupVerdict);
	        this.rules = this.convertValues(source["rules"], RuleLine);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class NativeStarter {
	    player: string;
	    name: string;
	    position: string;
	
	    static createFrom(source: any = {}) {
	        return new NativeStarter(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.player = source["player"];
	        this.name = source["name"];
	        this.position = source["position"];
	    }
	}
	export class NativeLineupCheck {
	    league: string;
	    name: string;
	    revision: number;
	    rosterSeason: number;
	    rulesSeason: number;
	    franchise: string;
	    franchiseName: string;
	    source: string;
	    rules: RuleLine[];
	    starters: NativeStarter[];
	    verdict: nativecheck.LineupVerdict;
	
	    static createFrom(source: any = {}) {
	        return new NativeLineupCheck(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.league = source["league"];
	        this.name = source["name"];
	        this.revision = source["revision"];
	        this.rosterSeason = source["rosterSeason"];
	        this.rulesSeason = source["rulesSeason"];
	        this.franchise = source["franchise"];
	        this.franchiseName = source["franchiseName"];
	        this.source = source["source"];
	        this.rules = this.convertValues(source["rules"], RuleLine);
	        this.starters = this.convertValues(source["starters"], NativeStarter);
	        this.verdict = this.convertValues(source["verdict"], nativecheck.LineupVerdict);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	export class NativeScoreItem {
	    stat: string;
	    words: string;
	    points?: number;
	
	    static createFrom(source: any = {}) {
	        return new NativeScoreItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.stat = source["stat"];
	        this.words = source["words"];
	        this.points = source["points"];
	    }
	}
	export class NativePlayerScore {
	    league: string;
	    player: string;
	    season: number;
	    week: number;
	    status: string;
	    words: string;
	    points?: number;
	    partialPoints?: number;
	    position?: string;
	    positionWhy?: string;
	    missing: string[];
	    items: NativeScoreItem[];
	    changes: ScoreChange[];
	    sha256?: string;
	    fetchedAt?: time.Time;
	
	    static createFrom(source: any = {}) {
	        return new NativePlayerScore(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.league = source["league"];
	        this.player = source["player"];
	        this.season = source["season"];
	        this.week = source["week"];
	        this.status = source["status"];
	        this.words = source["words"];
	        this.points = source["points"];
	        this.partialPoints = source["partialPoints"];
	        this.position = source["position"];
	        this.positionWhy = source["positionWhy"];
	        this.missing = source["missing"];
	        this.items = this.convertValues(source["items"], NativeScoreItem);
	        this.changes = this.convertValues(source["changes"], ScoreChange);
	        this.sha256 = source["sha256"];
	        this.fetchedAt = this.convertValues(source["fetchedAt"], time.Time);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class NativeReserveFranchise {
	    franchise: string;
	    name: string;
	    abbreviation: string;
	    ir: nativecheck.SquadUse;
	    taxi: nativecheck.SquadUse;
	    players: nativecheck.ReserveVerdict[];
	
	    static createFrom(source: any = {}) {
	        return new NativeReserveFranchise(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.franchise = source["franchise"];
	        this.name = source["name"];
	        this.abbreviation = source["abbreviation"];
	        this.ir = this.convertValues(source["ir"], nativecheck.SquadUse);
	        this.taxi = this.convertValues(source["taxi"], nativecheck.SquadUse);
	        this.players = this.convertValues(source["players"], nativecheck.ReserveVerdict);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class NativeReserveCheck {
	    corrections?: Correction[];
	    league: string;
	    name: string;
	    revision: number;
	    rosterSeason: number;
	    rulesSeason: number;
	    seeded: boolean;
	    source: string;
	    rules: RuleLine[];
	    franchises: NativeReserveFranchise[];
	
	    static createFrom(source: any = {}) {
	        return new NativeReserveCheck(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.corrections = this.convertValues(source["corrections"], Correction);
	        this.league = source["league"];
	        this.name = source["name"];
	        this.revision = source["revision"];
	        this.rosterSeason = source["rosterSeason"];
	        this.rulesSeason = source["rulesSeason"];
	        this.seeded = source["seeded"];
	        this.source = source["source"];
	        this.rules = this.convertValues(source["rules"], RuleLine);
	        this.franchises = this.convertValues(source["franchises"], NativeReserveFranchise);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class NativeResults {
	    league: string;
	    season: number;
	    week: number;
	    revision: number;
	    published: boolean;
	    words: string;
	    games: NativeGame[];
	    bye?: string;
	    byeName?: string;
	    scores: FranchiseWeekScore[];
	    missed: MissedLineup[];
	
	    static createFrom(source: any = {}) {
	        return new NativeResults(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.league = source["league"];
	        this.season = source["season"];
	        this.week = source["week"];
	        this.revision = source["revision"];
	        this.published = source["published"];
	        this.words = source["words"];
	        this.games = this.convertValues(source["games"], NativeGame);
	        this.bye = source["bye"];
	        this.byeName = source["byeName"];
	        this.scores = this.convertValues(source["scores"], FranchiseWeekScore);
	        this.missed = this.convertValues(source["missed"], MissedLineup);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class NativeRosterFranchise {
	    franchise: string;
	    counted: number;
	    fits: boolean;
	    words: string;
	    problems: string[];
	    notes: string[];
	    name: string;
	    abbreviation: string;
	
	    static createFrom(source: any = {}) {
	        return new NativeRosterFranchise(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.franchise = source["franchise"];
	        this.counted = source["counted"];
	        this.fits = source["fits"];
	        this.words = source["words"];
	        this.problems = source["problems"];
	        this.notes = source["notes"];
	        this.name = source["name"];
	        this.abbreviation = source["abbreviation"];
	    }
	}
	export class NativeRosterCheck {
	    corrections?: Correction[];
	    league: string;
	    name: string;
	    revision: number;
	    rosterSeason: number;
	    rulesSeason: number;
	    seeded: boolean;
	    source: string;
	    rules: RuleLine[];
	    franchises: NativeRosterFranchise[];
	
	    static createFrom(source: any = {}) {
	        return new NativeRosterCheck(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.corrections = this.convertValues(source["corrections"], Correction);
	        this.league = source["league"];
	        this.name = source["name"];
	        this.revision = source["revision"];
	        this.rosterSeason = source["rosterSeason"];
	        this.rulesSeason = source["rulesSeason"];
	        this.seeded = source["seeded"];
	        this.source = source["source"];
	        this.rules = this.convertValues(source["rules"], RuleLine);
	        this.franchises = this.convertValues(source["franchises"], NativeRosterFranchise);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	export class OrderOverride {
	    order: string[];
	    entry: string;
	    reason: string;
	
	    static createFrom(source: any = {}) {
	        return new OrderOverride(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.order = source["order"];
	        this.entry = source["entry"];
	        this.reason = source["reason"];
	    }
	}
	export class StandingsOverrideView {
	    words: string;
	    rows: StandingRow[];
	    order?: OrderOverride;
	    records: Record<string, AdjustedRecord>;
	
	    static createFrom(source: any = {}) {
	        return new StandingsOverrideView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.words = source["words"];
	        this.rows = this.convertValues(source["rows"], StandingRow);
	        this.order = this.convertValues(source["order"], OrderOverride);
	        this.records = this.convertValues(source["records"], AdjustedRecord, true);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class StandingRow {
	    rank: number;
	    franchise: string;
	    name: string;
	    division?: string;
	    wins: number;
	    losses: number;
	    ties: number;
	    record: string;
	    computedRecord?: string;
	    pointsFor: number;
	    pointsWords: string;
	    why: string[];
	
	    static createFrom(source: any = {}) {
	        return new StandingRow(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rank = source["rank"];
	        this.franchise = source["franchise"];
	        this.name = source["name"];
	        this.division = source["division"];
	        this.wins = source["wins"];
	        this.losses = source["losses"];
	        this.ties = source["ties"];
	        this.record = source["record"];
	        this.computedRecord = source["computedRecord"];
	        this.pointsFor = source["pointsFor"];
	        this.pointsWords = source["pointsWords"];
	        this.why = source["why"];
	    }
	}
	export class NativeStandings {
	    league: string;
	    name: string;
	    season: number;
	    revision: number;
	    status: string;
	    words: string;
	    through: number;
	    decided: number;
	    pending: number;
	    criteria: string[];
	    noData: string[];
	    computed: StandingRow[];
	    override?: StandingsOverrideView;
	    changes: ScoreChange[];
	
	    static createFrom(source: any = {}) {
	        return new NativeStandings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.league = source["league"];
	        this.name = source["name"];
	        this.season = source["season"];
	        this.revision = source["revision"];
	        this.status = source["status"];
	        this.words = source["words"];
	        this.through = source["through"];
	        this.decided = source["decided"];
	        this.pending = source["pending"];
	        this.criteria = source["criteria"];
	        this.noData = source["noData"];
	        this.computed = this.convertValues(source["computed"], StandingRow);
	        this.override = this.convertValues(source["override"], StandingsOverrideView);
	        this.changes = this.convertValues(source["changes"], ScoreChange);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class NativeWeek {
	    schedule: FranchiseSchedule;
	    current?: FranchiseGame;
	    nextWeek?: number;
	    message?: string;
	
	    static createFrom(source: any = {}) {
	        return new NativeWeek(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.schedule = this.convertValues(source["schedule"], FranchiseSchedule);
	        this.current = this.convertValues(source["current"], FranchiseGame);
	        this.nextWeek = source["nextWeek"];
	        this.message = source["message"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	export class PendingExpiry {
	    player: string;
	    franchise: string;
	    route: string;
	
	    static createFrom(source: any = {}) {
	        return new PendingExpiry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.player = source["player"];
	        this.franchise = source["franchise"];
	        this.route = source["route"];
	    }
	}
	export class PeopleVoteCount {
	    choice: string;
	    ballots: number;
	
	    static createFrom(source: any = {}) {
	        return new PeopleVoteCount(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.choice = source["choice"];
	        this.ballots = source["ballots"];
	    }
	}
	export class PeopleVote {
	    id: string;
	    status: string;
	    ballotsCast: number;
	    totalFranchises: number;
	    votesRequired: number;
	    requirement: string;
	    endsAt: time.Time;
	    candidates: string[];
	    counts: PeopleVoteCount[];
	    votedFranchises?: string[];
	
	    static createFrom(source: any = {}) {
	        return new PeopleVote(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.status = source["status"];
	        this.ballotsCast = source["ballotsCast"];
	        this.totalFranchises = source["totalFranchises"];
	        this.votesRequired = source["votesRequired"];
	        this.requirement = source["requirement"];
	        this.endsAt = this.convertValues(source["endsAt"], time.Time);
	        this.candidates = source["candidates"];
	        this.counts = this.convertValues(source["counts"], PeopleVoteCount);
	        this.votedFranchises = source["votedFranchises"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PeopleInvite {
	    id: string;
	    franchise: string;
	    person: string;
	    status: string;
	
	    static createFrom(source: any = {}) {
	        return new PeopleInvite(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.franchise = source["franchise"];
	        this.person = source["person"];
	        this.status = source["status"];
	    }
	}
	export class PeopleRole {
	    person: string;
	    role: string;
	
	    static createFrom(source: any = {}) {
	        return new PeopleRole(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.person = source["person"];
	        this.role = source["role"];
	    }
	}
	export class PeopleSeat {
	    franchise: string;
	    name: string;
	    abbreviation: string;
	    owner: string;
	    caretaker?: string;
	
	    static createFrom(source: any = {}) {
	        return new PeopleSeat(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.franchise = source["franchise"];
	        this.name = source["name"];
	        this.abbreviation = source["abbreviation"];
	        this.owner = source["owner"];
	        this.caretaker = source["caretaker"];
	    }
	}
	export class People {
	    league: string;
	    name: string;
	    revision: number;
	    seats: PeopleSeat[];
	    roles: PeopleRole[];
	    invites: PeopleInvite[];
	    dotCount: number;
	    dotStatus: string;
	    votes: PeopleVote[];
	    succession: string;
	    yourRole: string;
	    yourFranchise: string;
	    mayChangeRules: boolean;
	    mayGrantRoles: boolean;
	    mayPostNotices: boolean;
	
	    static createFrom(source: any = {}) {
	        return new People(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.league = source["league"];
	        this.name = source["name"];
	        this.revision = source["revision"];
	        this.seats = this.convertValues(source["seats"], PeopleSeat);
	        this.roles = this.convertValues(source["roles"], PeopleRole);
	        this.invites = this.convertValues(source["invites"], PeopleInvite);
	        this.dotCount = source["dotCount"];
	        this.dotStatus = source["dotStatus"];
	        this.votes = this.convertValues(source["votes"], PeopleVote);
	        this.succession = source["succession"];
	        this.yourRole = source["yourRole"];
	        this.yourFranchise = source["yourFranchise"];
	        this.mayChangeRules = source["mayChangeRules"];
	        this.mayGrantRoles = source["mayGrantRoles"];
	        this.mayPostNotices = source["mayPostNotices"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	
	
	
	
	export class PreviewAffects {
	    readings: string[];
	    count?: number;
	    words: string;
	
	    static createFrom(source: any = {}) {
	        return new PreviewAffects(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.readings = source["readings"];
	        this.count = source["count"];
	        this.words = source["words"];
	    }
	}
	export class PreviewField {
	    label: string;
	    before: string;
	    beforeSource: string;
	    after: string;
	    afterSource: string;
	    when: string;
	
	    static createFrom(source: any = {}) {
	        return new PreviewField(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.label = source["label"];
	        this.before = source["before"];
	        this.beforeSource = source["beforeSource"];
	        this.after = source["after"];
	        this.afterSource = source["afterSource"];
	        this.when = source["when"];
	    }
	}
	export class PreviewTarget {
	    league: string;
	    name: string;
	    version: number;
	
	    static createFrom(source: any = {}) {
	        return new PreviewTarget(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.league = source["league"];
	        this.name = source["name"];
	        this.version = source["version"];
	    }
	}
	export class Preview {
	    question: string;
	    target: PreviewTarget;
	    fields: PreviewField[];
	    affects: PreviewAffects;
	    blockers: string[];
	    safeguard: string;
	
	    static createFrom(source: any = {}) {
	        return new Preview(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.question = source["question"];
	        this.target = this.convertValues(source["target"], PreviewTarget);
	        this.fields = this.convertValues(source["fields"], PreviewField);
	        this.affects = this.convertValues(source["affects"], PreviewAffects);
	        this.blockers = source["blockers"];
	        this.safeguard = source["safeguard"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	
	export class RawContractAction {
	    action: string;
	    on?: boolean;
	    params: number[];
	
	    static createFrom(source: any = {}) {
	        return new RawContractAction(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.action = source["action"];
	        this.on = source["on"];
	        this.params = source["params"];
	    }
	}
	export class RawHistoryLink {
	    record: string;
	    franchise: string;
	
	    static createFrom(source: any = {}) {
	        return new RawHistoryLink(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.record = source["record"];
	        this.franchise = source["franchise"];
	    }
	}
	export class RawTitleChange {
	    season: number;
	    label?: string;
	
	    static createFrom(source: any = {}) {
	        return new RawTitleChange(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.season = source["season"];
	        this.label = source["label"];
	    }
	}
	export class RawBracketChange {
	    season: number;
	    started?: boolean;
	    preset?: string;
	    node?: string;
	    slot?: number;
	    game?: string;
	    team?: string;
	    name?: string;
	    shape?: BracketNodeSpec[];
	
	    static createFrom(source: any = {}) {
	        return new RawBracketChange(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.season = source["season"];
	        this.started = source["started"];
	        this.preset = source["preset"];
	        this.node = source["node"];
	        this.slot = source["slot"];
	        this.game = source["game"];
	        this.team = source["team"];
	        this.name = source["name"];
	        this.shape = this.convertValues(source["shape"], BracketNodeSpec);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class RawRolloverArgs {
	    to: number;
	    dry_run: boolean;
	
	    static createFrom(source: any = {}) {
	        return new RawRolloverArgs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.to = source["to"];
	        this.dry_run = source["dry_run"];
	    }
	}
	export class RawLineupChange {
	    season: number;
	    franchise: string;
	    week: number;
	    starters: string[];
	
	    static createFrom(source: any = {}) {
	        return new RawLineupChange(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.season = source["season"];
	        this.franchise = source["franchise"];
	        this.week = source["week"];
	        this.starters = source["starters"];
	    }
	}
	export class RawStatChange {
	    item?: string;
	    player?: string;
	    season?: number;
	    week?: number;
	    stat?: string;
	    value?: number;
	    source?: string;
	
	    static createFrom(source: any = {}) {
	        return new RawStatChange(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.item = source["item"];
	        this.player = source["player"];
	        this.season = source["season"];
	        this.week = source["week"];
	        this.stat = source["stat"];
	        this.value = source["value"];
	        this.source = source["source"];
	    }
	}
	export class RawStandingsChange {
	    season: number;
	    franchise?: string;
	    wins?: number;
	    losses?: number;
	    ties?: number;
	    order?: string[];
	
	    static createFrom(source: any = {}) {
	        return new RawStandingsChange(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.season = source["season"];
	        this.franchise = source["franchise"];
	        this.wins = source["wins"];
	        this.losses = source["losses"];
	        this.ties = source["ties"];
	        this.order = source["order"];
	    }
	}
	export class RawScoreChange {
	    player?: string;
	    position?: string;
	    franchise?: string;
	    season?: number;
	    week?: number;
	    delta?: number;
	
	    static createFrom(source: any = {}) {
	        return new RawScoreChange(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.player = source["player"];
	        this.position = source["position"];
	        this.franchise = source["franchise"];
	        this.season = source["season"];
	        this.week = source["week"];
	        this.delta = source["delta"];
	    }
	}
	export class RawNativeOverride {
	    player?: string;
	    franchise?: string;
	    years?: number;
	    salaryCents?: number;
	    amountCents?: number;
	    season?: number;
	    bucket?: string;
	    until?: string;
	
	    static createFrom(source: any = {}) {
	        return new RawNativeOverride(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.player = source["player"];
	        this.franchise = source["franchise"];
	        this.years = source["years"];
	        this.salaryCents = source["salaryCents"];
	        this.amountCents = source["amountCents"];
	        this.season = source["season"];
	        this.bucket = source["bucket"];
	        this.until = source["until"];
	    }
	}
	export class RawDate {
	    month: number;
	    day: number;
	
	    static createFrom(source: any = {}) {
	        return new RawDate(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.month = source["month"];
	        this.day = source["day"];
	    }
	}
	export class RawWindowArgs {
	    id: string;
	    start?: RawDate;
	    end?: RawDate;
	    opens?: string[];
	
	    static createFrom(source: any = {}) {
	        return new RawWindowArgs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.start = this.convertValues(source["start"], RawDate);
	        this.end = this.convertValues(source["end"], RawDate);
	        this.opens = source["opens"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class RawScheduleArgs {
	    weeks?: number;
	    pattern?: string;
	    week?: number;
	    games?: competition.ScheduleGame[];
	    published?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new RawScheduleArgs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.weeks = source["weeks"];
	        this.pattern = source["pattern"];
	        this.week = source["week"];
	        this.games = this.convertValues(source["games"], competition.ScheduleGame);
	        this.published = source["published"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class RawWindowException {
	    action: string;
	    franchise: string;
	    until: time.Time;
	
	    static createFrom(source: any = {}) {
	        return new RawWindowException(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.action = source["action"];
	        this.franchise = source["franchise"];
	        this.until = this.convertValues(source["until"], time.Time);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SeedValue {
	    field: string;
	    value: number[];
	
	    static createFrom(source: any = {}) {
	        return new SeedValue(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.field = source["field"];
	        this.value = source["value"];
	    }
	}
	export class RawNote {
	    text: string;
	    pin?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new RawNote(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.text = source["text"];
	        this.pin = source["pin"];
	    }
	}
	export class RawPeople {
	    order?: string[];
	    franchise?: string;
	    principal?: string;
	    name?: string;
	    abbreviation?: string;
	    id?: string;
	    role?: string;
	    ability?: string;
	    on?: boolean;
	    all?: boolean;
	    report?: string;
	    audience?: string;
	    successor?: string;
	    commissioner?: string;
	    choice?: string;
	    inSeasonDays?: number;
	    offseasonDays?: number;
	    electionDays?: number;
	    runoffDays?: number;
	
	    static createFrom(source: any = {}) {
	        return new RawPeople(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.order = source["order"];
	        this.franchise = source["franchise"];
	        this.principal = source["principal"];
	        this.name = source["name"];
	        this.abbreviation = source["abbreviation"];
	        this.id = source["id"];
	        this.role = source["role"];
	        this.ability = source["ability"];
	        this.on = source["on"];
	        this.all = source["all"];
	        this.report = source["report"];
	        this.audience = source["audience"];
	        this.successor = source["successor"];
	        this.commissioner = source["commissioner"];
	        this.choice = source["choice"];
	        this.inSeasonDays = source["inSeasonDays"];
	        this.offseasonDays = source["offseasonDays"];
	        this.electionDays = source["electionDays"];
	        this.runoffDays = source["runoffDays"];
	    }
	}
	export class RawBidArgs {
	    player?: string;
	    franchise?: string;
	    bid?: string;
	    salaryCents?: number;
	    years?: number;
	
	    static createFrom(source: any = {}) {
	        return new RawBidArgs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.player = source["player"];
	        this.franchise = source["franchise"];
	        this.bid = source["bid"];
	        this.salaryCents = source["salaryCents"];
	        this.years = source["years"];
	    }
	}
	export class WaiverClaimResult {
	    id: string;
	    refusal?: string;
	
	    static createFrom(source: any = {}) {
	        return new WaiverClaimResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.refusal = source["refusal"];
	    }
	}
	export class WaiverOutcome {
	    claims?: WaiverClaimResult[];
	    priorOrder?: string[];
	    closeAt?: time.Time;
	    player?: NativePlayer;
	    deadCents?: number;
	    season: number;
	    winner?: string;
	    order?: string[];
	    source?: string;
	    clearsDead: boolean;
	
	    static createFrom(source: any = {}) {
	        return new WaiverOutcome(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.claims = this.convertValues(source["claims"], WaiverClaimResult);
	        this.priorOrder = source["priorOrder"];
	        this.closeAt = this.convertValues(source["closeAt"], time.Time);
	        this.player = this.convertValues(source["player"], NativePlayer);
	        this.deadCents = source["deadCents"];
	        this.season = source["season"];
	        this.winner = source["winner"];
	        this.order = source["order"];
	        this.source = source["source"];
	        this.clearsDead = source["clearsDead"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class RawWaiverArgs {
	    bucket?: string;
	    player?: string;
	    franchise?: string;
	    window?: string;
	    claim?: string;
	    order?: string[];
	
	    static createFrom(source: any = {}) {
	        return new RawWaiverArgs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.bucket = source["bucket"];
	        this.player = source["player"];
	        this.franchise = source["franchise"];
	        this.window = source["window"];
	        this.claim = source["claim"];
	        this.order = source["order"];
	    }
	}
	export class RawDraftAct {
	    round?: number;
	    original?: string;
	    extra?: number;
	    player?: string;
	    replaces?: string;
	    franchise?: string;
	    timer?: string;
	    seconds?: number;
	    to?: string;
	
	    static createFrom(source: any = {}) {
	        return new RawDraftAct(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.round = source["round"];
	        this.original = source["original"];
	        this.extra = source["extra"];
	        this.player = source["player"];
	        this.replaces = source["replaces"];
	        this.franchise = source["franchise"];
	        this.timer = source["timer"];
	        this.seconds = source["seconds"];
	        this.to = source["to"];
	    }
	}
	export class RawDraftSelection {
	    franchise: string;
	    player: string;
	    entrySeason: number;
	    round: number;
	    slot: number;
	    original: string;
	    terms: rookiedraft.Terms;
	
	    static createFrom(source: any = {}) {
	        return new RawDraftSelection(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.franchise = source["franchise"];
	        this.player = source["player"];
	        this.entrySeason = source["entrySeason"];
	        this.round = source["round"];
	        this.slot = source["slot"];
	        this.original = source["original"];
	        this.terms = this.convertValues(source["terms"], rookiedraft.Terms);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class RawDraftArgs {
	    season?: number;
	    round?: number;
	    order?: string[];
	    source?: string;
	    pattern?: string;
	    tie?: string;
	    at?: time.Time;
	    pick?: RawDraftSelection;
	    act?: RawDraftAct;
	    running?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new RawDraftArgs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.season = source["season"];
	        this.round = source["round"];
	        this.order = source["order"];
	        this.source = source["source"];
	        this.pattern = source["pattern"];
	        this.tie = source["tie"];
	        this.at = this.convertValues(source["at"], time.Time);
	        this.pick = this.convertValues(source["pick"], RawDraftSelection);
	        this.act = this.convertValues(source["act"], RawDraftAct);
	        this.running = source["running"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class TradeVote {
	    who: string;
	    decision: string;
	    at: time.Time;
	    reason?: string;
	
	    static createFrom(source: any = {}) {
	        return new TradeVote(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.who = source["who"];
	        this.decision = source["decision"];
	        this.at = this.convertValues(source["at"], time.Time);
	        this.reason = source["reason"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class TradePolicy {
	    route: string;
	    seats: number;
	    votesToDecide: number;
	    futureYears: number;
	    minimumRound: number;
	    maximumRound: number;
	    refuseIllegalRoster: boolean;
	    offerLifetimeNanoseconds: number;
	    maximumParties: number;
	
	    static createFrom(source: any = {}) {
	        return new TradePolicy(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.route = source["route"];
	        this.seats = source["seats"];
	        this.votesToDecide = source["votesToDecide"];
	        this.futureYears = source["futureYears"];
	        this.minimumRound = source["minimumRound"];
	        this.maximumRound = source["maximumRound"];
	        this.refuseIllegalRoster = source["refuseIllegalRoster"];
	        this.offerLifetimeNanoseconds = source["offerLifetimeNanoseconds"];
	        this.maximumParties = source["maximumParties"];
	    }
	}
	export class TradeAssetDetail {
	    fromTeam: string;
	    toTeam: string;
	    label: string;
	    contract?: string;
	    yearlySalaryCents?: number;
	    contractEndsInSeason?: number;
	
	    static createFrom(source: any = {}) {
	        return new TradeAssetDetail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.fromTeam = source["fromTeam"];
	        this.toTeam = source["toTeam"];
	        this.label = source["label"];
	        this.contract = source["contract"];
	        this.yearlySalaryCents = source["yearlySalaryCents"];
	        this.contractEndsInSeason = source["contractEndsInSeason"];
	    }
	}
	export class TradeState {
	    vetoNumber: number;
	    appealedVetoNumber: number;
	    assetDetails: TradeAssetDetail[];
	    previous?: string;
	    id: string;
	    proposer: string;
	    parties: string[];
	    assets: TradeAsset[];
	    accepted: string[];
	    policy: TradePolicy;
	    offeredAt: time.Time;
	    expiresAt: time.Time;
	    status: string;
	    words: string;
	    votes: TradeVote[];
	    recusals: TradeVote[];
	    appeals: TradeVote[];
	    overrides: TradeVote[];
	
	    static createFrom(source: any = {}) {
	        return new TradeState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.vetoNumber = source["vetoNumber"];
	        this.appealedVetoNumber = source["appealedVetoNumber"];
	        this.assetDetails = this.convertValues(source["assetDetails"], TradeAssetDetail);
	        this.previous = source["previous"];
	        this.id = source["id"];
	        this.proposer = source["proposer"];
	        this.parties = source["parties"];
	        this.assets = this.convertValues(source["assets"], TradeAsset);
	        this.accepted = source["accepted"];
	        this.policy = this.convertValues(source["policy"], TradePolicy);
	        this.offeredAt = this.convertValues(source["offeredAt"], time.Time);
	        this.expiresAt = this.convertValues(source["expiresAt"], time.Time);
	        this.status = source["status"];
	        this.words = source["words"];
	        this.votes = this.convertValues(source["votes"], TradeVote);
	        this.recusals = this.convertValues(source["recusals"], TradeVote);
	        this.appeals = this.convertValues(source["appeals"], TradeVote);
	        this.overrides = this.convertValues(source["overrides"], TradeVote);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class TradeAsset {
	    from: string;
	    to: string;
	    player?: string;
	    season?: number;
	    round?: number;
	    original?: string;
	
	    static createFrom(source: any = {}) {
	        return new TradeAsset(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.from = source["from"];
	        this.to = source["to"];
	        this.player = source["player"];
	        this.season = source["season"];
	        this.round = source["round"];
	        this.original = source["original"];
	    }
	}
	export class RawTradeArgs {
	    id?: string;
	    franchise?: string;
	    parties?: string[];
	    assets?: TradeAsset[];
	    decision?: string;
	
	    static createFrom(source: any = {}) {
	        return new RawTradeArgs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.franchise = source["franchise"];
	        this.parties = source["parties"];
	        this.assets = this.convertValues(source["assets"], TradeAsset);
	        this.decision = source["decision"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class RawCorrection {
	    franchise?: string;
	    player?: string;
	    action?: string;
	    entry?: string;
	    via?: string;
	    pick?: CorrectionPick;
	    owner?: string;
	    rule?: string;
	    scope?: ExceptionScope;
	    until?: string;
	
	    static createFrom(source: any = {}) {
	        return new RawCorrection(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.franchise = source["franchise"];
	        this.player = source["player"];
	        this.action = source["action"];
	        this.entry = source["entry"];
	        this.via = source["via"];
	        this.pick = this.convertValues(source["pick"], CorrectionPick);
	        this.owner = source["owner"];
	        this.rule = source["rule"];
	        this.scope = this.convertValues(source["scope"], ExceptionScope);
	        this.until = source["until"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class RawArgs {
	    correction?: RawCorrection;
	    trade?: RawTradeArgs;
	    tradeOutcome?: TradeState;
	    draft?: RawDraftArgs;
	    waiver?: RawWaiverArgs;
	    waiverOutcome?: WaiverOutcome;
	    bid?: RawBidArgs;
	    bidOutcome?: BidOutcome;
	    draftOutcome?: DraftOutcome;
	    name?: string;
	    entry?: string;
	    franchises?: string[];
	    people?: RawPeople;
	    note?: RawNote;
	    field?: string;
	    value?: number[];
	    preset?: string;
	    franchiseCount?: number;
	    seed?: SeedValue[];
	    native?: RawNativeSnapshot;
	    exception?: RawWindowException;
	    schedule?: RawScheduleArgs;
	    window?: RawWindowArgs;
	    override?: RawNativeOverride;
	    score?: RawScoreChange;
	    standings?: RawStandingsChange;
	    stat?: RawStatChange;
	    lineup?: RawLineupChange;
	    rollover?: RawRolloverArgs;
	    bracket?: RawBracketChange;
	    title?: RawTitleChange;
	    link?: RawHistoryLink;
	    season?: number;
	    contractAction?: RawContractAction;
	
	    static createFrom(source: any = {}) {
	        return new RawArgs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.correction = this.convertValues(source["correction"], RawCorrection);
	        this.trade = this.convertValues(source["trade"], RawTradeArgs);
	        this.tradeOutcome = this.convertValues(source["tradeOutcome"], TradeState);
	        this.draft = this.convertValues(source["draft"], RawDraftArgs);
	        this.waiver = this.convertValues(source["waiver"], RawWaiverArgs);
	        this.waiverOutcome = this.convertValues(source["waiverOutcome"], WaiverOutcome);
	        this.bid = this.convertValues(source["bid"], RawBidArgs);
	        this.bidOutcome = this.convertValues(source["bidOutcome"], BidOutcome);
	        this.draftOutcome = this.convertValues(source["draftOutcome"], DraftOutcome);
	        this.name = source["name"];
	        this.entry = source["entry"];
	        this.franchises = source["franchises"];
	        this.people = this.convertValues(source["people"], RawPeople);
	        this.note = this.convertValues(source["note"], RawNote);
	        this.field = source["field"];
	        this.value = source["value"];
	        this.preset = source["preset"];
	        this.franchiseCount = source["franchiseCount"];
	        this.seed = this.convertValues(source["seed"], SeedValue);
	        this.native = this.convertValues(source["native"], RawNativeSnapshot);
	        this.exception = this.convertValues(source["exception"], RawWindowException);
	        this.schedule = this.convertValues(source["schedule"], RawScheduleArgs);
	        this.window = this.convertValues(source["window"], RawWindowArgs);
	        this.override = this.convertValues(source["override"], RawNativeOverride);
	        this.score = this.convertValues(source["score"], RawScoreChange);
	        this.standings = this.convertValues(source["standings"], RawStandingsChange);
	        this.stat = this.convertValues(source["stat"], RawStatChange);
	        this.lineup = this.convertValues(source["lineup"], RawLineupChange);
	        this.rollover = this.convertValues(source["rollover"], RawRolloverArgs);
	        this.bracket = this.convertValues(source["bracket"], RawBracketChange);
	        this.title = this.convertValues(source["title"], RawTitleChange);
	        this.link = this.convertValues(source["link"], RawHistoryLink);
	        this.season = source["season"];
	        this.contractAction = this.convertValues(source["contractAction"], RawContractAction);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	export class RawCommand {
	    league: string;
	    command: string;
	    args: RawArgs;
	    reason: string;
	    expected: number;
	    request: string;
	    effective: string;
	    visibility: string;
	    gravity?: string;
	
	    static createFrom(source: any = {}) {
	        return new RawCommand(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.league = source["league"];
	        this.command = source["command"];
	        this.args = this.convertValues(source["args"], RawArgs);
	        this.reason = source["reason"];
	        this.expected = source["expected"];
	        this.request = source["request"];
	        this.effective = source["effective"];
	        this.visibility = source["visibility"];
	        this.gravity = source["gravity"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	
	
	
	
	
	export class RawLeagueCalendarScope {
	    league: string;
	    season: number;
	    localZone: string;
	    franchise: string;
	    action: string;
	
	    static createFrom(source: any = {}) {
	        return new RawLeagueCalendarScope(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.league = source["league"];
	        this.season = source["season"];
	        this.localZone = source["localZone"];
	        this.franchise = source["franchise"];
	        this.action = source["action"];
	    }
	}
	export class RawLeagueScheduleScope {
	    league: string;
	
	    static createFrom(source: any = {}) {
	        return new RawLeagueScheduleScope(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.league = source["league"];
	    }
	}
	
	export class RawNativeDraftScope {
	    league: string;
	    season: number;
	    franchise: string;
	    at: time.Time;
	
	    static createFrom(source: any = {}) {
	        return new RawNativeDraftScope(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.league = source["league"];
	        this.season = source["season"];
	        this.franchise = source["franchise"];
	        this.at = this.convertValues(source["at"], time.Time);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class RawNativeLineup {
	    league: string;
	    franchise: string;
	    submitted: boolean;
	    starters: string[];
	
	    static createFrom(source: any = {}) {
	        return new RawNativeLineup(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.league = source["league"];
	        this.franchise = source["franchise"];
	        this.submitted = source["submitted"];
	        this.starters = source["starters"];
	    }
	}
	
	export class RawNativeScheduleScope {
	    league: string;
	    franchise: string;
	    season: number;
	    localZone: string;
	
	    static createFrom(source: any = {}) {
	        return new RawNativeScheduleScope(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.league = source["league"];
	        this.franchise = source["franchise"];
	        this.season = source["season"];
	        this.localZone = source["localZone"];
	    }
	}
	export class RawNativeSeedRequest {
	    league: string;
	    expected: number;
	    request: string;
	    reason: string;
	
	    static createFrom(source: any = {}) {
	        return new RawNativeSeedRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.league = source["league"];
	        this.expected = source["expected"];
	        this.request = source["request"];
	        this.reason = source["reason"];
	    }
	}
	
	export class RawNativeWeekScope {
	    league: string;
	    franchise: string;
	    season: number;
	    localZone: string;
	    at: time.Time;
	
	    static createFrom(source: any = {}) {
	        return new RawNativeWeekScope(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.league = source["league"];
	        this.franchise = source["franchise"];
	        this.season = source["season"];
	        this.localZone = source["localZone"];
	        this.at = this.convertValues(source["at"], time.Time);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	
	
	
	
	
	
	
	
	
	
	export class RolloverPlan {
	    from: number;
	    to: number;
	    beforeDigest: string;
	    blocked: boolean;
	    reasons: string[];
	    lines: string[];
	    after: RawNativeSnapshot;
	    pending: PendingExpiry[];
	    rules: leaguerules.RuleSet;
	    calendar: leagueclock.ActionCalendar;
	
	    static createFrom(source: any = {}) {
	        return new RolloverPlan(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.from = source["from"];
	        this.to = source["to"];
	        this.beforeDigest = source["beforeDigest"];
	        this.blocked = source["blocked"];
	        this.reasons = source["reasons"];
	        this.lines = source["lines"];
	        this.after = this.convertValues(source["after"], RawNativeSnapshot);
	        this.pending = this.convertValues(source["pending"], PendingExpiry);
	        this.rules = this.convertValues(source["rules"], leaguerules.RuleSet);
	        this.calendar = this.convertValues(source["calendar"], leagueclock.ActionCalendar);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Receipt {
	    entry: string;
	    league: string;
	    revision: number;
	    recordedAt: time.Time;
	    message: string;
	    plan?: RolloverPlan;
	
	    static createFrom(source: any = {}) {
	        return new Receipt(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.entry = source["entry"];
	        this.league = source["league"];
	        this.revision = source["revision"];
	        this.recordedAt = this.convertValues(source["recordedAt"], time.Time);
	        this.message = source["message"];
	        this.plan = this.convertValues(source["plan"], RolloverPlan);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class ResultFlip {
	    status: string;
	    words: string;
	    flips: FlippedGame[];
	
	    static createFrom(source: any = {}) {
	        return new ResultFlip(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.words = source["words"];
	        this.flips = this.convertValues(source["flips"], FlippedGame);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	export class RulePending {
	    entry: string;
	    value: string;
	    when: string;
	    replacedAt?: string;
	
	    static createFrom(source: any = {}) {
	        return new RulePending(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.entry = source["entry"];
	        this.value = source["value"];
	        this.when = source["when"];
	        this.replacedAt = source["replacedAt"];
	    }
	}
	export class RuleSafeguards {
	    nextRollover: string;
	    now: string;
	
	    static createFrom(source: any = {}) {
	        return new RuleSafeguards(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.nextRollover = source["nextRollover"];
	        this.now = source["now"];
	    }
	}
	export class RuleRow {
	    field: string;
	    label: string;
	    value: string;
	    state: string;
	    citation: string;
	    changedAt: string;
	    schema: leaguerules.ValueSchema;
	    competitive: boolean;
	    pending: RulePending[];
	    current: number[];
	    unit: string;
	    safeguards: RuleSafeguards;
	    governed?: Governed;
	
	    static createFrom(source: any = {}) {
	        return new RuleRow(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.field = source["field"];
	        this.label = source["label"];
	        this.value = source["value"];
	        this.state = source["state"];
	        this.citation = source["citation"];
	        this.changedAt = source["changedAt"];
	        this.schema = this.convertValues(source["schema"], leaguerules.ValueSchema);
	        this.competitive = source["competitive"];
	        this.pending = this.convertValues(source["pending"], RulePending);
	        this.current = source["current"];
	        this.unit = source["unit"];
	        this.safeguards = this.convertValues(source["safeguards"], RuleSafeguards);
	        this.governed = this.convertValues(source["governed"], Governed);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class RuleSection {
	    label: string;
	    rows: RuleRow[];
	
	    static createFrom(source: any = {}) {
	        return new RuleSection(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.label = source["label"];
	        this.rows = this.convertValues(source["rows"], RuleRow);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Rules {
	    league: string;
	    name: string;
	    revision: number;
	    rollover: string;
	    sections: RuleSection[];
	
	    static createFrom(source: any = {}) {
	        return new Rules(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.league = source["league"];
	        this.name = source["name"];
	        this.revision = source["revision"];
	        this.rollover = source["rollover"];
	        this.sections = this.convertValues(source["sections"], RuleSection);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Schedule {
	    league: string;
	    revision: number;
	    season: number;
	    weeks: competition.ScheduleWeek[];
	
	    static createFrom(source: any = {}) {
	        return new Schedule(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.league = source["league"];
	        this.revision = source["revision"];
	        this.season = source["season"];
	        this.weeks = this.convertValues(source["weeks"], competition.ScheduleWeek);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class ScorePreview {
	    command: string;
	    words: string;
	    warnings: string[];
	    flip: ResultFlip;
	    lineup?: nativecheck.LineupVerdict;
	
	    static createFrom(source: any = {}) {
	        return new ScorePreview(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.command = source["command"];
	        this.words = source["words"];
	        this.warnings = source["warnings"];
	        this.flip = this.convertValues(source["flip"], ResultFlip);
	        this.lineup = this.convertValues(source["lineup"], nativecheck.LineupVerdict);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	
	
	export class StatChange {
	    stat: string;
	    label: string;
	    before: string;
	    after: string;
	
	    static createFrom(source: any = {}) {
	        return new StatChange(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.stat = source["stat"];
	        this.label = source["label"];
	        this.before = source["before"];
	        this.after = source["after"];
	    }
	}
	export class StatDecision {
	    entry: string;
	    kind: string;
	    status: string;
	    player: string;
	    name: string;
	    season: number;
	    week: number;
	    words: string;
	    by: string;
	    at: time.Time;
	    reason: string;
	
	    static createFrom(source: any = {}) {
	        return new StatDecision(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.entry = source["entry"];
	        this.kind = source["kind"];
	        this.status = source["status"];
	        this.player = source["player"];
	        this.name = source["name"];
	        this.season = source["season"];
	        this.week = source["week"];
	        this.words = source["words"];
	        this.by = source["by"];
	        this.at = this.convertValues(source["at"], time.Time);
	        this.reason = source["reason"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class StatQueueItem {
	    id: string;
	    player: string;
	    name: string;
	    season: number;
	    week: number;
	    changes: StatChange[];
	    points?: number;
	    flip: string;
	    flipWords: string;
	    words: string;
	    arrived: time.Time;
	    sha256: string;
	
	    static createFrom(source: any = {}) {
	        return new StatQueueItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.player = source["player"];
	        this.name = source["name"];
	        this.season = source["season"];
	        this.week = source["week"];
	        this.changes = this.convertValues(source["changes"], StatChange);
	        this.points = source["points"];
	        this.flip = source["flip"];
	        this.flipWords = source["flipWords"];
	        this.words = source["words"];
	        this.arrived = this.convertValues(source["arrived"], time.Time);
	        this.sha256 = source["sha256"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class StatQueue {
	    league: string;
	    season: number;
	    revision: number;
	    words: string;
	    open: StatQueueItem[];
	    decided: StatDecision[];
	
	    static createFrom(source: any = {}) {
	        return new StatQueue(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.league = source["league"];
	        this.season = source["season"];
	        this.revision = source["revision"];
	        this.words = source["words"];
	        this.open = this.convertValues(source["open"], StatQueueItem);
	        this.decided = this.convertValues(source["decided"], StatDecision);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	
	
	
	
	

}

export namespace comms {
	
	export class CommsEvent {
	    sentence: string;
	    date: string;
	
	    static createFrom(source: any = {}) {
	        return new CommsEvent(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sentence = source["sentence"];
	        this.date = source["date"];
	    }
	}
	export class CommsBoard {
	    feed: report.Report;
	    transactions: report.Report;
	    events: CommsEvent[];
	    boardNote: string;
	    leagueTalk: string;
	    boardPosts: report.Report[];
	    team: string;
	    kind: string;
	    teams: report.Choice[];
	    kinds: report.Choice[];
	
	    static createFrom(source: any = {}) {
	        return new CommsBoard(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.feed = this.convertValues(source["feed"], report.Report);
	        this.transactions = this.convertValues(source["transactions"], report.Report);
	        this.events = this.convertValues(source["events"], CommsEvent);
	        this.boardNote = source["boardNote"];
	        this.leagueTalk = source["leagueTalk"];
	        this.boardPosts = this.convertValues(source["boardPosts"], report.Report);
	        this.team = source["team"];
	        this.kind = source["kind"];
	        this.teams = this.convertValues(source["teams"], report.Choice);
	        this.kinds = this.convertValues(source["kinds"], report.Choice);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class RawCommsScope {
	    team: string;
	    kind: string;
	
	    static createFrom(source: any = {}) {
	        return new RawCommsScope(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.team = source["team"];
	        this.kind = source["kind"];
	    }
	}

}

export namespace competition {
	
	export class BracketSide {
	    franchise: string;
	    name: string;
	    seed: number;
	    rank: number;
	    byHand: boolean;
	
	    static createFrom(source: any = {}) {
	        return new BracketSide(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.franchise = source["franchise"];
	        this.name = source["name"];
	        this.seed = source["seed"];
	        this.rank = source["rank"];
	        this.byHand = source["byHand"];
	    }
	}
	export class BracketGame {
	    id: string;
	    node: string;
	    conference?: string;
	    week: number;
	    homeSlot: string;
	    awaySlot: string;
	    home?: BracketSide;
	    away?: BracketSide;
	    status: string;
	    homePoints?: number;
	    awayPoints?: number;
	    homeBonus: number;
	    homeTotal?: number;
	    awayTotal?: number;
	    winner?: string;
	    byHand: boolean;
	    words: string;
	
	    static createFrom(source: any = {}) {
	        return new BracketGame(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.node = source["node"];
	        this.conference = source["conference"];
	        this.week = source["week"];
	        this.homeSlot = source["homeSlot"];
	        this.awaySlot = source["awaySlot"];
	        this.home = this.convertValues(source["home"], BracketSide);
	        this.away = this.convertValues(source["away"], BracketSide);
	        this.status = source["status"];
	        this.homePoints = source["homePoints"];
	        this.awayPoints = source["awayPoints"];
	        this.homeBonus = source["homeBonus"];
	        this.homeTotal = source["homeTotal"];
	        this.awayTotal = source["awayTotal"];
	        this.winner = source["winner"];
	        this.byHand = source["byHand"];
	        this.words = source["words"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class BracketRound {
	    round: number;
	    week: number;
	    name: string;
	    games: BracketGame[];
	
	    static createFrom(source: any = {}) {
	        return new BracketRound(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.round = source["round"];
	        this.week = source["week"];
	        this.name = source["name"];
	        this.games = this.convertValues(source["games"], BracketGame);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class FieldSeed {
	    seed: number;
	    franchise: string;
	    name: string;
	    division: string;
	    rank: number;
	    kind: string;
	    bye: boolean;
	    byHand: boolean;
	
	    static createFrom(source: any = {}) {
	        return new FieldSeed(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.seed = source["seed"];
	        this.franchise = source["franchise"];
	        this.name = source["name"];
	        this.division = source["division"];
	        this.rank = source["rank"];
	        this.kind = source["kind"];
	        this.bye = source["bye"];
	        this.byHand = source["byHand"];
	    }
	}
	export class ConferenceField {
	    conference: string;
	    seeds: FieldSeed[];
	
	    static createFrom(source: any = {}) {
	        return new ConferenceField(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.conference = source["conference"];
	        this.seeds = this.convertValues(source["seeds"], FieldSeed);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class ScheduleGame {
	    id: string;
	    home: string;
	    away: string;
	
	    static createFrom(source: any = {}) {
	        return new ScheduleGame(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.home = source["home"];
	        this.away = source["away"];
	    }
	}
	export class ScheduleWeek {
	    week: number;
	    games: ScheduleGame[];
	    bye?: string;
	
	    static createFrom(source: any = {}) {
	        return new ScheduleWeek(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.week = source["week"];
	        this.games = this.convertValues(source["games"], ScheduleGame);
	        this.bye = source["bye"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace contracts {
	
	export class WhatIfFigure {
	    label: string;
	    before: string;
	    after: string;
	    change: string;
	
	    static createFrom(source: any = {}) {
	        return new WhatIfFigure(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.label = source["label"];
	        this.before = source["before"];
	        this.after = source["after"];
	        this.change = source["change"];
	    }
	}
	export class AcquisitionCell {
	    text: string;
	    sub: string;
	    detail: string;
	    held: boolean;
	
	    static createFrom(source: any = {}) {
	        return new AcquisitionCell(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.text = source["text"];
	        this.sub = source["sub"];
	        this.detail = source["detail"];
	        this.held = source["held"];
	    }
	}
	export class AcquisitionAsset {
	    team?: string;
	    contract?: string;
	    // Go type: AcquisitionCell
	    now?: any;
	    // Go type: AcquisitionCell
	    dynasty?: any;
	    token: string;
	    name: string;
	    position?: string;
	
	    static createFrom(source: any = {}) {
	        return new AcquisitionAsset(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.team = source["team"];
	        this.contract = source["contract"];
	        this.now = this.convertValues(source["now"], null);
	        this.dynasty = this.convertValues(source["dynasty"], null);
	        this.token = source["token"];
	        this.name = source["name"];
	        this.position = source["position"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class AcquisitionChoice {
	    player: string;
	    bid: string;
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new AcquisitionChoice(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.player = source["player"];
	        this.bid = source["bid"];
	        this.name = source["name"];
	    }
	}
	export class AcquisitionView {
	    incoming: AcquisitionChoice[];
	    cuts: AcquisitionAsset[];
	    player: string;
	    seasons: WhatIfFigure[];
	    team: WhatIfFigure[];
	    deadCap: snapshot.ContractAmount;
	    cutName: string;
	    spot: string;
	    reasons: string[];
	
	    static createFrom(source: any = {}) {
	        return new AcquisitionView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.incoming = this.convertValues(source["incoming"], AcquisitionChoice);
	        this.cuts = this.convertValues(source["cuts"], AcquisitionAsset);
	        this.player = source["player"];
	        this.seasons = this.convertValues(source["seasons"], WhatIfFigure);
	        this.team = this.convertValues(source["team"], WhatIfFigure);
	        this.deadCap = this.convertValues(source["deadCap"], snapshot.ContractAmount);
	        this.cutName = source["cutName"];
	        this.spot = source["spot"];
	        this.reasons = source["reasons"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class AcquisitionReading {
	    // Go type: AcquisitionView
	    value: any;
	    provenance: snapshot.Provenance;
	    status: string;
	    absenceReason?: string;
	    count: number;
	    threshold: number;
	    baseline: string;
	    cutoff: string;
	    method: string;
	    inputSHA: string;
	    buildStamp: string;
	    howCounted: string;
	
	    static createFrom(source: any = {}) {
	        return new AcquisitionReading(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.value = this.convertValues(source["value"], null);
	        this.provenance = this.convertValues(source["provenance"], snapshot.Provenance);
	        this.status = source["status"];
	        this.absenceReason = source["absenceReason"];
	        this.count = source["count"];
	        this.threshold = source["threshold"];
	        this.baseline = source["baseline"];
	        this.cutoff = source["cutoff"];
	        this.method = source["method"];
	        this.inputSHA = source["inputSHA"];
	        this.buildStamp = source["buildStamp"];
	        this.howCounted = source["howCounted"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ContractActFigure {
	    season: number;
	    before: string;
	    after: string;
	    change: string;
	
	    static createFrom(source: any = {}) {
	        return new ContractActFigure(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.season = source["season"];
	        this.before = source["before"];
	        this.after = source["after"];
	        this.change = source["change"];
	    }
	}
	export class ContractActChoice {
	    value: number;
	    label: string;
	
	    static createFrom(source: any = {}) {
	        return new ContractActChoice(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.value = source["value"];
	        this.label = source["label"];
	    }
	}
	export class ContractActView {
	    id: string;
	    intent: string;
	    act: string;
	    levels: ContractActChoice[];
	    years: number[];
	    teamFigures: WhatIfFigure[];
	    player: string;
	    subject: string;
	    label: string;
	    price: string;
	    post: string;
	    how: string;
	    reasons: string[];
	    figures: ContractActFigure[];
	    receipt: envelope.Receipt;
	    canHandOff: boolean;
	    canPost: boolean;
	    canWithdraw: boolean;
	    canRebase: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ContractActView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.intent = source["intent"];
	        this.act = source["act"];
	        this.levels = this.convertValues(source["levels"], ContractActChoice);
	        this.years = source["years"];
	        this.teamFigures = this.convertValues(source["teamFigures"], WhatIfFigure);
	        this.player = source["player"];
	        this.subject = source["subject"];
	        this.label = source["label"];
	        this.price = source["price"];
	        this.post = source["post"];
	        this.how = source["how"];
	        this.reasons = source["reasons"];
	        this.figures = this.convertValues(source["figures"], ContractActFigure);
	        this.receipt = this.convertValues(source["receipt"], envelope.Receipt);
	        this.canHandOff = source["canHandOff"];
	        this.canPost = source["canPost"];
	        this.canWithdraw = source["canWithdraw"];
	        this.canRebase = source["canRebase"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ContractActsReading {
	    value: ContractActView[];
	    provenance: snapshot.Provenance;
	    status: string;
	    absenceReason?: string;
	    count: number;
	    threshold: number;
	    baseline: string;
	    cutoff: string;
	    method: string;
	    inputSHA: string;
	    buildStamp: string;
	    howCounted: string;
	
	    static createFrom(source: any = {}) {
	        return new ContractActsReading(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.value = this.convertValues(source["value"], ContractActView);
	        this.provenance = this.convertValues(source["provenance"], snapshot.Provenance);
	        this.status = source["status"];
	        this.absenceReason = source["absenceReason"];
	        this.count = source["count"];
	        this.threshold = source["threshold"];
	        this.baseline = source["baseline"];
	        this.cutoff = source["cutoff"];
	        this.method = source["method"];
	        this.inputSHA = source["inputSHA"];
	        this.buildStamp = source["buildStamp"];
	        this.howCounted = source["howCounted"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ContractsReadings {
	    table: report.Reading_github_com_secureprospective_TheWarRoom_internal_snapshot_ContractTable_;
	    cap: report.Reading_github_com_secureprospective_TheWarRoom_internal_snapshot_CapSummary_;
	    adjustments: report.Reading_github_com_secureprospective_TheWarRoom_internal_snapshot_SalaryAdjustments_;
	
	    static createFrom(source: any = {}) {
	        return new ContractsReadings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.table = this.convertValues(source["table"], report.Reading_github_com_secureprospective_TheWarRoom_internal_snapshot_ContractTable_);
	        this.cap = this.convertValues(source["cap"], report.Reading_github_com_secureprospective_TheWarRoom_internal_snapshot_CapSummary_);
	        this.adjustments = this.convertValues(source["adjustments"], report.Reading_github_com_secureprospective_TheWarRoom_internal_snapshot_SalaryAdjustments_);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class LineupPlayer {
	    id: string;
	    position?: string;
	
	    static createFrom(source: any = {}) {
	        return new LineupPlayer(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.position = source["position"];
	    }
	}
	export class LineupReading {
	    franchise: string;
	    week: number;
	    starterCount: number;
	    starters: LineupPlayer[];
	    bench: LineupPlayer[];
	    check: lineup.Result;
	    provenance: snapshot.Provenance;
	    rulesSource: snapshot.Provenance;
	
	    static createFrom(source: any = {}) {
	        return new LineupReading(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.franchise = source["franchise"];
	        this.week = source["week"];
	        this.starterCount = source["starterCount"];
	        this.starters = this.convertValues(source["starters"], LineupPlayer);
	        this.bench = this.convertValues(source["bench"], LineupPlayer);
	        this.check = this.convertValues(source["check"], lineup.Result);
	        this.provenance = this.convertValues(source["provenance"], snapshot.Provenance);
	        this.rulesSource = this.convertValues(source["rulesSource"], snapshot.Provenance);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PlayerContractReading {
	    // Go type: snapshot
	    value: any;
	    provenance: snapshot.Provenance;
	    status: string;
	    absenceReason?: string;
	    count: number;
	    threshold: number;
	    baseline: string;
	    cutoff: string;
	    method: string;
	    inputSHA: string;
	    buildStamp: string;
	    howCounted: string;
	
	    static createFrom(source: any = {}) {
	        return new PlayerContractReading(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.value = this.convertValues(source["value"], null);
	        this.provenance = this.convertValues(source["provenance"], snapshot.Provenance);
	        this.status = source["status"];
	        this.absenceReason = source["absenceReason"];
	        this.count = source["count"];
	        this.threshold = source["threshold"];
	        this.baseline = source["baseline"];
	        this.cutoff = source["cutoff"];
	        this.method = source["method"];
	        this.inputSHA = source["inputSHA"];
	        this.buildStamp = source["buildStamp"];
	        this.howCounted = source["howCounted"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class RawAcquisition {
	    mine: string;
	    player: string;
	    bid: string;
	    cut: string;
	
	    static createFrom(source: any = {}) {
	        return new RawAcquisition(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mine = source["mine"];
	        this.player = source["player"];
	        this.bid = source["bid"];
	        this.cut = source["cut"];
	    }
	}
	export class RawContractAct {
	    mine: string;
	    player: string;
	    action: string;
	    intent: string;
	    id: string;
	    level: number;
	    years: number;
	    source: number;
	    destination: number;
	    move: string;
	
	    static createFrom(source: any = {}) {
	        return new RawContractAct(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mine = source["mine"];
	        this.player = source["player"];
	        this.action = source["action"];
	        this.intent = source["intent"];
	        this.id = source["id"];
	        this.level = source["level"];
	        this.years = source["years"];
	        this.source = source["source"];
	        this.destination = source["destination"];
	        this.move = source["move"];
	    }
	}
	export class RawReleaseRequest {
	    mine: string;
	    action: string;
	    move: string;
	    id: string;
	    confirmed: boolean;
	
	    static createFrom(source: any = {}) {
	        return new RawReleaseRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mine = source["mine"];
	        this.action = source["action"];
	        this.move = source["move"];
	        this.id = source["id"];
	        this.confirmed = source["confirmed"];
	    }
	}
	export class RawWhatIfRequest {
	    mine: string;
	    action: string;
	    player: string;
	    move: string;
	
	    static createFrom(source: any = {}) {
	        return new RawWhatIfRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mine = source["mine"];
	        this.action = source["action"];
	        this.player = source["player"];
	        this.move = source["move"];
	    }
	}
	export class ReleasePlanView {
	    id: string;
	    player: string;
	    label: string;
	    charge: string;
	    confirmation: string;
	    post: string;
	    channel: string;
	    receipt: envelope.Receipt;
	    canHandOff: boolean;
	    canRebase: boolean;
	    rebaseLabel: string;
	    withdrawLabel: string;
	    cut: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ReleasePlanView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.player = source["player"];
	        this.label = source["label"];
	        this.charge = source["charge"];
	        this.confirmation = source["confirmation"];
	        this.post = source["post"];
	        this.channel = source["channel"];
	        this.receipt = this.convertValues(source["receipt"], envelope.Receipt);
	        this.canHandOff = source["canHandOff"];
	        this.canRebase = source["canRebase"];
	        this.rebaseLabel = source["rebaseLabel"];
	        this.withdrawLabel = source["withdrawLabel"];
	        this.cut = source["cut"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ReleasePlansReading {
	    value: ReleasePlanView[];
	    provenance: snapshot.Provenance;
	    status: string;
	    absenceReason?: string;
	    count: number;
	    threshold: number;
	    baseline: string;
	    cutoff: string;
	    method: string;
	    inputSHA: string;
	    buildStamp: string;
	    howCounted: string;
	
	    static createFrom(source: any = {}) {
	        return new ReleasePlansReading(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.value = this.convertValues(source["value"], ReleasePlanView);
	        this.provenance = this.convertValues(source["provenance"], snapshot.Provenance);
	        this.status = source["status"];
	        this.absenceReason = source["absenceReason"];
	        this.count = source["count"];
	        this.threshold = source["threshold"];
	        this.baseline = source["baseline"];
	        this.cutoff = source["cutoff"];
	        this.method = source["method"];
	        this.inputSHA = source["inputSHA"];
	        this.buildStamp = source["buildStamp"];
	        this.howCounted = source["howCounted"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class WhatIfChoice {
	    player: string;
	    cutLabel: string;
	    buyoutLabel: string;
	    buyoutAllowed: boolean;
	    reason: string;
	
	    static createFrom(source: any = {}) {
	        return new WhatIfChoice(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.player = source["player"];
	        this.cutLabel = source["cutLabel"];
	        this.buyoutLabel = source["buyoutLabel"];
	        this.buyoutAllowed = source["buyoutAllowed"];
	        this.reason = source["reason"];
	    }
	}
	export class WhatIfMove {
	    id: string;
	    player: string;
	    tag: string;
	    status: string;
	    label: string;
	    charge: snapshot.ContractAmount;
	    explanation: string;
	    plan: string;
	
	    static createFrom(source: any = {}) {
	        return new WhatIfMove(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.player = source["player"];
	        this.tag = source["tag"];
	        this.status = source["status"];
	        this.label = source["label"];
	        this.charge = this.convertValues(source["charge"], snapshot.ContractAmount);
	        this.explanation = source["explanation"];
	        this.plan = source["plan"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class WhatIfView {
	    stale: boolean;
	    banner: string;
	    empty: string;
	    moves: WhatIfMove[];
	    figures: WhatIfFigure[];
	    choices: WhatIfChoice[];
	    notice: string;
	
	    static createFrom(source: any = {}) {
	        return new WhatIfView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.stale = source["stale"];
	        this.banner = source["banner"];
	        this.empty = source["empty"];
	        this.moves = this.convertValues(source["moves"], WhatIfMove);
	        this.figures = this.convertValues(source["figures"], WhatIfFigure);
	        this.choices = this.convertValues(source["choices"], WhatIfChoice);
	        this.notice = source["notice"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class WhatIfReading {
	    // Go type: WhatIfView
	    value: any;
	    provenance: snapshot.Provenance;
	    status: string;
	    absenceReason?: string;
	    count: number;
	    threshold: number;
	    baseline: string;
	    cutoff: string;
	    method: string;
	    inputSHA: string;
	    buildStamp: string;
	    howCounted: string;
	
	    static createFrom(source: any = {}) {
	        return new WhatIfReading(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.value = this.convertValues(source["value"], null);
	        this.provenance = this.convertValues(source["provenance"], snapshot.Provenance);
	        this.status = source["status"];
	        this.absenceReason = source["absenceReason"];
	        this.count = source["count"];
	        this.threshold = source["threshold"];
	        this.baseline = source["baseline"];
	        this.cutoff = source["cutoff"];
	        this.method = source["method"];
	        this.inputSHA = source["inputSHA"];
	        this.buildStamp = source["buildStamp"];
	        this.howCounted = source["howCounted"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace control {
	
	export class Alert {
	    kind: string;
	    urgency: string;
	    title: string;
	    detail: string;
	    node: string;
	    workspace: string;
	    at: time.Time;
	    provenance: snapshot.Provenance;
	    playerId?: string;
	    mflUrl?: string;
	
	    static createFrom(source: any = {}) {
	        return new Alert(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.urgency = source["urgency"];
	        this.title = source["title"];
	        this.detail = source["detail"];
	        this.node = source["node"];
	        this.workspace = source["workspace"];
	        this.at = this.convertValues(source["at"], time.Time);
	        this.provenance = this.convertValues(source["provenance"], snapshot.Provenance);
	        this.playerId = source["playerId"];
	        this.mflUrl = source["mflUrl"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Unavailable {
	    kind: string;
	    note: string;
	
	    static createFrom(source: any = {}) {
	        return new Unavailable(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.note = source["note"];
	    }
	}
	export class AlertReading {
	    alerts: Alert[];
	    unavailable: Unavailable[];
	
	    static createFrom(source: any = {}) {
	        return new AlertReading(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.alerts = this.convertValues(source["alerts"], Alert);
	        this.unavailable = this.convertValues(source["unavailable"], Unavailable);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class BylawSection {
	    title: string;
	    lines: string[];
	
	    static createFrom(source: any = {}) {
	        return new BylawSection(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.lines = source["lines"];
	    }
	}
	export class MFLKeyStatus {
	    myFranchise?: string;
	    teamActionRefusal: string;
	    state: string;
	    league: string;
	    season: number;
	    verifiedAt?: string;
	    detail?: string;
	
	    static createFrom(source: any = {}) {
	        return new MFLKeyStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.myFranchise = source["myFranchise"];
	        this.teamActionRefusal = source["teamActionRefusal"];
	        this.state = source["state"];
	        this.league = source["league"];
	        this.season = source["season"];
	        this.verifiedAt = source["verifiedAt"];
	        this.detail = source["detail"];
	    }
	}
	export class ParamView {
	    key: string;
	    label: string;
	    group: string;
	    position: string;
	    description: string;
	    default: number;
	    min: number;
	    max: number;
	    value: number;
	    calibrated: boolean;
	    overridden: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ParamView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.label = source["label"];
	        this.group = source["group"];
	        this.position = source["position"];
	        this.description = source["description"];
	        this.default = source["default"];
	        this.min = source["min"];
	        this.max = source["max"];
	        this.value = source["value"];
	        this.calibrated = source["calibrated"];
	        this.overridden = source["overridden"];
	    }
	}
	export class ParamsResult {
	    ok: boolean;
	    error: string;
	    params: ParamView[];
	
	    static createFrom(source: any = {}) {
	        return new ParamsResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.error = source["error"];
	        this.params = this.convertValues(source["params"], ParamView);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SeasonReading {
	    // Go type: snapshot
	    transactions: any;
	    lineups: snapshot.Sourced_github_com_secureprospective_TheWarRoom_internal_leaguefeed_Lineups_;
	    // Go type: snapshot
	    pendingTrades: any;
	
	    static createFrom(source: any = {}) {
	        return new SeasonReading(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.transactions = this.convertValues(source["transactions"], null);
	        this.lineups = this.convertValues(source["lineups"], snapshot.Sourced_github_com_secureprospective_TheWarRoom_internal_leaguefeed_Lineups_);
	        this.pendingTrades = this.convertValues(source["pendingTrades"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SetParamResult {
	    ok: boolean;
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new SetParamResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.error = source["error"];
	    }
	}
	export class ShellStatus {
	    health: string;
	    state: string;
	    gameday: string;
	
	    static createFrom(source: any = {}) {
	        return new ShellStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.health = source["health"];
	        this.state = source["state"];
	        this.gameday = source["gameday"];
	    }
	}

}

export namespace counterparties {
	
	export class CounterRow {
	    dayText: string;
	    sinceText: string;
	    nText: string;
	    partnersText: string;
	    fid: string;
	    name: string;
	    n: number;
	    perSeason: number;
	    tiers: Record<string, number>;
	    // Go type: CounterShare
	    forNow: any;
	    // Go type: CounterShare
	    forLater: any;
	    // Go type: lab
	    day: any;
	    // Go type: lab
	    since: any;
	    // Go type: lab
	    picks: any;
	    // Go type: lab
	    age: any;
	    // Go type: lab
	    partner?: any;
	    partnerName: string;
	    partnersCount: number;
	    standingTitle: string;
	    partnerTrades: string;
	
	    static createFrom(source: any = {}) {
	        return new CounterRow(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.dayText = source["dayText"];
	        this.sinceText = source["sinceText"];
	        this.nText = source["nText"];
	        this.partnersText = source["partnersText"];
	        this.fid = source["fid"];
	        this.name = source["name"];
	        this.n = source["n"];
	        this.perSeason = source["perSeason"];
	        this.tiers = source["tiers"];
	        this.forNow = this.convertValues(source["forNow"], null);
	        this.forLater = this.convertValues(source["forLater"], null);
	        this.day = this.convertValues(source["day"], null);
	        this.since = this.convertValues(source["since"], null);
	        this.picks = this.convertValues(source["picks"], null);
	        this.age = this.convertValues(source["age"], null);
	        this.partner = this.convertValues(source["partner"], null);
	        this.partnerName = source["partnerName"];
	        this.partnersCount = source["partnersCount"];
	        this.standingTitle = source["standingTitle"];
	        this.partnerTrades = source["partnerTrades"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PairView {
	    other: string;
	    excluded: boolean;
	    n: number;
	    // Go type: lab
	    now: any;
	    // Go type: lab
	    real: any;
	    trades: string;
	    title: string;
	
	    static createFrom(source: any = {}) {
	        return new PairView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.other = source["other"];
	        this.excluded = source["excluded"];
	        this.n = source["n"];
	        this.now = this.convertValues(source["now"], null);
	        this.real = this.convertValues(source["real"], null);
	        this.trades = source["trades"];
	        this.title = source["title"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CounterShift {
	    early: number;
	    late: number;
	    from: number[];
	    to: number[];
	    clear: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CounterShift(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.early = source["early"];
	        this.late = source["late"];
	        this.from = source["from"];
	        this.to = source["to"];
	        this.clear = source["clear"];
	    }
	}
	export class CounterShifts {
	    // Go type: CounterShift
	    now?: any;
	    // Go type: CounterShift
	    later?: any;
	
	    static createFrom(source: any = {}) {
	        return new CounterShifts(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.now = this.convertValues(source["now"], null);
	        this.later = this.convertValues(source["later"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CounterShare {
	    k: number;
	    n: number;
	    p?: number;
	    lo?: number;
	    hi?: number;
	    base?: number;
	    tone?: string;
	    countText: string;
	
	    static createFrom(source: any = {}) {
	        return new CounterShare(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.k = source["k"];
	        this.n = source["n"];
	        this.p = source["p"];
	        this.lo = source["lo"];
	        this.hi = source["hi"];
	        this.base = source["base"];
	        this.tone = source["tone"];
	        this.countText = source["countText"];
	    }
	}
	export class CounterTrendRow {
	    season: number;
	    n: number;
	    net?: number;
	    // Go type: CounterShare
	    now: any;
	    // Go type: CounterShare
	    later: any;
	    showShares: boolean;
	    nText: string;
	
	    static createFrom(source: any = {}) {
	        return new CounterTrendRow(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.season = source["season"];
	        this.n = source["n"];
	        this.net = source["net"];
	        this.now = this.convertValues(source["now"], null);
	        this.later = this.convertValues(source["later"], null);
	        this.showShares = source["showShares"];
	        this.nText = source["nText"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CounterTrend {
	    tier: string;
	    rows: CounterTrendRow[];
	    // Go type: CounterShifts
	    shift: any;
	
	    static createFrom(source: any = {}) {
	        return new CounterTrend(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tier = source["tier"];
	        this.rows = this.convertValues(source["rows"], CounterTrendRow);
	        this.shift = this.convertValues(source["shift"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CounterTrends {
	    seasons: number[];
	    tiers: CounterTrend[];
	
	    static createFrom(source: any = {}) {
	        return new CounterTrends(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.seasons = source["seasons"];
	        this.tiers = this.convertValues(source["tiers"], CounterTrend);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CardRow {
	    label: string;
	    text: string;
	    tone: string;
	    // Go type: lab
	    range?: any;
	
	    static createFrom(source: any = {}) {
	        return new CardRow(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.label = source["label"];
	        this.text = source["text"];
	        this.tone = source["tone"];
	        this.range = this.convertValues(source["range"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PostureCell {
	    id: string;
	    label: string;
	    p?: number;
	    tone: string;
	    title: string;
	
	    static createFrom(source: any = {}) {
	        return new PostureCell(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.label = source["label"];
	        this.p = source["p"];
	        this.tone = source["tone"];
	        this.title = source["title"];
	    }
	}
	export class TierCard {
	    id: string;
	    label: string;
	    hint: string;
	    excluded: boolean;
	    thin: boolean;
	    perSeason: number;
	    perSeasonText: string;
	    posture: PostureCell[];
	    rows: CardRow[];
	
	    static createFrom(source: any = {}) {
	        return new TierCard(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.label = source["label"];
	        this.hint = source["hint"];
	        this.excluded = source["excluded"];
	        this.thin = source["thin"];
	        this.perSeason = source["perSeason"];
	        this.perSeasonText = source["perSeasonText"];
	        this.posture = this.convertValues(source["posture"], PostureCell);
	        this.rows = this.convertValues(source["rows"], CardRow);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class TeamOption {
	    fid: string;
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new TeamOption(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.fid = source["fid"];
	        this.name = source["name"];
	    }
	}
	export class TierPick {
	    with: string;
	    tradingAs: string;
	    id: string;
	    label: string;
	    hint: string;
	    count: number;
	
	    static createFrom(source: any = {}) {
	        return new TierPick(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.with = source["with"];
	        this.tradingAs = source["tradingAs"];
	        this.id = source["id"];
	        this.label = source["label"];
	        this.hint = source["hint"];
	        this.count = source["count"];
	    }
	}
	export class CounterWords {
	    team: string;
	    sides: string;
	    perSeason: string;
	    now: string;
	    later: string;
	    picks: string;
	    age: string;
	    partners: string;
	    partner: string;
	    day: string;
	    since: string;
	    legend: string;
	    up: string;
	    down: string;
	
	    static createFrom(source: any = {}) {
	        return new CounterWords(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.team = source["team"];
	        this.sides = source["sides"];
	        this.perSeason = source["perSeason"];
	        this.now = source["now"];
	        this.later = source["later"];
	        this.picks = source["picks"];
	        this.age = source["age"];
	        this.partners = source["partners"];
	        this.partner = source["partner"];
	        this.day = source["day"];
	        this.since = source["since"];
	        this.legend = source["legend"];
	        this.up = source["up"];
	        this.down = source["down"];
	    }
	}
	export class Counterparties {
	    // Go type: CounterWords
	    words: any;
	    countText: string;
	    summary: string;
	    asOf: string;
	    headline: string;
	    note: string;
	    picks: TierPick[];
	    teams: TeamOption[];
	    cards: TierCard[];
	    trendSay: string[];
	    // Go type: CounterTrends
	    trends: any;
	    pairs: PairView[];
	    rows: CounterRow[];
	
	    static createFrom(source: any = {}) {
	        return new Counterparties(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.words = this.convertValues(source["words"], null);
	        this.countText = source["countText"];
	        this.summary = source["summary"];
	        this.asOf = source["asOf"];
	        this.headline = source["headline"];
	        this.note = source["note"];
	        this.picks = this.convertValues(source["picks"], TierPick);
	        this.teams = this.convertValues(source["teams"], TeamOption);
	        this.cards = this.convertValues(source["cards"], TierCard);
	        this.trendSay = source["trendSay"];
	        this.trends = this.convertValues(source["trends"], null);
	        this.pairs = this.convertValues(source["pairs"], PairView);
	        this.rows = this.convertValues(source["rows"], CounterRow);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class BoardReading {
	    // Go type: Counterparties
	    value: any;
	    provenance: snapshot.Provenance;
	    status: string;
	    absenceReason?: string;
	    count: number;
	    threshold: number;
	    baseline: string;
	    cutoff: string;
	    method: string;
	    inputSHA: string;
	    buildStamp: string;
	    howCounted: string;
	
	    static createFrom(source: any = {}) {
	        return new BoardReading(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.value = this.convertValues(source["value"], null);
	        this.provenance = this.convertValues(source["provenance"], snapshot.Provenance);
	        this.status = source["status"];
	        this.absenceReason = source["absenceReason"];
	        this.count = source["count"];
	        this.threshold = source["threshold"];
	        this.baseline = source["baseline"];
	        this.cutoff = source["cutoff"];
	        this.method = source["method"];
	        this.inputSHA = source["inputSHA"];
	        this.buildStamp = source["buildStamp"];
	        this.howCounted = source["howCounted"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class RawCounterpartiesScope {
	    from: number;
	    to: number;
	    tiers: string[];
	    windows: string[];
	    postures: string[];
	    team: string;
	
	    static createFrom(source: any = {}) {
	        return new RawCounterpartiesScope(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.from = source["from"];
	        this.to = source["to"];
	        this.tiers = source["tiers"];
	        this.windows = source["windows"];
	        this.postures = source["postures"];
	        this.team = source["team"];
	    }
	}
	export class RawCounterpartTeam {
	    scope: RawCounterpartiesScope;
	    fid: string;
	
	    static createFrom(source: any = {}) {
	        return new RawCounterpartTeam(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.scope = this.convertValues(source["scope"], RawCounterpartiesScope);
	        this.fid = source["fid"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class AsTierView {
	    excluded: boolean;
	    tier: string;
	    n: number;
	    // Go type: lab
	    edge: any;
	    label: string;
	    text: string;
	
	    static createFrom(source: any = {}) {
	        return new AsTierView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.excluded = source["excluded"];
	        this.tier = source["tier"];
	        this.n = source["n"];
	        this.edge = this.convertValues(source["edge"], null);
	        this.label = source["label"];
	        this.text = source["text"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PartnerView {
	    fid: string;
	    name: string;
	    n: number;
	    // Go type: lab
	    net: any;
	    tradesText: string;
	    netText: string;
	
	    static createFrom(source: any = {}) {
	        return new PartnerView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.fid = source["fid"];
	        this.name = source["name"];
	        this.n = source["n"];
	        this.net = this.convertValues(source["net"], null);
	        this.tradesText = source["tradesText"];
	        this.netText = source["netText"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class KindFlowView {
	    kind: string;
	    name: string;
	    inN: number;
	    outN: number;
	    inW: number;
	    outW: number;
	    text: string;
	
	    static createFrom(source: any = {}) {
	        return new KindFlowView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.name = source["name"];
	        this.inN = source["inN"];
	        this.outN = source["outN"];
	        this.inW = source["inW"];
	        this.outW = source["outW"];
	        this.text = source["text"];
	    }
	}
	export class SeasonCellView {
	    nowText: string;
	    laterText: string;
	    swapText: string;
	    againstText: string;
	    season: number;
	    n: number;
	    now: number;
	    later: number;
	    swap: number;
	    against: number;
	    flagged: boolean;
	    mostlyText: string;
	    net?: number;
	    finish?: number;
	    label: string;
	    title: string;
	
	    static createFrom(source: any = {}) {
	        return new SeasonCellView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.nowText = source["nowText"];
	        this.laterText = source["laterText"];
	        this.swapText = source["swapText"];
	        this.againstText = source["againstText"];
	        this.season = source["season"];
	        this.n = source["n"];
	        this.now = source["now"];
	        this.later = source["later"];
	        this.swap = source["swap"];
	        this.against = source["against"];
	        this.flagged = source["flagged"];
	        this.mostlyText = source["mostlyText"];
	        this.net = source["net"];
	        this.finish = source["finish"];
	        this.label = source["label"];
	        this.title = source["title"];
	    }
	}
	export class TeamCounterpart {
	    partnersText: string;
	    // Go type: CounterWords
	    words: any;
	    countText: string;
	    fid: string;
	    name: string;
	    asOf: string;
	    flowScale: number;
	    sides: number;
	    empty: boolean;
	    note: string;
	    seasons: SeasonCellView[];
	    seasonSay: string;
	    posture: PostureCell[];
	    kinds: KindFlowView[];
	    kindSay: string;
	    partners: PartnerView[];
	    partnersSay: string;
	    // Go type: lab
	    behaviour: any;
	    asTier: AsTierView[];
	    // Go type: lab
	    day: any;
	    // Go type: lab
	    since: any;
	    dayText: string;
	    sinceText: string;
	
	    static createFrom(source: any = {}) {
	        return new TeamCounterpart(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.partnersText = source["partnersText"];
	        this.words = this.convertValues(source["words"], null);
	        this.countText = source["countText"];
	        this.fid = source["fid"];
	        this.name = source["name"];
	        this.asOf = source["asOf"];
	        this.flowScale = source["flowScale"];
	        this.sides = source["sides"];
	        this.empty = source["empty"];
	        this.note = source["note"];
	        this.seasons = this.convertValues(source["seasons"], SeasonCellView);
	        this.seasonSay = source["seasonSay"];
	        this.posture = this.convertValues(source["posture"], PostureCell);
	        this.kinds = this.convertValues(source["kinds"], KindFlowView);
	        this.kindSay = source["kindSay"];
	        this.partners = this.convertValues(source["partners"], PartnerView);
	        this.partnersSay = source["partnersSay"];
	        this.behaviour = this.convertValues(source["behaviour"], null);
	        this.asTier = this.convertValues(source["asTier"], AsTierView);
	        this.day = this.convertValues(source["day"], null);
	        this.since = this.convertValues(source["since"], null);
	        this.dayText = source["dayText"];
	        this.sinceText = source["sinceText"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class TeamCounterpartReading {
	    // Go type: TeamCounterpart
	    value: any;
	    provenance: snapshot.Provenance;
	    status: string;
	    absenceReason?: string;
	    count: number;
	    threshold: number;
	    baseline: string;
	    cutoff: string;
	    method: string;
	    inputSHA: string;
	    buildStamp: string;
	    howCounted: string;
	
	    static createFrom(source: any = {}) {
	        return new TeamCounterpartReading(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.value = this.convertValues(source["value"], null);
	        this.provenance = this.convertValues(source["provenance"], snapshot.Provenance);
	        this.status = source["status"];
	        this.absenceReason = source["absenceReason"];
	        this.count = source["count"];
	        this.threshold = source["threshold"];
	        this.baseline = source["baseline"];
	        this.cutoff = source["cutoff"];
	        this.method = source["method"];
	        this.inputSHA = source["inputSHA"];
	        this.buildStamp = source["buildStamp"];
	        this.howCounted = source["howCounted"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace domain {
	
	export class Freshness {
	    state: string;
	    fetchedAt: string;
	    note: string;
	
	    static createFrom(source: any = {}) {
	        return new Freshness(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.state = source["state"];
	        this.fetchedAt = source["fetchedAt"];
	        this.note = source["note"];
	    }
	}

}

export namespace draft {
	
	export class LabReading {
	    // Go type: report
	    value: any;
	    provenance: snapshot.Provenance;
	    status: string;
	    absenceReason?: string;
	    count: number;
	    threshold: number;
	    baseline: string;
	    cutoff: string;
	    method: string;
	    inputSHA: string;
	    buildStamp: string;
	    howCounted: string;
	
	    static createFrom(source: any = {}) {
	        return new LabReading(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.value = this.convertValues(source["value"], null);
	        this.provenance = this.convertValues(source["provenance"], snapshot.Provenance);
	        this.status = source["status"];
	        this.absenceReason = source["absenceReason"];
	        this.count = source["count"];
	        this.threshold = source["threshold"];
	        this.baseline = source["baseline"];
	        this.cutoff = source["cutoff"];
	        this.method = source["method"];
	        this.inputSHA = source["inputSHA"];
	        this.buildStamp = source["buildStamp"];
	        this.howCounted = source["howCounted"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class RawDraftScope {
	    from: number;
	    to: number;
	    seasons: number;
	    segments: string[];
	    nfl: string[];
	    group: string;
	
	    static createFrom(source: any = {}) {
	        return new RawDraftScope(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.from = source["from"];
	        this.to = source["to"];
	        this.seasons = source["seasons"];
	        this.segments = source["segments"];
	        this.nfl = source["nfl"];
	        this.group = source["group"];
	    }
	}

}

export namespace eligibility {
	
	export class Decision {
	    direction: string;
	    verdict: string;
	    reason?: string;
	    source: string;
	
	    static createFrom(source: any = {}) {
	        return new Decision(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.direction = source["direction"];
	        this.verdict = source["verdict"];
	        this.reason = source["reason"];
	        this.source = source["source"];
	    }
	}
	export class PlayerVerdicts {
	    ir: Decision;
	    taxi: Decision;
	
	    static createFrom(source: any = {}) {
	        return new PlayerVerdicts(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ir = this.convertValues(source["ir"], Decision);
	        this.taxi = this.convertValues(source["taxi"], Decision);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace envelope {
	
	export class AssetLabel {
	    token: string;
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new AssetLabel(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.token = source["token"];
	        this.name = source["name"];
	    }
	}
	export class AuditEntry {
	    at: time.Time;
	    from: string;
	    event: string;
	    to: string;
	    note: string;
	
	    static createFrom(source: any = {}) {
	        return new AuditEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.at = this.convertValues(source["at"], time.Time);
	        this.from = source["from"];
	        this.event = source["event"];
	        this.to = source["to"];
	        this.note = source["note"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class BidFacts {
	    conditionalDrop: string;
	    conditionalName: string;
	    player: string;
	    salary: number;
	    years: number;
	    points: number;
	    snipe: boolean;
	    beat?: number;
	    evidence: string;
	    closes: string;
	
	    static createFrom(source: any = {}) {
	        return new BidFacts(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.conditionalDrop = source["conditionalDrop"];
	        this.conditionalName = source["conditionalName"];
	        this.player = source["player"];
	        this.salary = source["salary"];
	        this.years = source["years"];
	        this.points = source["points"];
	        this.snipe = source["snipe"];
	        this.beat = source["beat"];
	        this.evidence = source["evidence"];
	        this.closes = source["closes"];
	    }
	}
	export class ExpectedLineup {
	    week: number;
	    starters: string[];
	    baseline: string[];
	
	    static createFrom(source: any = {}) {
	        return new ExpectedLineup(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.week = source["week"];
	        this.starters = source["starters"];
	        this.baseline = source["baseline"];
	    }
	}
	export class ExpectedTrade {
	    tradeId: string;
	    offering: string;
	    accepting: string;
	    offeringGives: string[];
	    acceptingGives: string[];
	
	    static createFrom(source: any = {}) {
	        return new ExpectedTrade(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tradeId = source["tradeId"];
	        this.offering = source["offering"];
	        this.accepting = source["accepting"];
	        this.offeringGives = source["offeringGives"];
	        this.acceptingGives = source["acceptingGives"];
	    }
	}
	export class ExpectedChange {
	    trade?: ExpectedTrade;
	    lineup?: ExpectedLineup;
	    player?: string;
	    rosterStatus?: string;
	
	    static createFrom(source: any = {}) {
	        return new ExpectedChange(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.trade = this.convertValues(source["trade"], ExpectedTrade);
	        this.lineup = this.convertValues(source["lineup"], ExpectedLineup);
	        this.player = source["player"];
	        this.rosterStatus = source["rosterStatus"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	export class Target {
	    kind: string;
	    url?: string;
	
	    static createFrom(source: any = {}) {
	        return new Target(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.url = source["url"];
	    }
	}
	export class Subject {
	    players: string[];
	    picks: string[];
	
	    static createFrom(source: any = {}) {
	        return new Subject(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.players = source["players"];
	        this.picks = source["picks"];
	    }
	}
	export class Spec {
	    intent: string;
	    leagueId: string;
	    franchiseId: string;
	    subject: Subject;
	    expected: ExpectedChange;
	    gravity: string;
	    undo: string;
	    target: Target;
	    deadline?: time.Time;
	
	    static createFrom(source: any = {}) {
	        return new Spec(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.intent = source["intent"];
	        this.leagueId = source["leagueId"];
	        this.franchiseId = source["franchiseId"];
	        this.subject = this.convertValues(source["subject"], Subject);
	        this.expected = this.convertValues(source["expected"], ExpectedChange);
	        this.gravity = source["gravity"];
	        this.undo = source["undo"];
	        this.target = this.convertValues(source["target"], Target);
	        this.deadline = this.convertValues(source["deadline"], time.Time);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Receipt {
	    tradeAssets?: AssetLabel[];
	    correlationId: string;
	    spec: Spec;
	    state: string;
	    audit: AuditEntry[];
	
	    static createFrom(source: any = {}) {
	        return new Receipt(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tradeAssets = this.convertValues(source["tradeAssets"], AssetLabel);
	        this.correlationId = source["correlationId"];
	        this.spec = this.convertValues(source["spec"], Spec);
	        this.state = source["state"];
	        this.audit = this.convertValues(source["audit"], AuditEntry);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	

}

export namespace leagueclock {
	
	export class WindowException {
	    permanent?: boolean;
	    action: string;
	    franchise: string;
	    start: time.Time;
	    until: time.Time;
	    reason: string;
	
	    static createFrom(source: any = {}) {
	        return new WindowException(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.permanent = source["permanent"];
	        this.action = source["action"];
	        this.franchise = source["franchise"];
	        this.start = this.convertValues(source["start"], time.Time);
	        this.until = this.convertValues(source["until"], time.Time);
	        this.reason = source["reason"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ActionWindow {
	    id: string;
	    opens: string[];
	    start: time.Time;
	    close?: time.Time;
	    live: boolean;
	    secondsToClose?: number;
	
	    static createFrom(source: any = {}) {
	        return new ActionWindow(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.opens = source["opens"];
	        this.start = this.convertValues(source["start"], time.Time);
	        this.close = this.convertValues(source["close"], time.Time);
	        this.live = source["live"];
	        this.secondsToClose = source["secondsToClose"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ActionCalendar {
	    windows: ActionWindow[];
	    exceptions: WindowException[];
	    tradeDeadline?: time.Time;
	    secondsToTradeDeadline?: number;
	
	    static createFrom(source: any = {}) {
	        return new ActionCalendar(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.windows = this.convertValues(source["windows"], ActionWindow);
	        this.exceptions = this.convertValues(source["exceptions"], WindowException);
	        this.tradeDeadline = this.convertValues(source["tradeDeadline"], time.Time);
	        this.secondsToTradeDeadline = source["secondsToTradeDeadline"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ActionDecision {
	    open: boolean;
	    reason: string;
	    window: string;
	    next?: time.Time;
	
	    static createFrom(source: any = {}) {
	        return new ActionDecision(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.open = source["open"];
	        this.reason = source["reason"];
	        this.window = source["window"];
	        this.next = this.convertValues(source["next"], time.Time);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class Deadline {
	    id: string;
	    label: string;
	    at?: time.Time;
	    urgency: string;
	    pinned: boolean;
	    promoted: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Deadline(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.label = source["label"];
	        this.at = this.convertValues(source["at"], time.Time);
	        this.urgency = source["urgency"];
	        this.pinned = source["pinned"];
	        this.promoted = source["promoted"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Window {
	    kind: string;
	    status: string;
	    at?: time.Time;
	    moved: boolean;
	    reason: string;
	
	    static createFrom(source: any = {}) {
	        return new Window(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.status = source["status"];
	        this.at = this.convertValues(source["at"], time.Time);
	        this.moved = source["moved"];
	        this.reason = source["reason"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Reading {
	    season: number;
	    phase: string;
	    week?: number;
	    deadlines: Deadline[];
	    windows: Window[];
	
	    static createFrom(source: any = {}) {
	        return new Reading(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.season = source["season"];
	        this.phase = source["phase"];
	        this.week = source["week"];
	        this.deadlines = this.convertValues(source["deadlines"], Deadline);
	        this.windows = this.convertValues(source["windows"], Window);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	

}

export namespace leaguefeed {
	
	export class FuturePick {
	    franchise: string;
	    year: number;
	    round: number;
	
	    static createFrom(source: any = {}) {
	        return new FuturePick(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.franchise = source["franchise"];
	        this.year = source["year"];
	        this.round = source["round"];
	    }
	}
	export class CurrentPick {
	    round: number;
	    pick: number;
	
	    static createFrom(source: any = {}) {
	        return new CurrentPick(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.round = source["round"];
	        this.pick = source["pick"];
	    }
	}
	export class Asset {
	    player?: string;
	    currentPick?: CurrentPick;
	    futurePick?: FuturePick;
	    blindBidCents?: number;
	
	    static createFrom(source: any = {}) {
	        return new Asset(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.player = source["player"];
	        this.currentPick = this.convertValues(source["currentPick"], CurrentPick);
	        this.futurePick = this.convertValues(source["futurePick"], FuturePick);
	        this.blindBidCents = source["blindBidCents"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	export class PlayerScore {
	    id: string;
	    score: number;
	    secondsRemaining: number;
	
	    static createFrom(source: any = {}) {
	        return new PlayerScore(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.score = source["score"];
	        this.secondsRemaining = source["secondsRemaining"];
	    }
	}
	export class Lineup {
	    playing: number;
	    yetToPlay: number;
	    players: PlayerScore[];
	    franchise: string;
	    starters: string[];
	    nonStarters: string[];
	    score: number;
	    secondsRemaining: number;
	
	    static createFrom(source: any = {}) {
	        return new Lineup(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.playing = source["playing"];
	        this.yetToPlay = source["yetToPlay"];
	        this.players = this.convertValues(source["players"], PlayerScore);
	        this.franchise = source["franchise"];
	        this.starters = source["starters"];
	        this.nonStarters = source["nonStarters"];
	        this.score = source["score"];
	        this.secondsRemaining = source["secondsRemaining"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Matchup {
	    home: string;
	    away: string;
	
	    static createFrom(source: any = {}) {
	        return new Matchup(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.home = source["home"];
	        this.away = source["away"];
	    }
	}
	export class Lineups {
	    matchups: Matchup[];
	    week: number;
	    franchises: Lineup[];
	
	    static createFrom(source: any = {}) {
	        return new Lineups(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.matchups = this.convertValues(source["matchups"], Matchup);
	        this.week = source["week"];
	        this.franchises = this.convertValues(source["franchises"], Lineup);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class PendingTrade {
	    id: string;
	    offering: string;
	    offeredTo: string;
	    gives: Asset[];
	    gets: Asset[];
	    proposed: time.Time;
	    expires: time.Time;
	    comments: string;
	    description: string;
	
	    static createFrom(source: any = {}) {
	        return new PendingTrade(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.offering = source["offering"];
	        this.offeredTo = source["offeredTo"];
	        this.gives = this.convertValues(source["gives"], Asset);
	        this.gets = this.convertValues(source["gets"], Asset);
	        this.proposed = this.convertValues(source["proposed"], time.Time);
	        this.expires = this.convertValues(source["expires"], time.Time);
	        this.comments = source["comments"];
	        this.description = source["description"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class RosterChange {
	    in: Asset[];
	    out: Asset[];
	
	    static createFrom(source: any = {}) {
	        return new RosterChange(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.in = this.convertValues(source["in"], Asset);
	        this.out = this.convertValues(source["out"], Asset);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Trade {
	    counterparty: string;
	    gave: Asset[];
	    counterpartyGave: Asset[];
	    comments: string;
	    expires: time.Time;
	
	    static createFrom(source: any = {}) {
	        return new Trade(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.counterparty = source["counterparty"];
	        this.gave = this.convertValues(source["gave"], Asset);
	        this.counterpartyGave = this.convertValues(source["counterpartyGave"], Asset);
	        this.comments = source["comments"];
	        this.expires = this.convertValues(source["expires"], time.Time);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Transaction {
	    kind: string;
	    time: time.Time;
	    franchise: string;
	    byCommish: boolean;
	    trade?: Trade;
	    addsDrops?: RosterChange;
	    ir?: RosterChange;
	    taxi?: RosterChange;
	    unparsed: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Transaction(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.time = this.convertValues(source["time"], time.Time);
	        this.franchise = source["franchise"];
	        this.byCommish = source["byCommish"];
	        this.trade = this.convertValues(source["trade"], Trade);
	        this.addsDrops = this.convertValues(source["addsDrops"], RosterChange);
	        this.ir = this.convertValues(source["ir"], RosterChange);
	        this.taxi = this.convertValues(source["taxi"], RosterChange);
	        this.unparsed = source["unparsed"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace leaguerules {
	
	export class AutoMoveValue {
	    State: number;
	    Value: boolean;
	    Citation: string;
	
	    static createFrom(source: any = {}) {
	        return new AutoMoveValue(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.State = source["State"];
	        this.Value = source["Value"];
	        this.Citation = source["Citation"];
	    }
	}
	export class Value_int_ {
	    State: number;
	    Value: number;
	    Citation: string;
	
	    static createFrom(source: any = {}) {
	        return new Value_int_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.State = source["State"];
	        this.Value = source["Value"];
	        this.Citation = source["Citation"];
	    }
	}
	export class Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Money_ {
	    State: number;
	    Value: number;
	    Citation: string;
	
	    static createFrom(source: any = {}) {
	        return new Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Money_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.State = source["State"];
	        this.Value = source["Value"];
	        this.Citation = source["Citation"];
	    }
	}
	export class Value_bool_ {
	    State: number;
	    Value: boolean;
	    Citation: string;
	
	    static createFrom(source: any = {}) {
	        return new Value_bool_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.State = source["State"];
	        this.Value = source["Value"];
	        this.Citation = source["Citation"];
	    }
	}
	export class BoundsMoney {
	    MaximumUnlimited: Value_bool_;
	    Min: Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Money_;
	    Max: Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Money_;
	
	    static createFrom(source: any = {}) {
	        return new BoundsMoney(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.MaximumUnlimited = this.convertValues(source["MaximumUnlimited"], Value_bool_);
	        this.Min = this.convertValues(source["Min"], Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Money_);
	        this.Max = this.convertValues(source["Max"], Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Money_);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class BidTier {
	    Salary: BoundsMoney;
	    MinimumYears: Value_int_;
	    LowerInclusive: Value_bool_;
	
	    static createFrom(source: any = {}) {
	        return new BidTier(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Salary = this.convertValues(source["Salary"], BoundsMoney);
	        this.MinimumYears = this.convertValues(source["MinimumYears"], Value_int_);
	        this.LowerInclusive = this.convertValues(source["LowerInclusive"], Value_bool_);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Value_time_Duration_ {
	    State: number;
	    Value: number;
	    Citation: string;
	
	    static createFrom(source: any = {}) {
	        return new Value_time_Duration_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.State = source["State"];
	        this.Value = source["Value"];
	        this.Citation = source["Citation"];
	    }
	}
	export class Value___github_com_secureprospective_TheWarRoom_internal_leaguerules_BidTier_ {
	    State: number;
	    Value: BidTier[];
	    Citation: string;
	
	    static createFrom(source: any = {}) {
	        return new Value___github_com_secureprospective_TheWarRoom_internal_leaguerules_BidTier_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.State = source["State"];
	        this.Value = this.convertValues(source["Value"], BidTier);
	        this.Citation = source["Citation"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Decimal_ {
	    State: number;
	    Value: number;
	    Citation: string;
	
	    static createFrom(source: any = {}) {
	        return new Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Decimal_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.State = source["State"];
	        this.Value = source["Value"];
	        this.Citation = source["Citation"];
	    }
	}
	export class Value___github_com_secureprospective_TheWarRoom_internal_leaguerules_Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Decimal__ {
	    State: number;
	    Value: Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Decimal_[];
	    Citation: string;
	
	    static createFrom(source: any = {}) {
	        return new Value___github_com_secureprospective_TheWarRoom_internal_leaguerules_Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Decimal__(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.State = source["State"];
	        this.Value = this.convertValues(source["Value"], Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Decimal_);
	        this.Citation = source["Citation"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_BidMode_ {
	    State: number;
	    Value: string;
	    Citation: string;
	
	    static createFrom(source: any = {}) {
	        return new Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_BidMode_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.State = source["State"];
	        this.Value = source["Value"];
	        this.Citation = source["Citation"];
	    }
	}
	export class BidsRules {
	    UFA: Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_BidMode_;
	    RFA: Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_BidMode_;
	    // Go type: Value___github_com_secureprospective_TheWarRoom_internal_leaguerules_Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Decimal__
	    YearMultipliers: any;
	    MaximumActive: Value_int_;
	    OneYearMaximum: Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Money_;
	    // Go type: Value___github_com_secureprospective_TheWarRoom_internal_leaguerules_BidTier_
	    Tiers: any;
	    MinimumIncrement: Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Decimal_;
	    StandFor: Value_time_Duration_;
	    SnipeAfter: Value_time_Duration_;
	    SnipeIncrement: Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Decimal_;
	    MultiYearEndWeek: Value_int_;
	    PlayoffOnlyAfterWeek: Value_int_;
	    PlayoffMaximumYears: Value_int_;
	    PlayoffCapLifted: Value_bool_;
	    PointDecimals: Value_int_;
	
	    static createFrom(source: any = {}) {
	        return new BidsRules(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.UFA = this.convertValues(source["UFA"], Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_BidMode_);
	        this.RFA = this.convertValues(source["RFA"], Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_BidMode_);
	        this.YearMultipliers = this.convertValues(source["YearMultipliers"], null);
	        this.MaximumActive = this.convertValues(source["MaximumActive"], Value_int_);
	        this.OneYearMaximum = this.convertValues(source["OneYearMaximum"], Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Money_);
	        this.Tiers = this.convertValues(source["Tiers"], null);
	        this.MinimumIncrement = this.convertValues(source["MinimumIncrement"], Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Decimal_);
	        this.StandFor = this.convertValues(source["StandFor"], Value_time_Duration_);
	        this.SnipeAfter = this.convertValues(source["SnipeAfter"], Value_time_Duration_);
	        this.SnipeIncrement = this.convertValues(source["SnipeIncrement"], Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Decimal_);
	        this.MultiYearEndWeek = this.convertValues(source["MultiYearEndWeek"], Value_int_);
	        this.PlayoffOnlyAfterWeek = this.convertValues(source["PlayoffOnlyAfterWeek"], Value_int_);
	        this.PlayoffMaximumYears = this.convertValues(source["PlayoffMaximumYears"], Value_int_);
	        this.PlayoffCapLifted = this.convertValues(source["PlayoffCapLifted"], Value_bool_);
	        this.PointDecimals = this.convertValues(source["PointDecimals"], Value_int_);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Bonus {
	    At: Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Decimal_;
	    Points: Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Decimal_;
	
	    static createFrom(source: any = {}) {
	        return new Bonus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.At = this.convertValues(source["At"], Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Decimal_);
	        this.Points = this.convertValues(source["Points"], Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Decimal_);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Bounds {
	    Min: Value_int_;
	    Max: Value_int_;
	
	    static createFrom(source: any = {}) {
	        return new Bounds(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Min = this.convertValues(source["Min"], Value_int_);
	        this.Max = this.convertValues(source["Max"], Value_int_);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class Value_string_ {
	    State: number;
	    Value: string;
	    Citation: string;
	
	    static createFrom(source: any = {}) {
	        return new Value_string_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.State = source["State"];
	        this.Value = source["Value"];
	        this.Citation = source["Citation"];
	    }
	}
	export class BracketNode {
	    ID: Value_string_;
	    Week: Value_int_;
	    HomeSeed: Value_int_;
	    AwaySeed: Value_int_;
	    HomeWinner: Value_string_;
	    AwayWinner: Value_string_;
	
	    static createFrom(source: any = {}) {
	        return new BracketNode(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = this.convertValues(source["ID"], Value_string_);
	        this.Week = this.convertValues(source["Week"], Value_int_);
	        this.HomeSeed = this.convertValues(source["HomeSeed"], Value_int_);
	        this.AwaySeed = this.convertValues(source["AwaySeed"], Value_int_);
	        this.HomeWinner = this.convertValues(source["HomeWinner"], Value_string_);
	        this.AwayWinner = this.convertValues(source["AwayWinner"], Value_string_);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Value_map_int_github_com_secureprospective_TheWarRoom_internal_leaguerules_Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Decimal__ {
	    State: number;
	    Value: Record<number, Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Decimal_>;
	    Citation: string;
	
	    static createFrom(source: any = {}) {
	        return new Value_map_int_github_com_secureprospective_TheWarRoom_internal_leaguerules_Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Decimal__(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.State = source["State"];
	        this.Value = this.convertValues(source["Value"], Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Decimal_, true);
	        this.Citation = source["Citation"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class BuyoutRules {
	    Enabled: Value_bool_;
	    AnnualLimit: Value_int_;
	    OffseasonOnly: Value_bool_;
	    OneYearAllowed: Value_bool_;
	    Rates: Value_map_int_github_com_secureprospective_TheWarRoom_internal_leaguerules_Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Decimal__;
	    RebidNextOffseason: Value_bool_;
	
	    static createFrom(source: any = {}) {
	        return new BuyoutRules(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Enabled = this.convertValues(source["Enabled"], Value_bool_);
	        this.AnnualLimit = this.convertValues(source["AnnualLimit"], Value_int_);
	        this.OffseasonOnly = this.convertValues(source["OffseasonOnly"], Value_bool_);
	        this.OneYearAllowed = this.convertValues(source["OneYearAllowed"], Value_bool_);
	        this.Rates = this.convertValues(source["Rates"], Value_map_int_github_com_secureprospective_TheWarRoom_internal_leaguerules_Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Decimal__);
	        this.RebidNextOffseason = this.convertValues(source["RebidNextOffseason"], Value_bool_);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_BoundarySide_ {
	    State: number;
	    Value: string;
	    Citation: string;
	
	    static createFrom(source: any = {}) {
	        return new Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_BoundarySide_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.State = source["State"];
	        this.Value = source["Value"];
	        this.Citation = source["Citation"];
	    }
	}
	export class Value_time_Time_ {
	    State: number;
	    Value: time.Time;
	    Citation: string;
	
	    static createFrom(source: any = {}) {
	        return new Value_time_Time_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.State = source["State"];
	        this.Value = this.convertValues(source["Value"], time.Time);
	        this.Citation = source["Citation"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Value___github_com_secureprospective_TheWarRoom_internal_leaguerules_Action_ {
	    State: number;
	    Value: string[];
	    Citation: string;
	
	    static createFrom(source: any = {}) {
	        return new Value___github_com_secureprospective_TheWarRoom_internal_leaguerules_Action_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.State = source["State"];
	        this.Value = source["Value"];
	        this.Citation = source["Citation"];
	    }
	}
	export class MonthDay {
	    Month: Value_int_;
	    Day: Value_int_;
	
	    static createFrom(source: any = {}) {
	        return new MonthDay(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Month = this.convertValues(source["Month"], Value_int_);
	        this.Day = this.convertValues(source["Day"], Value_int_);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_MonthDay_ {
	    State: number;
	    Value: MonthDay;
	    Citation: string;
	
	    static createFrom(source: any = {}) {
	        return new Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_MonthDay_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.State = source["State"];
	        this.Value = this.convertValues(source["Value"], MonthDay);
	        this.Citation = source["Citation"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Window {
	    Start: Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_MonthDay_;
	    End: Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_MonthDay_;
	    // Go type: Value___github_com_secureprospective_TheWarRoom_internal_leaguerules_Action_
	    Opens: any;
	    DatedStart: Value_time_Time_;
	    DatedEnd: Value_time_Time_;
	
	    static createFrom(source: any = {}) {
	        return new Window(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Start = this.convertValues(source["Start"], Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_MonthDay_);
	        this.End = this.convertValues(source["End"], Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_MonthDay_);
	        this.Opens = this.convertValues(source["Opens"], null);
	        this.DatedStart = this.convertValues(source["DatedStart"], Value_time_Time_);
	        this.DatedEnd = this.convertValues(source["DatedEnd"], Value_time_Time_);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Value_map_string_github_com_secureprospective_TheWarRoom_internal_leaguerules_Window_ {
	    State: number;
	    Value: Record<string, Window>;
	    Citation: string;
	
	    static createFrom(source: any = {}) {
	        return new Value_map_string_github_com_secureprospective_TheWarRoom_internal_leaguerules_Window_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.State = source["State"];
	        this.Value = this.convertValues(source["Value"], Window, true);
	        this.Citation = source["Citation"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CalendarRules {
	    Windows: Value_map_string_github_com_secureprospective_TheWarRoom_internal_leaguerules_Window_;
	    Timezone: Value_string_;
	    Boundary: Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_BoundarySide_;
	    Rollover: Value_time_Time_;
	    RolloverDate: Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_MonthDay_;
	    TradeDeadline: Value_time_Time_;
	    TradeDeadlineWeek: Value_int_;
	
	    static createFrom(source: any = {}) {
	        return new CalendarRules(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Windows = this.convertValues(source["Windows"], Value_map_string_github_com_secureprospective_TheWarRoom_internal_leaguerules_Window_);
	        this.Timezone = this.convertValues(source["Timezone"], Value_string_);
	        this.Boundary = this.convertValues(source["Boundary"], Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_BoundarySide_);
	        this.Rollover = this.convertValues(source["Rollover"], Value_time_Time_);
	        this.RolloverDate = this.convertValues(source["RolloverDate"], Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_MonthDay_);
	        this.TradeDeadline = this.convertValues(source["TradeDeadline"], Value_time_Time_);
	        this.TradeDeadlineWeek = this.convertValues(source["TradeDeadlineWeek"], Value_int_);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_CapMode_ {
	    State: number;
	    Value: string;
	    Citation: string;
	
	    static createFrom(source: any = {}) {
	        return new Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_CapMode_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.State = source["State"];
	        this.Value = source["Value"];
	        this.Citation = source["Citation"];
	    }
	}
	export class CapRules {
	    Amount: Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Money_;
	    Enforcement: Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_CapMode_;
	    IRShare: Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Decimal_;
	    TaxiShare: Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Decimal_;
	    Quantum: Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Money_;
	    ComplianceDeadline: Value_time_Time_;
	
	    static createFrom(source: any = {}) {
	        return new CapRules(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Amount = this.convertValues(source["Amount"], Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Money_);
	        this.Enforcement = this.convertValues(source["Enforcement"], Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_CapMode_);
	        this.IRShare = this.convertValues(source["IRShare"], Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Decimal_);
	        this.TaxiShare = this.convertValues(source["TaxiShare"], Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Decimal_);
	        this.Quantum = this.convertValues(source["Quantum"], Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Money_);
	        this.ComplianceDeadline = this.convertValues(source["ComplianceDeadline"], Value_time_Time_);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Choice {
	    value: string;
	    label: string;
	
	    static createFrom(source: any = {}) {
	        return new Choice(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.value = source["value"];
	        this.label = source["label"];
	    }
	}
	export class Value___string_ {
	    State: number;
	    Value: string[];
	    Citation: string;
	
	    static createFrom(source: any = {}) {
	        return new Value___string_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.State = source["State"];
	        this.Value = source["Value"];
	        this.Citation = source["Citation"];
	    }
	}
	export class ReliefRules {
	    Enabled: Value_bool_;
	    Reasons: Value___string_;
	
	    static createFrom(source: any = {}) {
	        return new ReliefRules(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Enabled = this.convertValues(source["Enabled"], Value_bool_);
	        this.Reasons = this.convertValues(source["Reasons"], Value___string_);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_ReliefRules_ {
	    State: number;
	    Value: ReliefRules;
	    Citation: string;
	
	    static createFrom(source: any = {}) {
	        return new Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_ReliefRules_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.State = source["State"];
	        this.Value = this.convertValues(source["Value"], ReliefRules);
	        this.Citation = source["Citation"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SpecialRules {
	    Enabled: Value_bool_;
	    Rate: Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Decimal_;
	
	    static createFrom(source: any = {}) {
	        return new SpecialRules(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Enabled = this.convertValues(source["Enabled"], Value_bool_);
	        this.Rate = this.convertValues(source["Rate"], Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Decimal_);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_SpecialRules_ {
	    State: number;
	    Value: SpecialRules;
	    Citation: string;
	
	    static createFrom(source: any = {}) {
	        return new Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_SpecialRules_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.State = source["State"];
	        this.Value = this.convertValues(source["Value"], SpecialRules);
	        this.Citation = source["Citation"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ReSignRules {
	    Enabled: Value_bool_;
	    Premiums: Value_map_int_github_com_secureprospective_TheWarRoom_internal_leaguerules_Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Decimal__;
	
	    static createFrom(source: any = {}) {
	        return new ReSignRules(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Enabled = this.convertValues(source["Enabled"], Value_bool_);
	        this.Premiums = this.convertValues(source["Premiums"], Value_map_int_github_com_secureprospective_TheWarRoom_internal_leaguerules_Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Decimal__);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_ReSignRules_ {
	    State: number;
	    Value: ReSignRules;
	    Citation: string;
	
	    static createFrom(source: any = {}) {
	        return new Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_ReSignRules_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.State = source["State"];
	        this.Value = this.convertValues(source["Value"], ReSignRules);
	        this.Citation = source["Citation"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Value___int_ {
	    State: number;
	    Value: number[];
	    Citation: string;
	
	    static createFrom(source: any = {}) {
	        return new Value___int_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.State = source["State"];
	        this.Value = source["Value"];
	        this.Citation = source["Citation"];
	    }
	}
	export class Tender {
	    Price: Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Money_;
	    Rounds: Value___int_;
	
	    static createFrom(source: any = {}) {
	        return new Tender(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Price = this.convertValues(source["Price"], Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Money_);
	        this.Rounds = this.convertValues(source["Rounds"], Value___int_);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Value___github_com_secureprospective_TheWarRoom_internal_leaguerules_Tender_ {
	    State: number;
	    Value: Tender[];
	    Citation: string;
	
	    static createFrom(source: any = {}) {
	        return new Value___github_com_secureprospective_TheWarRoom_internal_leaguerules_Tender_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.State = source["State"];
	        this.Value = this.convertValues(source["Value"], Tender);
	        this.Citation = source["Citation"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class TenderRules {
	    Enabled: Value_bool_;
	    // Go type: Value___github_com_secureprospective_TheWarRoom_internal_leaguerules_Tender_
	    Prices: any;
	    PriorSalaryMultiplier: Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Decimal_;
	    MatchWindow: Value_time_Duration_;
	    CompensatoryAllowed: Value_bool_;
	
	    static createFrom(source: any = {}) {
	        return new TenderRules(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Enabled = this.convertValues(source["Enabled"], Value_bool_);
	        this.Prices = this.convertValues(source["Prices"], null);
	        this.PriorSalaryMultiplier = this.convertValues(source["PriorSalaryMultiplier"], Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Decimal_);
	        this.MatchWindow = this.convertValues(source["MatchWindow"], Value_time_Duration_);
	        this.CompensatoryAllowed = this.convertValues(source["CompensatoryAllowed"], Value_bool_);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_TenderRules_ {
	    State: number;
	    Value: TenderRules;
	    Citation: string;
	
	    static createFrom(source: any = {}) {
	        return new Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_TenderRules_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.State = source["State"];
	        this.Value = this.convertValues(source["Value"], TenderRules);
	        this.Citation = source["Citation"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_BuyoutRules_ {
	    State: number;
	    Value: BuyoutRules;
	    Citation: string;
	
	    static createFrom(source: any = {}) {
	        return new Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_BuyoutRules_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.State = source["State"];
	        this.Value = this.convertValues(source["Value"], BuyoutRules);
	        this.Citation = source["Citation"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class MoneyTier {
	    At: Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Money_;
	    Amount: Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Money_;
	
	    static createFrom(source: any = {}) {
	        return new MoneyTier(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.At = this.convertValues(source["At"], Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Money_);
	        this.Amount = this.convertValues(source["Amount"], Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Money_);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Value___github_com_secureprospective_TheWarRoom_internal_leaguerules_MoneyTier_ {
	    State: number;
	    Value: MoneyTier[];
	    Citation: string;
	
	    static createFrom(source: any = {}) {
	        return new Value___github_com_secureprospective_TheWarRoom_internal_leaguerules_MoneyTier_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.State = source["State"];
	        this.Value = this.convertValues(source["Value"], MoneyTier);
	        this.Citation = source["Citation"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class RestructureRules {
	    Enabled: Value_bool_;
	    AnnualLimit: Value_int_;
	    PerContract: Value_int_;
	    ExtraPerExtension: Value_int_;
	    // Go type: Value___github_com_secureprospective_TheWarRoom_internal_leaguerules_MoneyTier_
	    Tiers: any;
	    ConserveTotal: Value_bool_;
	    ExistingYearsOnly: Value_bool_;
	
	    static createFrom(source: any = {}) {
	        return new RestructureRules(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Enabled = this.convertValues(source["Enabled"], Value_bool_);
	        this.AnnualLimit = this.convertValues(source["AnnualLimit"], Value_int_);
	        this.PerContract = this.convertValues(source["PerContract"], Value_int_);
	        this.ExtraPerExtension = this.convertValues(source["ExtraPerExtension"], Value_int_);
	        this.Tiers = this.convertValues(source["Tiers"], null);
	        this.ConserveTotal = this.convertValues(source["ConserveTotal"], Value_bool_);
	        this.ExistingYearsOnly = this.convertValues(source["ExistingYearsOnly"], Value_bool_);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_RestructureRules_ {
	    State: number;
	    Value: RestructureRules;
	    Citation: string;
	
	    static createFrom(source: any = {}) {
	        return new Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_RestructureRules_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.State = source["State"];
	        this.Value = this.convertValues(source["Value"], RestructureRules);
	        this.Citation = source["Citation"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Value_map_github_com_secureprospective_TheWarRoom_internal_leaguerules_Position_github_com_secureprospective_TheWarRoom_internal_leaguerules_Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Money__ {
	    State: number;
	    Value: Record<string, Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Money_>;
	    Citation: string;
	
	    static createFrom(source: any = {}) {
	        return new Value_map_github_com_secureprospective_TheWarRoom_internal_leaguerules_Position_github_com_secureprospective_TheWarRoom_internal_leaguerules_Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Money__(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.State = source["State"];
	        this.Value = this.convertValues(source["Value"], Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Money_, true);
	        this.Citation = source["Citation"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ExtensionRules {
	    Enabled: Value_bool_;
	    AnnualLimit: Value_int_;
	    MinimumRemaining: Value_int_;
	    AddedYears: Value_int_;
	    TotalYears: Value_int_;
	    Multiplier: Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Decimal_;
	    Floors: Value_map_github_com_secureprospective_TheWarRoom_internal_leaguerules_Position_github_com_secureprospective_TheWarRoom_internal_leaguerules_Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Money__;
	    RepeatBeforeFreeAgency: Value_bool_;
	    UFAEligible: Value_bool_;
	
	    static createFrom(source: any = {}) {
	        return new ExtensionRules(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Enabled = this.convertValues(source["Enabled"], Value_bool_);
	        this.AnnualLimit = this.convertValues(source["AnnualLimit"], Value_int_);
	        this.MinimumRemaining = this.convertValues(source["MinimumRemaining"], Value_int_);
	        this.AddedYears = this.convertValues(source["AddedYears"], Value_int_);
	        this.TotalYears = this.convertValues(source["TotalYears"], Value_int_);
	        this.Multiplier = this.convertValues(source["Multiplier"], Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Decimal_);
	        this.Floors = this.convertValues(source["Floors"], Value_map_github_com_secureprospective_TheWarRoom_internal_leaguerules_Position_github_com_secureprospective_TheWarRoom_internal_leaguerules_Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Money__);
	        this.RepeatBeforeFreeAgency = this.convertValues(source["RepeatBeforeFreeAgency"], Value_bool_);
	        this.UFAEligible = this.convertValues(source["UFAEligible"], Value_bool_);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_ExtensionRules_ {
	    State: number;
	    Value: ExtensionRules;
	    Citation: string;
	
	    static createFrom(source: any = {}) {
	        return new Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_ExtensionRules_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.State = source["State"];
	        this.Value = this.convertValues(source["Value"], ExtensionRules);
	        this.Citation = source["Citation"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class TagRules {
	    Enabled: Value_bool_;
	    AnnualLimit: Value_int_;
	    TopPlayers: Value_int_;
	    FloorMultiplier: Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Decimal_;
	    ConsecutiveLimit: Value_int_;
	    SecondMultiplier: Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Decimal_;
	    ExtensionMultiplier: Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Decimal_;
	    ExcludePlayoffContracts: Value_bool_;
	
	    static createFrom(source: any = {}) {
	        return new TagRules(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Enabled = this.convertValues(source["Enabled"], Value_bool_);
	        this.AnnualLimit = this.convertValues(source["AnnualLimit"], Value_int_);
	        this.TopPlayers = this.convertValues(source["TopPlayers"], Value_int_);
	        this.FloorMultiplier = this.convertValues(source["FloorMultiplier"], Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Decimal_);
	        this.ConsecutiveLimit = this.convertValues(source["ConsecutiveLimit"], Value_int_);
	        this.SecondMultiplier = this.convertValues(source["SecondMultiplier"], Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Decimal_);
	        this.ExtensionMultiplier = this.convertValues(source["ExtensionMultiplier"], Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Decimal_);
	        this.ExcludePlayoffContracts = this.convertValues(source["ExcludePlayoffContracts"], Value_bool_);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_TagRules_ {
	    State: number;
	    Value: TagRules;
	    Citation: string;
	
	    static createFrom(source: any = {}) {
	        return new Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_TagRules_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.State = source["State"];
	        this.Value = this.convertValues(source["Value"], TagRules);
	        this.Citation = source["Citation"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ExperienceSalary {
	    MaximumUnlimited: Value_bool_;
	    Years: Bounds;
	    Minimum: Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Money_;
	
	    static createFrom(source: any = {}) {
	        return new ExperienceSalary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.MaximumUnlimited = this.convertValues(source["MaximumUnlimited"], Value_bool_);
	        this.Years = this.convertValues(source["Years"], Bounds);
	        this.Minimum = this.convertValues(source["Minimum"], Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Money_);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Value___github_com_secureprospective_TheWarRoom_internal_leaguerules_ExperienceSalary_ {
	    State: number;
	    Value: ExperienceSalary[];
	    Citation: string;
	
	    static createFrom(source: any = {}) {
	        return new Value___github_com_secureprospective_TheWarRoom_internal_leaguerules_ExperienceSalary_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.State = source["State"];
	        this.Value = this.convertValues(source["Value"], ExperienceSalary);
	        this.Citation = source["Citation"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ContractsRules {
	    MaximumYears: Value_int_;
	    FlatSalaries: Value_bool_;
	    // Go type: Value___github_com_secureprospective_TheWarRoom_internal_leaguerules_ExperienceSalary_
	    MinimumSalaries: any;
	    DeadCap: Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Decimal_;
	    RestructuredDeadCap: Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Decimal_;
	    ClaimClearsDeadCap: Value_bool_;
	    ReleaseReSignSameSeason: Value_bool_;
	    Tag: Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_TagRules_;
	    Extension: Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_ExtensionRules_;
	    Restructure: Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_RestructureRules_;
	    Buyout: Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_BuyoutRules_;
	    Tender: Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_TenderRules_;
	    ReSign: Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_ReSignRules_;
	    Retirement: Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_SpecialRules_;
	    Death: Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_SpecialRules_;
	    CapRelief: Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_ReliefRules_;
	
	    static createFrom(source: any = {}) {
	        return new ContractsRules(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.MaximumYears = this.convertValues(source["MaximumYears"], Value_int_);
	        this.FlatSalaries = this.convertValues(source["FlatSalaries"], Value_bool_);
	        this.MinimumSalaries = this.convertValues(source["MinimumSalaries"], null);
	        this.DeadCap = this.convertValues(source["DeadCap"], Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Decimal_);
	        this.RestructuredDeadCap = this.convertValues(source["RestructuredDeadCap"], Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Decimal_);
	        this.ClaimClearsDeadCap = this.convertValues(source["ClaimClearsDeadCap"], Value_bool_);
	        this.ReleaseReSignSameSeason = this.convertValues(source["ReleaseReSignSameSeason"], Value_bool_);
	        this.Tag = this.convertValues(source["Tag"], Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_TagRules_);
	        this.Extension = this.convertValues(source["Extension"], Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_ExtensionRules_);
	        this.Restructure = this.convertValues(source["Restructure"], Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_RestructureRules_);
	        this.Buyout = this.convertValues(source["Buyout"], Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_BuyoutRules_);
	        this.Tender = this.convertValues(source["Tender"], Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_TenderRules_);
	        this.ReSign = this.convertValues(source["ReSign"], Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_ReSignRules_);
	        this.Retirement = this.convertValues(source["Retirement"], Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_SpecialRules_);
	        this.Death = this.convertValues(source["Death"], Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_SpecialRules_);
	        this.CapRelief = this.convertValues(source["CapRelief"], Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_ReliefRules_);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DistanceScale {
	    From: number;
	    Base: number;
	    Pivot: number;
	    Above: number;
	    Below: number;
	    Capped: boolean;
	    Cap: number;
	
	    static createFrom(source: any = {}) {
	        return new DistanceScale(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.From = source["From"];
	        this.Base = source["Base"];
	        this.Pivot = source["Pivot"];
	        this.Above = source["Above"];
	        this.Below = source["Below"];
	        this.Capped = source["Capped"];
	        this.Cap = source["Cap"];
	    }
	}
	export class Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_TieRule_ {
	    State: number;
	    Value: string;
	    Citation: string;
	
	    static createFrom(source: any = {}) {
	        return new Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_TieRule_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.State = source["State"];
	        this.Value = source["Value"];
	        this.Citation = source["Citation"];
	    }
	}
	export class Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_OrderPattern_ {
	    State: number;
	    Value: string;
	    Citation: string;
	
	    static createFrom(source: any = {}) {
	        return new Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_OrderPattern_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.State = source["State"];
	        this.Value = source["Value"];
	        this.Citation = source["Citation"];
	    }
	}
	export class Value_map_int___string_ {
	    State: number;
	    Value: Record<number, Array<string>>;
	    Citation: string;
	
	    static createFrom(source: any = {}) {
	        return new Value_map_int___string_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.State = source["State"];
	        this.Value = source["Value"];
	        this.Citation = source["Citation"];
	    }
	}
	export class Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_OrderSource_ {
	    State: number;
	    Value: string;
	    Citation: string;
	
	    static createFrom(source: any = {}) {
	        return new Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_OrderSource_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.State = source["State"];
	        this.Value = source["Value"];
	        this.Citation = source["Citation"];
	    }
	}
	export class Value_map_int_github_com_secureprospective_TheWarRoom_internal_leaguerules_Value_bool__ {
	    State: number;
	    Value: Record<number, Value_bool_>;
	    Citation: string;
	
	    static createFrom(source: any = {}) {
	        return new Value_map_int_github_com_secureprospective_TheWarRoom_internal_leaguerules_Value_bool__(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.State = source["State"];
	        this.Value = this.convertValues(source["Value"], Value_bool_, true);
	        this.Citation = source["Citation"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Value_map_int_github_com_secureprospective_TheWarRoom_internal_leaguerules_Value_int__ {
	    State: number;
	    Value: Record<number, Value_int_>;
	    Citation: string;
	
	    static createFrom(source: any = {}) {
	        return new Value_map_int_github_com_secureprospective_TheWarRoom_internal_leaguerules_Value_int__(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.State = source["State"];
	        this.Value = this.convertValues(source["Value"], Value_int_, true);
	        this.Citation = source["Citation"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Value_map_int___github_com_secureprospective_TheWarRoom_internal_leaguerules_SlotPrice_ {
	    State: number;
	    Value: Record<number, Array<SlotPrice>>;
	    Citation: string;
	
	    static createFrom(source: any = {}) {
	        return new Value_map_int___github_com_secureprospective_TheWarRoom_internal_leaguerules_SlotPrice_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.State = source["State"];
	        this.Value = this.convertValues(source["Value"], Array<SlotPrice>, true);
	        this.Citation = source["Citation"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class TimeoutPolicy {
	    Unlimited: Value_bool_;
	    Count: Value_int_;
	
	    static createFrom(source: any = {}) {
	        return new TimeoutPolicy(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Unlimited = this.convertValues(source["Unlimited"], Value_bool_);
	        this.Count = this.convertValues(source["Count"], Value_int_);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_TimeoutPolicy_ {
	    State: number;
	    Value: TimeoutPolicy;
	    Citation: string;
	
	    static createFrom(source: any = {}) {
	        return new Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_TimeoutPolicy_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.State = source["State"];
	        this.Value = this.convertValues(source["Value"], TimeoutPolicy);
	        this.Citation = source["Citation"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_ExpiryAction_ {
	    State: number;
	    Value: string;
	    Citation: string;
	
	    static createFrom(source: any = {}) {
	        return new Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_ExpiryAction_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.State = source["State"];
	        this.Value = source["Value"];
	        this.Citation = source["Citation"];
	    }
	}
	export class Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_DraftMode_ {
	    State: number;
	    Value: string;
	    Citation: string;
	
	    static createFrom(source: any = {}) {
	        return new Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_DraftMode_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.State = source["State"];
	        this.Value = source["Value"];
	        this.Citation = source["Citation"];
	    }
	}
	export class DraftRules {
	    Mode: Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_DraftMode_;
	    Clock: Value_time_Duration_;
	    OnExpire: Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_ExpiryAction_;
	    Timeouts: Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_TimeoutPolicy_;
	    Rounds: Value_int_;
	    MaximumPicks: Value_int_;
	    ForfeitExtra: Value_bool_;
	    // Go type: Value_map_int___github_com_secureprospective_TheWarRoom_internal_leaguerules_SlotPrice_
	    Pricing: any;
	    Lengths: Value_map_int_github_com_secureprospective_TheWarRoom_internal_leaguerules_Value_int__;
	    RFAAfter: Value_map_int_github_com_secureprospective_TheWarRoom_internal_leaguerules_Value_bool__;
	    OrderSource: Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_OrderSource_;
	    // Go type: Value_map_int___string_
	    Order: any;
	    LaterPattern: Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_OrderPattern_;
	    TieRule: Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_TieRule_;
	
	    static createFrom(source: any = {}) {
	        return new DraftRules(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Mode = this.convertValues(source["Mode"], Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_DraftMode_);
	        this.Clock = this.convertValues(source["Clock"], Value_time_Duration_);
	        this.OnExpire = this.convertValues(source["OnExpire"], Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_ExpiryAction_);
	        this.Timeouts = this.convertValues(source["Timeouts"], Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_TimeoutPolicy_);
	        this.Rounds = this.convertValues(source["Rounds"], Value_int_);
	        this.MaximumPicks = this.convertValues(source["MaximumPicks"], Value_int_);
	        this.ForfeitExtra = this.convertValues(source["ForfeitExtra"], Value_bool_);
	        this.Pricing = this.convertValues(source["Pricing"], null);
	        this.Lengths = this.convertValues(source["Lengths"], Value_map_int_github_com_secureprospective_TheWarRoom_internal_leaguerules_Value_int__);
	        this.RFAAfter = this.convertValues(source["RFAAfter"], Value_map_int_github_com_secureprospective_TheWarRoom_internal_leaguerules_Value_bool__);
	        this.OrderSource = this.convertValues(source["OrderSource"], Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_OrderSource_);
	        this.Order = this.convertValues(source["Order"], null);
	        this.LaterPattern = this.convertValues(source["LaterPattern"], Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_OrderPattern_);
	        this.TieRule = this.convertValues(source["TieRule"], Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_TieRule_);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	export class Value_map_string_string_ {
	    State: number;
	    Value: Record<string, string>;
	    Citation: string;
	
	    static createFrom(source: any = {}) {
	        return new Value_map_string_string_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.State = source["State"];
	        this.Value = source["Value"];
	        this.Citation = source["Citation"];
	    }
	}
	export class HouseRules {
	    Texts: Value_map_string_string_;
	
	    static createFrom(source: any = {}) {
	        return new HouseRules(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Texts = this.convertValues(source["Texts"], Value_map_string_string_);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Value___github_com_secureprospective_TheWarRoom_internal_leaguerules_NFLStatus_ {
	    State: number;
	    Value: string[];
	    Citation: string;
	
	    static createFrom(source: any = {}) {
	        return new Value___github_com_secureprospective_TheWarRoom_internal_leaguerules_NFLStatus_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.State = source["State"];
	        this.Value = source["Value"];
	        this.Citation = source["Citation"];
	    }
	}
	export class IRRules {
	    Slots: Value_int_;
	    // Go type: Value___github_com_secureprospective_TheWarRoom_internal_leaguerules_NFLStatus_
	    Eligible: any;
	    MustLeaveHealthy: Value_bool_;
	
	    static createFrom(source: any = {}) {
	        return new IRRules(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Slots = this.convertValues(source["Slots"], Value_int_);
	        this.Eligible = this.convertValues(source["Eligible"], null);
	        this.MustLeaveHealthy = this.convertValues(source["MustLeaveHealthy"], Value_bool_);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_CarryPolicy_ {
	    State: number;
	    Value: string;
	    Citation: string;
	
	    static createFrom(source: any = {}) {
	        return new Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_CarryPolicy_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.State = source["State"];
	        this.Value = source["Value"];
	        this.Citation = source["Citation"];
	    }
	}
	export class Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_InitialRosters_ {
	    State: number;
	    Value: string;
	    Citation: string;
	
	    static createFrom(source: any = {}) {
	        return new Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_InitialRosters_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.State = source["State"];
	        this.Value = source["Value"];
	        this.Citation = source["Citation"];
	    }
	}
	export class Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Format_ {
	    State: number;
	    Value: string;
	    Citation: string;
	
	    static createFrom(source: any = {}) {
	        return new Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Format_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.State = source["State"];
	        this.Value = source["Value"];
	        this.Citation = source["Citation"];
	    }
	}
	export class LeagueRules {
	    Teams: Value_int_;
	    Conferences: Value_int_;
	    Divisions: Value_int_;
	    Format: Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Format_;
	    InitialRosters: Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_InitialRosters_;
	    Alignment: Value_map_string_string_;
	    Dynasty: Value_bool_;
	    Carry: Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_CarryPolicy_;
	    Keepers: Value_int_;
	
	    static createFrom(source: any = {}) {
	        return new LeagueRules(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Teams = this.convertValues(source["Teams"], Value_int_);
	        this.Conferences = this.convertValues(source["Conferences"], Value_int_);
	        this.Divisions = this.convertValues(source["Divisions"], Value_int_);
	        this.Format = this.convertValues(source["Format"], Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Format_);
	        this.InitialRosters = this.convertValues(source["InitialRosters"], Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_InitialRosters_);
	        this.Alignment = this.convertValues(source["Alignment"], Value_map_string_string_);
	        this.Dynasty = this.convertValues(source["Dynasty"], Value_bool_);
	        this.Carry = this.convertValues(source["Carry"], Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_CarryPolicy_);
	        this.Keepers = this.convertValues(source["Keepers"], Value_int_);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class LineUnit {
	    Used: boolean;
	    Weights: Record<string, number>;
	    Labels: Record<string, string>;
	    NoSack: number;
	    Sacks: string;
	    Clamped: boolean;
	    Clamp: number;
	
	    static createFrom(source: any = {}) {
	        return new LineUnit(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Used = source["Used"];
	        this.Weights = source["Weights"];
	        this.Labels = source["Labels"];
	        this.NoSack = source["NoSack"];
	        this.Sacks = source["Sacks"];
	        this.Clamped = source["Clamped"];
	        this.Clamp = source["Clamp"];
	    }
	}
	export class Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Penalty_ {
	    State: number;
	    Value: string;
	    Citation: string;
	
	    static createFrom(source: any = {}) {
	        return new Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Penalty_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.State = source["State"];
	        this.Value = source["Value"];
	        this.Citation = source["Citation"];
	    }
	}
	export class Value_map_github_com_secureprospective_TheWarRoom_internal_leaguerules_LineupGroup_github_com_secureprospective_TheWarRoom_internal_leaguerules_Bounds_ {
	    State: number;
	    Value: Record<string, Bounds>;
	    Citation: string;
	
	    static createFrom(source: any = {}) {
	        return new Value_map_github_com_secureprospective_TheWarRoom_internal_leaguerules_LineupGroup_github_com_secureprospective_TheWarRoom_internal_leaguerules_Bounds_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.State = source["State"];
	        this.Value = this.convertValues(source["Value"], Bounds, true);
	        this.Citation = source["Citation"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Value_map_github_com_secureprospective_TheWarRoom_internal_leaguerules_Position_github_com_secureprospective_TheWarRoom_internal_leaguerules_Bounds_ {
	    State: number;
	    Value: Record<string, Bounds>;
	    Citation: string;
	
	    static createFrom(source: any = {}) {
	        return new Value_map_github_com_secureprospective_TheWarRoom_internal_leaguerules_Position_github_com_secureprospective_TheWarRoom_internal_leaguerules_Bounds_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.State = source["State"];
	        this.Value = this.convertValues(source["Value"], Bounds, true);
	        this.Citation = source["Citation"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class LineupRules {
	    Positions: Value_map_github_com_secureprospective_TheWarRoom_internal_leaguerules_Position_github_com_secureprospective_TheWarRoom_internal_leaguerules_Bounds_;
	    Groups: Value_map_github_com_secureprospective_TheWarRoom_internal_leaguerules_LineupGroup_github_com_secureprospective_TheWarRoom_internal_leaguerules_Bounds_;
	    InvalidPenalty: Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Penalty_;
	    MissedFlag: Value_bool_;
	
	    static createFrom(source: any = {}) {
	        return new LineupRules(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Positions = this.convertValues(source["Positions"], Value_map_github_com_secureprospective_TheWarRoom_internal_leaguerules_Position_github_com_secureprospective_TheWarRoom_internal_leaguerules_Bounds_);
	        this.Groups = this.convertValues(source["Groups"], Value_map_github_com_secureprospective_TheWarRoom_internal_leaguerules_LineupGroup_github_com_secureprospective_TheWarRoom_internal_leaguerules_Bounds_);
	        this.InvalidPenalty = this.convertValues(source["InvalidPenalty"], Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Penalty_);
	        this.MissedFlag = this.convertValues(source["MissedFlag"], Value_bool_);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	export class NotifyRules {
	    Transactions: Value_bool_;
	    Reminders: Value_bool_;
	
	    static createFrom(source: any = {}) {
	        return new NotifyRules(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Transactions = this.convertValues(source["Transactions"], Value_bool_);
	        this.Reminders = this.convertValues(source["Reminders"], Value_bool_);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Value___github_com_secureprospective_TheWarRoom_internal_leaguerules_BracketNode_ {
	    State: number;
	    Value: BracketNode[];
	    Citation: string;
	
	    static createFrom(source: any = {}) {
	        return new Value___github_com_secureprospective_TheWarRoom_internal_leaguerules_BracketNode_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.State = source["State"];
	        this.Value = this.convertValues(source["Value"], BracketNode);
	        this.Citation = source["Citation"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_SeedPolicy_ {
	    State: number;
	    Value: string;
	    Citation: string;
	
	    static createFrom(source: any = {}) {
	        return new Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_SeedPolicy_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.State = source["State"];
	        this.Value = source["Value"];
	        this.Citation = source["Citation"];
	    }
	}
	export class PlayoffsRules {
	    PerConference: Value_int_;
	    DivisionWinners: Value_int_;
	    WildCards: Value_int_;
	    ByeSeed: Value_int_;
	    Seeding: Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_SeedPolicy_;
	    Reseed: Value_bool_;
	    StartWeek: Value_int_;
	    FinalWeek: Value_int_;
	    HomeBonus: Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Decimal_;
	    // Go type: Value___github_com_secureprospective_TheWarRoom_internal_leaguerules_BracketNode_
	    Graph: any;
	
	    static createFrom(source: any = {}) {
	        return new PlayoffsRules(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.PerConference = this.convertValues(source["PerConference"], Value_int_);
	        this.DivisionWinners = this.convertValues(source["DivisionWinners"], Value_int_);
	        this.WildCards = this.convertValues(source["WildCards"], Value_int_);
	        this.ByeSeed = this.convertValues(source["ByeSeed"], Value_int_);
	        this.Seeding = this.convertValues(source["Seeding"], Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_SeedPolicy_);
	        this.Reseed = this.convertValues(source["Reseed"], Value_bool_);
	        this.StartWeek = this.convertValues(source["StartWeek"], Value_int_);
	        this.FinalWeek = this.convertValues(source["FinalWeek"], Value_int_);
	        this.HomeBonus = this.convertValues(source["HomeBonus"], Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Decimal_);
	        this.Graph = this.convertValues(source["Graph"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class StatRange {
	    Min: number;
	    Max: number;
	
	    static createFrom(source: any = {}) {
	        return new StatRange(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Min = source["Min"];
	        this.Max = source["Max"];
	    }
	}
	export class PositionExtra {
	    Positions: string[];
	    Points: number;
	    Ranged: boolean;
	    Range: StatRange;
	
	    static createFrom(source: any = {}) {
	        return new PositionExtra(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Positions = source["Positions"];
	        this.Points = source["Points"];
	        this.Ranged = source["Ranged"];
	        this.Range = this.convertValues(source["Range"], StatRange);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	
	export class Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Bounds_ {
	    State: number;
	    Value: Bounds;
	    Citation: string;
	
	    static createFrom(source: any = {}) {
	        return new Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Bounds_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.State = source["State"];
	        this.Value = this.convertValues(source["Value"], Bounds);
	        this.Citation = source["Citation"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class RosterRules {
	    Limits: Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Bounds_;
	    ReserveExcluded: Value_bool_;
	    PositionBounds: Value_map_github_com_secureprospective_TheWarRoom_internal_leaguerules_Position_github_com_secureprospective_TheWarRoom_internal_leaguerules_Bounds_;
	
	    static createFrom(source: any = {}) {
	        return new RosterRules(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Limits = this.convertValues(source["Limits"], Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Bounds_);
	        this.ReserveExcluded = this.convertValues(source["ReserveExcluded"], Value_bool_);
	        this.PositionBounds = this.convertValues(source["PositionBounds"], Value_map_github_com_secureprospective_TheWarRoom_internal_leaguerules_Position_github_com_secureprospective_TheWarRoom_internal_leaguerules_Bounds_);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_SchedulePattern_ {
	    State: number;
	    Value: string;
	    Citation: string;
	
	    static createFrom(source: any = {}) {
	        return new Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_SchedulePattern_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.State = source["State"];
	        this.Value = source["Value"];
	        this.Citation = source["Citation"];
	    }
	}
	export class ScheduleRules {
	    RegularWeeks: Value_int_;
	    Pattern: Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_SchedulePattern_;
	
	    static createFrom(source: any = {}) {
	        return new ScheduleRules(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.RegularWeeks = this.convertValues(source["RegularWeeks"], Value_int_);
	        this.Pattern = this.convertValues(source["Pattern"], Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_SchedulePattern_);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_ReviewRoute_ {
	    State: number;
	    Value: string;
	    Citation: string;
	
	    static createFrom(source: any = {}) {
	        return new Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_ReviewRoute_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.State = source["State"];
	        this.Value = source["Value"];
	        this.Citation = source["Citation"];
	    }
	}
	export class TradesRules {
	    Review: Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_ReviewRoute_;
	    DOTSeats: Value_int_;
	    DecideAt: Value_int_;
	    FutureYears: Value_int_;
	    Rounds: Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Bounds_;
	    RosterGate: Value_bool_;
	    OfferExpiry: Value_time_Duration_;
	    MaximumParties: Value_int_;
	    CompensatoryPicks: Value_bool_;
	    CapSpace: Value_bool_;
	    AcceptedWithdrawal: Value_bool_;
	
	    static createFrom(source: any = {}) {
	        return new TradesRules(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Review = this.convertValues(source["Review"], Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_ReviewRoute_);
	        this.DOTSeats = this.convertValues(source["DOTSeats"], Value_int_);
	        this.DecideAt = this.convertValues(source["DecideAt"], Value_int_);
	        this.FutureYears = this.convertValues(source["FutureYears"], Value_int_);
	        this.Rounds = this.convertValues(source["Rounds"], Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Bounds_);
	        this.RosterGate = this.convertValues(source["RosterGate"], Value_bool_);
	        this.OfferExpiry = this.convertValues(source["OfferExpiry"], Value_time_Duration_);
	        this.MaximumParties = this.convertValues(source["MaximumParties"], Value_int_);
	        this.CompensatoryPicks = this.convertValues(source["CompensatoryPicks"], Value_bool_);
	        this.CapSpace = this.convertValues(source["CapSpace"], Value_bool_);
	        this.AcceptedWithdrawal = this.convertValues(source["AcceptedWithdrawal"], Value_bool_);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_WaiverMethod_ {
	    State: number;
	    Value: string;
	    Citation: string;
	
	    static createFrom(source: any = {}) {
	        return new Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_WaiverMethod_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.State = source["State"];
	        this.Value = source["Value"];
	        this.Citation = source["Citation"];
	    }
	}
	export class Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_WaiverTrigger_ {
	    State: number;
	    Value: string;
	    Citation: string;
	
	    static createFrom(source: any = {}) {
	        return new Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_WaiverTrigger_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.State = source["State"];
	        this.Value = source["Value"];
	        this.Citation = source["Citation"];
	    }
	}
	export class WaiversRules {
	    Trigger: Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_WaiverTrigger_;
	    Duration: Value_time_Duration_;
	    Method: Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_WaiverMethod_;
	
	    static createFrom(source: any = {}) {
	        return new WaiversRules(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Trigger = this.convertValues(source["Trigger"], Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_WaiverTrigger_);
	        this.Duration = this.convertValues(source["Duration"], Value_time_Duration_);
	        this.Method = this.convertValues(source["Method"], Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_WaiverMethod_);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_ExperienceBasis_ {
	    State: number;
	    Value: string;
	    Citation: string;
	
	    static createFrom(source: any = {}) {
	        return new Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_ExperienceBasis_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.State = source["State"];
	        this.Value = source["Value"];
	        this.Citation = source["Citation"];
	    }
	}
	export class TaxiRules {
	    Slots: Value_int_;
	    MaximumYears: Value_int_;
	    Basis: Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_ExperienceBasis_;
	
	    static createFrom(source: any = {}) {
	        return new TaxiRules(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Slots = this.convertValues(source["Slots"], Value_int_);
	        this.MaximumYears = this.convertValues(source["MaximumYears"], Value_int_);
	        this.Basis = this.convertValues(source["Basis"], Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_ExperienceBasis_);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Value___github_com_secureprospective_TheWarRoom_internal_leaguerules_Criterion_ {
	    State: number;
	    Value: string[];
	    Citation: string;
	
	    static createFrom(source: any = {}) {
	        return new Value___github_com_secureprospective_TheWarRoom_internal_leaguerules_Criterion_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.State = source["State"];
	        this.Value = source["Value"];
	        this.Citation = source["Citation"];
	    }
	}
	export class StandingsRules {
	    // Go type: Value___github_com_secureprospective_TheWarRoom_internal_leaguerules_Criterion_
	    Criteria: any;
	
	    static createFrom(source: any = {}) {
	        return new StandingsRules(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Criteria = this.convertValues(source["Criteria"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_LineUnit_ {
	    State: number;
	    Value: LineUnit;
	    Citation: string;
	
	    static createFrom(source: any = {}) {
	        return new Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_LineUnit_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.State = source["State"];
	        this.Value = this.convertValues(source["Value"], LineUnit);
	        this.Citation = source["Citation"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Value_map_github_com_secureprospective_TheWarRoom_internal_leaguerules_StatID___github_com_secureprospective_TheWarRoom_internal_leaguerules_StatID_ {
	    State: number;
	    Value: Record<string, Array<string>>;
	    Citation: string;
	
	    static createFrom(source: any = {}) {
	        return new Value_map_github_com_secureprospective_TheWarRoom_internal_leaguerules_StatID___github_com_secureprospective_TheWarRoom_internal_leaguerules_StatID_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.State = source["State"];
	        this.Value = source["Value"];
	        this.Citation = source["Citation"];
	    }
	}
	export class Value_map_github_com_secureprospective_TheWarRoom_internal_leaguerules_StatID_github_com_secureprospective_TheWarRoom_internal_leaguerules_DistanceScale_ {
	    State: number;
	    Value: Record<string, DistanceScale>;
	    Citation: string;
	
	    static createFrom(source: any = {}) {
	        return new Value_map_github_com_secureprospective_TheWarRoom_internal_leaguerules_StatID_github_com_secureprospective_TheWarRoom_internal_leaguerules_DistanceScale_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.State = source["State"];
	        this.Value = this.convertValues(source["Value"], DistanceScale, true);
	        this.Citation = source["Citation"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Value_map_github_com_secureprospective_TheWarRoom_internal_leaguerules_StatID_github_com_secureprospective_TheWarRoom_internal_leaguerules_StatRange_ {
	    State: number;
	    Value: Record<string, StatRange>;
	    Citation: string;
	
	    static createFrom(source: any = {}) {
	        return new Value_map_github_com_secureprospective_TheWarRoom_internal_leaguerules_StatID_github_com_secureprospective_TheWarRoom_internal_leaguerules_StatRange_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.State = source["State"];
	        this.Value = this.convertValues(source["Value"], StatRange, true);
	        this.Citation = source["Citation"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Value_map_github_com_secureprospective_TheWarRoom_internal_leaguerules_StatID___github_com_secureprospective_TheWarRoom_internal_leaguerules_PositionExtra_ {
	    State: number;
	    Value: Record<string, Array<PositionExtra>>;
	    Citation: string;
	
	    static createFrom(source: any = {}) {
	        return new Value_map_github_com_secureprospective_TheWarRoom_internal_leaguerules_StatID___github_com_secureprospective_TheWarRoom_internal_leaguerules_PositionExtra_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.State = source["State"];
	        this.Value = this.convertValues(source["Value"], Array<PositionExtra>, true);
	        this.Citation = source["Citation"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Value___github_com_secureprospective_TheWarRoom_internal_leaguerules_Position_ {
	    State: number;
	    Value: string[];
	    Citation: string;
	
	    static createFrom(source: any = {}) {
	        return new Value___github_com_secureprospective_TheWarRoom_internal_leaguerules_Position_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.State = source["State"];
	        this.Value = source["Value"];
	        this.Citation = source["Citation"];
	    }
	}
	export class Value___github_com_secureprospective_TheWarRoom_internal_leaguerules_Bonus_ {
	    State: number;
	    Value: Bonus[];
	    Citation: string;
	
	    static createFrom(source: any = {}) {
	        return new Value___github_com_secureprospective_TheWarRoom_internal_leaguerules_Bonus_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.State = source["State"];
	        this.Value = this.convertValues(source["Value"], Bonus);
	        this.Citation = source["Citation"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Stat {
	    Label: Value_string_;
	    Group: Value_string_;
	    Used: Value_bool_;
	    Points: Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Decimal_;
	    // Go type: Value___github_com_secureprospective_TheWarRoom_internal_leaguerules_Bonus_
	    Bonuses: any;
	    // Go type: Value___github_com_secureprospective_TheWarRoom_internal_leaguerules_Position_
	    Positions: any;
	    Once: Value_bool_;
	
	    static createFrom(source: any = {}) {
	        return new Stat(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Label = this.convertValues(source["Label"], Value_string_);
	        this.Group = this.convertValues(source["Group"], Value_string_);
	        this.Used = this.convertValues(source["Used"], Value_bool_);
	        this.Points = this.convertValues(source["Points"], Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Decimal_);
	        this.Bonuses = this.convertValues(source["Bonuses"], null);
	        this.Positions = this.convertValues(source["Positions"], null);
	        this.Once = this.convertValues(source["Once"], Value_bool_);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Value_map_github_com_secureprospective_TheWarRoom_internal_leaguerules_StatID_github_com_secureprospective_TheWarRoom_internal_leaguerules_Stat_ {
	    State: number;
	    Value: Record<string, Stat>;
	    Citation: string;
	
	    static createFrom(source: any = {}) {
	        return new Value_map_github_com_secureprospective_TheWarRoom_internal_leaguerules_StatID_github_com_secureprospective_TheWarRoom_internal_leaguerules_Stat_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.State = source["State"];
	        this.Value = this.convertValues(source["Value"], Stat, true);
	        this.Citation = source["Citation"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ScoringRules {
	    Catalogue: Value_map_github_com_secureprospective_TheWarRoom_internal_leaguerules_StatID_github_com_secureprospective_TheWarRoom_internal_leaguerules_Stat_;
	    RoundStatPoints: Value_bool_;
	    // Go type: Value_map_github_com_secureprospective_TheWarRoom_internal_leaguerules_StatID___github_com_secureprospective_TheWarRoom_internal_leaguerules_PositionExtra_
	    Extras: any;
	    Ranges: Value_map_github_com_secureprospective_TheWarRoom_internal_leaguerules_StatID_github_com_secureprospective_TheWarRoom_internal_leaguerules_StatRange_;
	    Scales: Value_map_github_com_secureprospective_TheWarRoom_internal_leaguerules_StatID_github_com_secureprospective_TheWarRoom_internal_leaguerules_DistanceScale_;
	    // Go type: Value_map_github_com_secureprospective_TheWarRoom_internal_leaguerules_StatID___github_com_secureprospective_TheWarRoom_internal_leaguerules_StatID_
	    Totals: any;
	    LineUnit: Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_LineUnit_;
	
	    static createFrom(source: any = {}) {
	        return new ScoringRules(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Catalogue = this.convertValues(source["Catalogue"], Value_map_github_com_secureprospective_TheWarRoom_internal_leaguerules_StatID_github_com_secureprospective_TheWarRoom_internal_leaguerules_Stat_);
	        this.RoundStatPoints = this.convertValues(source["RoundStatPoints"], Value_bool_);
	        this.Extras = this.convertValues(source["Extras"], null);
	        this.Ranges = this.convertValues(source["Ranges"], Value_map_github_com_secureprospective_TheWarRoom_internal_leaguerules_StatID_github_com_secureprospective_TheWarRoom_internal_leaguerules_StatRange_);
	        this.Scales = this.convertValues(source["Scales"], Value_map_github_com_secureprospective_TheWarRoom_internal_leaguerules_StatID_github_com_secureprospective_TheWarRoom_internal_leaguerules_DistanceScale_);
	        this.Totals = this.convertValues(source["Totals"], null);
	        this.LineUnit = this.convertValues(source["LineUnit"], Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_LineUnit_);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class RuleSet {
	    LeagueID: string;
	    Season: number;
	    League: LeagueRules;
	    Roster: RosterRules;
	    Lineup: LineupRules;
	    Scoring: ScoringRules;
	    Standings: StandingsRules;
	    Cap: CapRules;
	    Contracts: ContractsRules;
	    IR: IRRules;
	    Taxi: TaxiRules;
	    Waivers: WaiversRules;
	    Bids: BidsRules;
	    Trades: TradesRules;
	    Schedule: ScheduleRules;
	    Playoffs: PlayoffsRules;
	    House: HouseRules;
	    Calendar: CalendarRules;
	    Draft: DraftRules;
	    Notify: NotifyRules;
	
	    static createFrom(source: any = {}) {
	        return new RuleSet(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.LeagueID = source["LeagueID"];
	        this.Season = source["Season"];
	        this.League = this.convertValues(source["League"], LeagueRules);
	        this.Roster = this.convertValues(source["Roster"], RosterRules);
	        this.Lineup = this.convertValues(source["Lineup"], LineupRules);
	        this.Scoring = this.convertValues(source["Scoring"], ScoringRules);
	        this.Standings = this.convertValues(source["Standings"], StandingsRules);
	        this.Cap = this.convertValues(source["Cap"], CapRules);
	        this.Contracts = this.convertValues(source["Contracts"], ContractsRules);
	        this.IR = this.convertValues(source["IR"], IRRules);
	        this.Taxi = this.convertValues(source["Taxi"], TaxiRules);
	        this.Waivers = this.convertValues(source["Waivers"], WaiversRules);
	        this.Bids = this.convertValues(source["Bids"], BidsRules);
	        this.Trades = this.convertValues(source["Trades"], TradesRules);
	        this.Schedule = this.convertValues(source["Schedule"], ScheduleRules);
	        this.Playoffs = this.convertValues(source["Playoffs"], PlayoffsRules);
	        this.House = this.convertValues(source["House"], HouseRules);
	        this.Calendar = this.convertValues(source["Calendar"], CalendarRules);
	        this.Draft = this.convertValues(source["Draft"], DraftRules);
	        this.Notify = this.convertValues(source["Notify"], NotifyRules);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class ValueSchema {
	    kind: string;
	    minimum?: number;
	    maximum?: number;
	    choices: Choice[];
	    fields: SchemaField[];
	    key?: ValueSchema;
	    element?: ValueSchema;
	
	    static createFrom(source: any = {}) {
	        return new ValueSchema(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.minimum = source["minimum"];
	        this.maximum = source["maximum"];
	        this.choices = this.convertValues(source["choices"], Choice);
	        this.fields = this.convertValues(source["fields"], SchemaField);
	        this.key = this.convertValues(source["key"], ValueSchema);
	        this.element = this.convertValues(source["element"], ValueSchema);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SchemaField {
	    name: string;
	    label: string;
	    schema: ValueSchema;
	
	    static createFrom(source: any = {}) {
	        return new SchemaField(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.label = source["label"];
	        this.schema = this.convertValues(source["schema"], ValueSchema);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class SlotPrice {
	    Slots: Bounds;
	    Salary: Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Money_;
	
	    static createFrom(source: any = {}) {
	        return new SlotPrice(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Slots = this.convertValues(source["Slots"], Bounds);
	        this.Salary = this.convertValues(source["Salary"], Value_github_com_secureprospective_TheWarRoom_internal_leaguerules_Money_);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	

}

export namespace lineup {
	
	export class Problem {
	    subject: string;
	    kind: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new Problem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.subject = source["subject"];
	        this.kind = source["kind"];
	        this.message = source["message"];
	    }
	}
	export class Result {
	    full: boolean;
	    legal: boolean;
	    problems: Problem[];
	
	    static createFrom(source: any = {}) {
	        return new Result(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.full = source["full"];
	        this.legal = source["legal"];
	        this.problems = this.convertValues(source["problems"], Problem);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace livegame {
	
	export class BoxRow {
	    id?: string;
	    name: string;
	    stats: string[];
	
	    static createFrom(source: any = {}) {
	        return new BoxRow(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.stats = source["stats"];
	    }
	}
	export class Box {
	    team: string;
	    category: string;
	    labels: string[];
	    rows: BoxRow[];
	
	    static createFrom(source: any = {}) {
	        return new Box(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.team = source["team"];
	        this.category = source["category"];
	        this.labels = source["labels"];
	        this.rows = this.convertValues(source["rows"], BoxRow);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class Play {
	    id: string;
	    sequence: number;
	    text: string;
	    clock: string;
	    period: number;
	    at: string;
	    homeScore: number;
	    awayScore: number;
	    start?: State;
	    end?: State;
	    participants: string[];
	
	    static createFrom(source: any = {}) {
	        return new Play(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.sequence = source["sequence"];
	        this.text = source["text"];
	        this.clock = source["clock"];
	        this.period = source["period"];
	        this.at = source["at"];
	        this.homeScore = source["homeScore"];
	        this.awayScore = source["awayScore"];
	        this.start = this.convertValues(source["start"], State);
	        this.end = this.convertValues(source["end"], State);
	        this.participants = source["participants"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class State {
	    possession: string;
	    down: number;
	    distance: number;
	    yardLine: number;
	    toEndzone: number;
	
	    static createFrom(source: any = {}) {
	        return new State(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.possession = source["possession"];
	        this.down = source["down"];
	        this.distance = source["distance"];
	        this.yardLine = source["yardLine"];
	        this.toEndzone = source["toEndzone"];
	    }
	}
	export class Team {
	    id: string;
	    abbreviation: string;
	    score: number;
	
	    static createFrom(source: any = {}) {
	        return new Team(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.abbreviation = source["abbreviation"];
	        this.score = source["score"];
	    }
	}
	export class Game {
	    id: string;
	    kickoff: string;
	    home: Team;
	    away: Team;
	    status: string;
	    clock: string;
	    period: number;
	    state?: State;
	    last?: Play;
	    box: Box[];
	    plays: Play[];
	
	    static createFrom(source: any = {}) {
	        return new Game(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.kickoff = source["kickoff"];
	        this.home = this.convertValues(source["home"], Team);
	        this.away = this.convertValues(source["away"], Team);
	        this.status = source["status"];
	        this.clock = source["clock"];
	        this.period = source["period"];
	        this.state = this.convertValues(source["state"], State);
	        this.last = this.convertValues(source["last"], Play);
	        this.box = this.convertValues(source["box"], Box);
	        this.plays = this.convertValues(source["plays"], Play);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	

}

export namespace market {
	
	export class AssetCell {
	    text: string;
	    sub: string;
	    detail: string;
	    held: boolean;
	
	    static createFrom(source: any = {}) {
	        return new AssetCell(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.text = source["text"];
	        this.sub = source["sub"];
	        this.detail = source["detail"];
	        this.held = source["held"];
	    }
	}
	export class AssetValue {
	    id: string;
	    now: AssetCell;
	    dynasty: AssetCell;
	    market: AssetCell;
	    nowValue: number;
	    dynastyValue: number;
	
	    static createFrom(source: any = {}) {
	        return new AssetValue(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.now = this.convertValues(source["now"], AssetCell);
	        this.dynasty = this.convertValues(source["dynasty"], AssetCell);
	        this.market = this.convertValues(source["market"], AssetCell);
	        this.nowValue = source["nowValue"];
	        this.dynastyValue = source["dynastyValue"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class AssetBoard {
	    values: AssetValue[];
	    nowSource: string;
	    dynastySource: string;
	    how: string;
	
	    static createFrom(source: any = {}) {
	        return new AssetBoard(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.values = this.convertValues(source["values"], AssetValue);
	        this.nowSource = source["nowSource"];
	        this.dynastySource = source["dynastySource"];
	        this.how = source["how"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	export class Choice {
	    id: string;
	    words: string;
	
	    static createFrom(source: any = {}) {
	        return new Choice(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.words = source["words"];
	    }
	}
	export class ContractCapEffect {
	    beforeCents?: number;
	    afterCents?: number;
	    changeCents?: number;
	    roomAfterCents?: number;
	    words: string;
	
	    static createFrom(source: any = {}) {
	        return new ContractCapEffect(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.beforeCents = source["beforeCents"];
	        this.afterCents = source["afterCents"];
	        this.changeCents = source["changeCents"];
	        this.roomAfterCents = source["roomAfterCents"];
	        this.words = source["words"];
	    }
	}
	export class ContractQuote {
	    league: string;
	    revision: number;
	    season: number;
	    player: string;
	    name: string;
	    franchise: string;
	    action: string;
	    actionName: string;
	    allowed: boolean;
	    reason: string;
	    needs: string[];
	    years: number;
	    salaryCents?: number;
	    salaries: nativecheck.SeasonAmount[];
	    deadMoney: nativecheck.SeasonAmount[];
	    reliefCents: number;
	    cap?: ContractCapEffect;
	    basis: string[];
	    notes: string[];
	    source: string;
	
	    static createFrom(source: any = {}) {
	        return new ContractQuote(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.league = source["league"];
	        this.revision = source["revision"];
	        this.season = source["season"];
	        this.player = source["player"];
	        this.name = source["name"];
	        this.franchise = source["franchise"];
	        this.action = source["action"];
	        this.actionName = source["actionName"];
	        this.allowed = source["allowed"];
	        this.reason = source["reason"];
	        this.needs = source["needs"];
	        this.years = source["years"];
	        this.salaryCents = source["salaryCents"];
	        this.salaries = this.convertValues(source["salaries"], nativecheck.SeasonAmount);
	        this.deadMoney = this.convertValues(source["deadMoney"], nativecheck.SeasonAmount);
	        this.reliefCents = source["reliefCents"];
	        this.cap = this.convertValues(source["cap"], ContractCapEffect);
	        this.basis = source["basis"];
	        this.notes = source["notes"];
	        this.source = source["source"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Market {
	    league: string;
	    revision: number;
	    windows: Choice[];
	    claims: Choice[];
	    players: Choice[];
	    bids: Choice[];
	    trades: Choice[];
	
	    static createFrom(source: any = {}) {
	        return new Market(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.league = source["league"];
	        this.revision = source["revision"];
	        this.windows = this.convertValues(source["windows"], Choice);
	        this.claims = this.convertValues(source["claims"], Choice);
	        this.players = this.convertValues(source["players"], Choice);
	        this.bids = this.convertValues(source["bids"], Choice);
	        this.trades = this.convertValues(source["trades"], Choice);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class NativeBidAward {
	    player: string;
	    name: string;
	    winner?: string;
	    team?: string;
	    contract?: string;
	    points?: string;
	    decided: time.Time;
	    words: string;
	    yours?: string;
	
	    static createFrom(source: any = {}) {
	        return new NativeBidAward(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.player = source["player"];
	        this.name = source["name"];
	        this.winner = source["winner"];
	        this.team = source["team"];
	        this.contract = source["contract"];
	        this.points = source["points"];
	        this.decided = this.convertValues(source["decided"], time.Time);
	        this.words = source["words"];
	        this.yours = source["yours"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class NativeBidRow {
	    id: string;
	    franchise: string;
	    team: string;
	    contract: string;
	    points: string;
	    placedAt: time.Time;
	    placed: string;
	
	    static createFrom(source: any = {}) {
	        return new NativeBidRow(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.franchise = source["franchise"];
	        this.team = source["team"];
	        this.contract = source["contract"];
	        this.points = source["points"];
	        this.placedAt = this.convertValues(source["placedAt"], time.Time);
	        this.placed = source["placed"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class NativeBidPlayer {
	    player: string;
	    name: string;
	    mode: string;
	    modeWords: string;
	    leader?: NativeBidRow;
	    bids?: number;
	    yourBid?: NativeBidRow;
	    youLead: boolean;
	    lateAt?: time.Time;
	    lateWords?: string;
	    lateNow: boolean;
	    closesAt?: time.Time;
	    closes: string;
	
	    static createFrom(source: any = {}) {
	        return new NativeBidPlayer(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.player = source["player"];
	        this.name = source["name"];
	        this.mode = source["mode"];
	        this.modeWords = source["modeWords"];
	        this.leader = this.convertValues(source["leader"], NativeBidRow);
	        this.bids = source["bids"];
	        this.yourBid = this.convertValues(source["yourBid"], NativeBidRow);
	        this.youLead = source["youLead"];
	        this.lateAt = this.convertValues(source["lateAt"], time.Time);
	        this.lateWords = source["lateWords"];
	        this.lateNow = source["lateNow"];
	        this.closesAt = this.convertValues(source["closesAt"], time.Time);
	        this.closes = source["closes"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class NativeBids {
	    league: string;
	    franchise: string;
	    revision: number;
	    season: number;
	    mode: string;
	    activeBids: number;
	    activeLimit: string;
	    players: NativeBidPlayer[];
	    awards: NativeBidAward[];
	    notes: string[];
	
	    static createFrom(source: any = {}) {
	        return new NativeBids(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.league = source["league"];
	        this.franchise = source["franchise"];
	        this.revision = source["revision"];
	        this.season = source["season"];
	        this.mode = source["mode"];
	        this.activeBids = source["activeBids"];
	        this.activeLimit = source["activeLimit"];
	        this.players = this.convertValues(source["players"], NativeBidPlayer);
	        this.awards = this.convertValues(source["awards"], NativeBidAward);
	        this.notes = source["notes"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class NativeTradeOffer {
	    vetoNumber: number;
	    appealedVetoNumber: number;
	    assetDetails: commissioner.TradeAssetDetail[];
	    previous?: string;
	    id: string;
	    proposer: string;
	    parties: string[];
	    assets: commissioner.TradeAsset[];
	    accepted: string[];
	    policy: commissioner.TradePolicy;
	    offeredAt: time.Time;
	    expiresAt: time.Time;
	    status: string;
	    words: string;
	    votes: commissioner.TradeVote[];
	    recusals: commissioner.TradeVote[];
	    appeals: commissioner.TradeVote[];
	    overrides: commissioner.TradeVote[];
	    expires: string;
	
	    static createFrom(source: any = {}) {
	        return new NativeTradeOffer(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.vetoNumber = source["vetoNumber"];
	        this.appealedVetoNumber = source["appealedVetoNumber"];
	        this.assetDetails = this.convertValues(source["assetDetails"], commissioner.TradeAssetDetail);
	        this.previous = source["previous"];
	        this.id = source["id"];
	        this.proposer = source["proposer"];
	        this.parties = source["parties"];
	        this.assets = this.convertValues(source["assets"], commissioner.TradeAsset);
	        this.accepted = source["accepted"];
	        this.policy = this.convertValues(source["policy"], commissioner.TradePolicy);
	        this.offeredAt = this.convertValues(source["offeredAt"], time.Time);
	        this.expiresAt = this.convertValues(source["expiresAt"], time.Time);
	        this.status = source["status"];
	        this.words = source["words"];
	        this.votes = this.convertValues(source["votes"], commissioner.TradeVote);
	        this.recusals = this.convertValues(source["recusals"], commissioner.TradeVote);
	        this.appeals = this.convertValues(source["appeals"], commissioner.TradeVote);
	        this.overrides = this.convertValues(source["overrides"], commissioner.TradeVote);
	        this.expires = source["expires"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class NativeTrades {
	    league: string;
	    viewer: string;
	    revision: number;
	    rules: commissioner.RuleLine[];
	    offers: NativeTradeOffer[];
	    notes: string[];
	
	    static createFrom(source: any = {}) {
	        return new NativeTrades(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.league = source["league"];
	        this.viewer = source["viewer"];
	        this.revision = source["revision"];
	        this.rules = this.convertValues(source["rules"], commissioner.RuleLine);
	        this.offers = this.convertValues(source["offers"], NativeTradeOffer);
	        this.notes = source["notes"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class NativeWaiverClaim {
	    id: string;
	    window: string;
	    player: string;
	    status: string;
	
	    static createFrom(source: any = {}) {
	        return new NativeWaiverClaim(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.window = source["window"];
	        this.player = source["player"];
	        this.status = source["status"];
	    }
	}
	export class NativeWaiverPlayer {
	    window: string;
	    player: string;
	    name: string;
	    closesAt: time.Time;
	    closes: string;
	    status: string;
	    winner?: string;
	
	    static createFrom(source: any = {}) {
	        return new NativeWaiverPlayer(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.window = source["window"];
	        this.player = source["player"];
	        this.name = source["name"];
	        this.closesAt = this.convertValues(source["closesAt"], time.Time);
	        this.closes = source["closes"];
	        this.status = source["status"];
	        this.winner = source["winner"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class NativeWaivers {
	    corrections?: commissioner.Correction[];
	    league: string;
	    franchise: string;
	    revision: number;
	    season: number;
	    players: NativeWaiverPlayer[];
	    claims: NativeWaiverClaim[];
	    order: string[];
	    orderSource: string;
	    method: string;
	
	    static createFrom(source: any = {}) {
	        return new NativeWaivers(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.corrections = this.convertValues(source["corrections"], commissioner.Correction);
	        this.league = source["league"];
	        this.franchise = source["franchise"];
	        this.revision = source["revision"];
	        this.season = source["season"];
	        this.players = this.convertValues(source["players"], NativeWaiverPlayer);
	        this.claims = this.convertValues(source["claims"], NativeWaiverClaim);
	        this.order = source["order"];
	        this.orderSource = source["orderSource"];
	        this.method = source["method"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class RawContractQuote {
	    league: string;
	    player: string;
	    action: string;
	    viewerZone: string;
	    years: number;
	    level: number;
	    moveCents: number;
	    amountCents: number;
	    reason: string;
	
	    static createFrom(source: any = {}) {
	        return new RawContractQuote(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.league = source["league"];
	        this.player = source["player"];
	        this.action = source["action"];
	        this.viewerZone = source["viewerZone"];
	        this.years = source["years"];
	        this.level = source["level"];
	        this.moveCents = source["moveCents"];
	        this.amountCents = source["amountCents"];
	        this.reason = source["reason"];
	    }
	}
	export class RawMarketScope {
	    from: number;
	    to: number;
	    phases: string[];
	    assets: string[];
	    byAge: boolean;
	
	    static createFrom(source: any = {}) {
	        return new RawMarketScope(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.from = source["from"];
	        this.to = source["to"];
	        this.phases = source["phases"];
	        this.assets = source["assets"];
	        this.byAge = source["byAge"];
	    }
	}
	export class RawMarketAsset {
	    scope: RawMarketScope;
	    kind: string;
	
	    static createFrom(source: any = {}) {
	        return new RawMarketAsset(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.scope = this.convertValues(source["scope"], RawMarketScope);
	        this.kind = source["kind"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace nativecheck {
	
	export class LineupProblem {
	    subject: string;
	    words: string;
	
	    static createFrom(source: any = {}) {
	        return new LineupProblem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.subject = source["subject"];
	        this.words = source["words"];
	    }
	}
	export class LineupVerdict {
	    status: string;
	    words: string;
	    problems: LineupProblem[];
	    flagged: boolean;
	    penalty: string;
	
	    static createFrom(source: any = {}) {
	        return new LineupVerdict(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.words = source["words"];
	        this.problems = this.convertValues(source["problems"], LineupProblem);
	        this.flagged = source["flagged"];
	        this.penalty = source["penalty"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ReserveVerdict {
	    player: string;
	    name: string;
	    squad: string;
	    standing: string;
	    words: string;
	
	    static createFrom(source: any = {}) {
	        return new ReserveVerdict(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.player = source["player"];
	        this.name = source["name"];
	        this.squad = source["squad"];
	        this.standing = source["standing"];
	        this.words = source["words"];
	    }
	}
	export class SeasonAmount {
	    season: number;
	    cents: number;
	
	    static createFrom(source: any = {}) {
	        return new SeasonAmount(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.season = source["season"];
	        this.cents = source["cents"];
	    }
	}
	export class SquadUse {
	    used: number;
	    slots: number;
	    judged: boolean;
	    words: string;
	
	    static createFrom(source: any = {}) {
	        return new SquadUse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.used = source["used"];
	        this.slots = source["slots"];
	        this.judged = source["judged"];
	        this.words = source["words"];
	    }
	}

}

export namespace now {
	
	export class Board {
	    reportWeek: number;
	    reportWeeks: report.Choice[];
	    week: number;
	    weeks: report.Choice[];
	    matchup: report.Report;
	    board: report.Board;
	    waiting: string;
	
	    static createFrom(source: any = {}) {
	        return new Board(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.reportWeek = source["reportWeek"];
	        this.reportWeeks = this.convertValues(source["reportWeeks"], report.Choice);
	        this.week = source["week"];
	        this.weeks = this.convertValues(source["weeks"], report.Choice);
	        this.matchup = this.convertValues(source["matchup"], report.Report);
	        this.board = this.convertValues(source["board"], report.Board);
	        this.waiting = source["waiting"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PhotoView {
	    src: string;
	    alt: string;
	
	    static createFrom(source: any = {}) {
	        return new PhotoView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.src = source["src"];
	        this.alt = source["alt"];
	    }
	}
	export class SpinePlayer {
	    id: string;
	    name: string;
	    team: string;
	    mine: boolean;
	    photo: PhotoView;
	
	    static createFrom(source: any = {}) {
	        return new SpinePlayer(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.team = source["team"];
	        this.mine = source["mine"];
	        this.photo = this.convertValues(source["photo"], PhotoView);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SpineEntry {
	    player: SpinePlayer;
	    play: livegame.Play;
	    game: string;
	
	    static createFrom(source: any = {}) {
	        return new SpineEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.player = this.convertValues(source["player"], SpinePlayer);
	        this.play = this.convertValues(source["play"], livegame.Play);
	        this.game = source["game"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class LiveGameView {
	    games: livegame.Game[];
	    followed?: livegame.Game;
	    source: string;
	    asOf: string;
	    failure: string;
	    detailAsOf: string;
	    detailFailure: string;
	    scoring: livegame.Play[];
	    spine: SpineEntry[];
	    unmatched: number;
	    weekSpine: SpineEntry[];
	    weekHeld: number;
	    weekGames: number;
	    box: report.Report;
	    active: boolean;
	
	    static createFrom(source: any = {}) {
	        return new LiveGameView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.games = this.convertValues(source["games"], livegame.Game);
	        this.followed = this.convertValues(source["followed"], livegame.Game);
	        this.source = source["source"];
	        this.asOf = source["asOf"];
	        this.failure = source["failure"];
	        this.detailAsOf = source["detailAsOf"];
	        this.detailFailure = source["detailFailure"];
	        this.scoring = this.convertValues(source["scoring"], livegame.Play);
	        this.spine = this.convertValues(source["spine"], SpineEntry);
	        this.unmatched = source["unmatched"];
	        this.weekSpine = this.convertValues(source["weekSpine"], SpineEntry);
	        this.weekHeld = source["weekHeld"];
	        this.weekGames = source["weekGames"];
	        this.box = this.convertValues(source["box"], report.Report);
	        this.active = source["active"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	

}

export namespace offers {
	
	export class BidAction {
	    id: string;
	    label: string;
	    disabled: boolean;
	    reason: string;
	
	    static createFrom(source: any = {}) {
	        return new BidAction(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.label = source["label"];
	        this.disabled = source["disabled"];
	        this.reason = source["reason"];
	    }
	}
	export class BidDropOption {
	    id: string;
	    name: string;
	    label: string;
	    salary: number;
	    deadCap: number;
	    freed: number;
	    covers: boolean;
	    available: boolean;
	
	    static createFrom(source: any = {}) {
	        return new BidDropOption(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.label = source["label"];
	        this.salary = source["salary"];
	        this.deadCap = source["deadCap"];
	        this.freed = source["freed"];
	        this.covers = source["covers"];
	        this.available = source["available"];
	    }
	}
	export class BidForm {
	    salary: string;
	    beat: string;
	
	    static createFrom(source: any = {}) {
	        return new BidForm(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.salary = source["salary"];
	        this.beat = source["beat"];
	    }
	}
	export class BidPlan {
	    receipt: envelope.Receipt;
	    facts: envelope.BidFacts;
	    player: string;
	    salary: snapshot.ContractAmount;
	    points: string;
	    contract: string;
	    beatText: string;
	    form: BidForm;
	    state: string;
	    status: string;
	    rules: bid.Result[];
	    ruleSummary: string;
	    post: string;
	    actions: BidAction[];
	
	    static createFrom(source: any = {}) {
	        return new BidPlan(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.receipt = this.convertValues(source["receipt"], envelope.Receipt);
	        this.facts = this.convertValues(source["facts"], envelope.BidFacts);
	        this.player = source["player"];
	        this.salary = this.convertValues(source["salary"], snapshot.ContractAmount);
	        this.points = source["points"];
	        this.contract = source["contract"];
	        this.beatText = source["beatText"];
	        this.form = this.convertValues(source["form"], BidForm);
	        this.state = source["state"];
	        this.status = source["status"];
	        this.rules = this.convertValues(source["rules"], bid.Result);
	        this.ruleSummary = source["ruleSummary"];
	        this.post = source["post"];
	        this.actions = this.convertValues(source["actions"], BidAction);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class BidPreview {
	    conditionalDrops: BidDropOption[];
	    problem: string;
	    player: string;
	    salary: snapshot.ContractAmount;
	    snapped: string;
	    points: string;
	    rules: bid.Result[];
	    blocked: boolean;
	    note: string;
	    post: string;
	    ruleSummary: string;
	    closesNote: string;
	
	    static createFrom(source: any = {}) {
	        return new BidPreview(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.conditionalDrops = this.convertValues(source["conditionalDrops"], BidDropOption);
	        this.problem = source["problem"];
	        this.player = source["player"];
	        this.salary = this.convertValues(source["salary"], snapshot.ContractAmount);
	        this.snapped = source["snapped"];
	        this.points = source["points"];
	        this.rules = this.convertValues(source["rules"], bid.Result);
	        this.blocked = source["blocked"];
	        this.note = source["note"];
	        this.post = source["post"];
	        this.ruleSummary = source["ruleSummary"];
	        this.closesNote = source["closesNote"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PersonalPlayer {
	    id: string;
	    status: string;
	    market: string[];
	
	    static createFrom(source: any = {}) {
	        return new PersonalPlayer(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.status = source["status"];
	        this.market = source["market"];
	    }
	}
	export class PersonalReading {
	    entries: personal.Entry[];
	    players: PersonalPlayer[];
	    bait: report.Report;
	
	    static createFrom(source: any = {}) {
	        return new PersonalReading(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.entries = this.convertValues(source["entries"], personal.Entry);
	        this.players = this.convertValues(source["players"], PersonalPlayer);
	        this.bait = this.convertValues(source["bait"], report.Report);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class RawBidInput {
	    conditionalDrop: string;
	    franchiseId: string;
	    player: string;
	    salary: string;
	    years: number;
	    snipe: boolean;
	    beat: string;
	    evidence: string;
	    closes: string;
	
	    static createFrom(source: any = {}) {
	        return new RawBidInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.conditionalDrop = source["conditionalDrop"];
	        this.franchiseId = source["franchiseId"];
	        this.player = source["player"];
	        this.salary = source["salary"];
	        this.years = source["years"];
	        this.snipe = source["snipe"];
	        this.beat = source["beat"];
	        this.evidence = source["evidence"];
	        this.closes = source["closes"];
	    }
	}
	export class RawOfferScope {
	    from: number;
	    to: number;
	    direction: string;
	    window: string;
	    team: string;
	
	    static createFrom(source: any = {}) {
	        return new RawOfferScope(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.from = source["from"];
	        this.to = source["to"];
	        this.direction = source["direction"];
	        this.window = source["window"];
	        this.team = source["team"];
	    }
	}
	export class RawPoolScope {
	    showNoTeam: boolean;
	    franchiseId: string;
	    position: string;
	    sort: string;
	    desc: boolean;
	
	    static createFrom(source: any = {}) {
	        return new RawPoolScope(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.showNoTeam = source["showNoTeam"];
	        this.franchiseId = source["franchiseId"];
	        this.position = source["position"];
	        this.sort = source["sort"];
	        this.desc = source["desc"];
	    }
	}

}

export namespace personal {
	
	export class Entry {
	    kind: string;
	    entity: string;
	    id: string;
	    text: string;
	    at: string;
	    order: number;
	
	    static createFrom(source: any = {}) {
	        return new Entry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.entity = source["entity"];
	        this.id = source["id"];
	        this.text = source["text"];
	        this.at = source["at"];
	        this.order = source["order"];
	    }
	}
	export class RawChange {
	    mine: string;
	    kind: string;
	    entity: string;
	    id: string;
	    text: string;
	    action: string;
	
	    static createFrom(source: any = {}) {
	        return new RawChange(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mine = source["mine"];
	        this.kind = source["kind"];
	        this.entity = source["entity"];
	        this.id = source["id"];
	        this.text = source["text"];
	        this.action = source["action"];
	    }
	}

}

export namespace platform {
	
	export class PhotoView {
	    src: string;
	    alt: string;
	
	    static createFrom(source: any = {}) {
	        return new PhotoView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.src = source["src"];
	        this.alt = source["alt"];
	    }
	}

}

export namespace race {
	
	export class RaceBoard {
	    standings: report.Report;
	    board: report.Board;
	    week: number;
	    weeks: report.Choice[];
	    team: string;
	    teams: report.Choice[];
	
	    static createFrom(source: any = {}) {
	        return new RaceBoard(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.standings = this.convertValues(source["standings"], report.Report);
	        this.board = this.convertValues(source["board"], report.Board);
	        this.week = source["week"];
	        this.weeks = this.convertValues(source["weeks"], report.Choice);
	        this.team = source["team"];
	        this.teams = this.convertValues(source["teams"], report.Choice);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace report {
	
	export class Link {
	    label: string;
	    endpoint: string;
	
	    static createFrom(source: any = {}) {
	        return new Link(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.label = source["label"];
	        this.endpoint = source["endpoint"];
	    }
	}
	export class Range {
	    max?: number;
	    value: number;
	    lo: number;
	    hi: number;
	    text: string;
	
	    static createFrom(source: any = {}) {
	        return new Range(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.max = source["max"];
	        this.value = source["value"];
	        this.lo = source["lo"];
	        this.hi = source["hi"];
	        this.text = source["text"];
	    }
	}
	export class Row {
	    sort?: number[];
	    titles?: string[];
	    cells: string[];
	    range?: Range;
	    mine?: boolean;
	    bars?: number[];
	    group?: string;
	
	    static createFrom(source: any = {}) {
	        return new Row(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sort = source["sort"];
	        this.titles = source["titles"];
	        this.cells = source["cells"];
	        this.range = this.convertValues(source["range"], Range);
	        this.mine = source["mine"];
	        this.bars = source["bars"];
	        this.group = source["group"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Entity {
	    text: string;
	    kind: string;
	    id: string;
	
	    static createFrom(source: any = {}) {
	        return new Entity(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.text = source["text"];
	        this.kind = source["kind"];
	        this.id = source["id"];
	    }
	}
	export class Report {
	    entities?: Entity[];
	    sortAscending?: string;
	    sortDescending?: string;
	    id: string;
	    title: string;
	    note: string;
	    status: string;
	    source: string;
	    asOf: string;
	    how: string;
	    columns: string[];
	    rows: Row[];
	    summary: Row[];
	    link?: Link;
	    barMax?: number;
	    barLabels?: string[];
	
	    static createFrom(source: any = {}) {
	        return new Report(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.entities = this.convertValues(source["entities"], Entity);
	        this.sortAscending = source["sortAscending"];
	        this.sortDescending = source["sortDescending"];
	        this.id = source["id"];
	        this.title = source["title"];
	        this.note = source["note"];
	        this.status = source["status"];
	        this.source = source["source"];
	        this.asOf = source["asOf"];
	        this.how = source["how"];
	        this.columns = source["columns"];
	        this.rows = this.convertValues(source["rows"], Row);
	        this.summary = this.convertValues(source["summary"], Row);
	        this.link = this.convertValues(source["link"], Link);
	        this.barMax = source["barMax"];
	        this.barLabels = source["barLabels"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Board {
	    title: string;
	    loading: string;
	    refresh: string;
	    howLabel: string;
	    mflLabel: string;
	    mflUrl: string;
	    reports: Report[];
	
	    static createFrom(source: any = {}) {
	        return new Board(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.loading = source["loading"];
	        this.refresh = source["refresh"];
	        this.howLabel = source["howLabel"];
	        this.mflLabel = source["mflLabel"];
	        this.mflUrl = source["mflUrl"];
	        this.reports = this.convertValues(source["reports"], Report);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Choice {
	    value: string;
	    label: string;
	
	    static createFrom(source: any = {}) {
	        return new Choice(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.value = source["value"];
	        this.label = source["label"];
	    }
	}
	
	
	
	export class Reading_github_com_secureprospective_TheWarRoom_internal_app_market_MarketSurface_ {
	    // Go type: market
	    value: any;
	    provenance: snapshot.Provenance;
	    status: string;
	    absenceReason?: string;
	    count: number;
	    threshold: number;
	    baseline: string;
	    cutoff: string;
	    method: string;
	    inputSHA: string;
	    buildStamp: string;
	    howCounted: string;
	
	    static createFrom(source: any = {}) {
	        return new Reading_github_com_secureprospective_TheWarRoom_internal_app_market_MarketSurface_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.value = this.convertValues(source["value"], null);
	        this.provenance = this.convertValues(source["provenance"], snapshot.Provenance);
	        this.status = source["status"];
	        this.absenceReason = source["absenceReason"];
	        this.count = source["count"];
	        this.threshold = source["threshold"];
	        this.baseline = source["baseline"];
	        this.cutoff = source["cutoff"];
	        this.method = source["method"];
	        this.inputSHA = source["inputSHA"];
	        this.buildStamp = source["buildStamp"];
	        this.howCounted = source["howCounted"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Reading_github_com_secureprospective_TheWarRoom_internal_app_offers_BidBoard_ {
	    // Go type: offers
	    value: any;
	    provenance: snapshot.Provenance;
	    status: string;
	    absenceReason?: string;
	    count: number;
	    threshold: number;
	    baseline: string;
	    cutoff: string;
	    method: string;
	    inputSHA: string;
	    buildStamp: string;
	    howCounted: string;
	
	    static createFrom(source: any = {}) {
	        return new Reading_github_com_secureprospective_TheWarRoom_internal_app_offers_BidBoard_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.value = this.convertValues(source["value"], null);
	        this.provenance = this.convertValues(source["provenance"], snapshot.Provenance);
	        this.status = source["status"];
	        this.absenceReason = source["absenceReason"];
	        this.count = source["count"];
	        this.threshold = source["threshold"];
	        this.baseline = source["baseline"];
	        this.cutoff = source["cutoff"];
	        this.method = source["method"];
	        this.inputSHA = source["inputSHA"];
	        this.buildStamp = source["buildStamp"];
	        this.howCounted = source["howCounted"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Reading_github_com_secureprospective_TheWarRoom_internal_app_offers_OffersView_ {
	    // Go type: offers
	    value: any;
	    provenance: snapshot.Provenance;
	    status: string;
	    absenceReason?: string;
	    count: number;
	    threshold: number;
	    baseline: string;
	    cutoff: string;
	    method: string;
	    inputSHA: string;
	    buildStamp: string;
	    howCounted: string;
	
	    static createFrom(source: any = {}) {
	        return new Reading_github_com_secureprospective_TheWarRoom_internal_app_offers_OffersView_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.value = this.convertValues(source["value"], null);
	        this.provenance = this.convertValues(source["provenance"], snapshot.Provenance);
	        this.status = source["status"];
	        this.absenceReason = source["absenceReason"];
	        this.count = source["count"];
	        this.threshold = source["threshold"];
	        this.baseline = source["baseline"];
	        this.cutoff = source["cutoff"];
	        this.method = source["method"];
	        this.inputSHA = source["inputSHA"];
	        this.buildStamp = source["buildStamp"];
	        this.howCounted = source["howCounted"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Reading_github_com_secureprospective_TheWarRoom_internal_app_offers_Pool_ {
	    // Go type: offers
	    value: any;
	    provenance: snapshot.Provenance;
	    status: string;
	    absenceReason?: string;
	    count: number;
	    threshold: number;
	    baseline: string;
	    cutoff: string;
	    method: string;
	    inputSHA: string;
	    buildStamp: string;
	    howCounted: string;
	
	    static createFrom(source: any = {}) {
	        return new Reading_github_com_secureprospective_TheWarRoom_internal_app_offers_Pool_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.value = this.convertValues(source["value"], null);
	        this.provenance = this.convertValues(source["provenance"], snapshot.Provenance);
	        this.status = source["status"];
	        this.absenceReason = source["absenceReason"];
	        this.count = source["count"];
	        this.threshold = source["threshold"];
	        this.baseline = source["baseline"];
	        this.cutoff = source["cutoff"];
	        this.method = source["method"];
	        this.inputSHA = source["inputSHA"];
	        this.buildStamp = source["buildStamp"];
	        this.howCounted = source["howCounted"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Reading_github_com_secureprospective_TheWarRoom_internal_lab_Market_ {
	    // Go type: lab
	    value: any;
	    provenance: snapshot.Provenance;
	    status: string;
	    absenceReason?: string;
	    count: number;
	    threshold: number;
	    baseline: string;
	    cutoff: string;
	    method: string;
	    inputSHA: string;
	    buildStamp: string;
	    howCounted: string;
	
	    static createFrom(source: any = {}) {
	        return new Reading_github_com_secureprospective_TheWarRoom_internal_lab_Market_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.value = this.convertValues(source["value"], null);
	        this.provenance = this.convertValues(source["provenance"], snapshot.Provenance);
	        this.status = source["status"];
	        this.absenceReason = source["absenceReason"];
	        this.count = source["count"];
	        this.threshold = source["threshold"];
	        this.baseline = source["baseline"];
	        this.cutoff = source["cutoff"];
	        this.method = source["method"];
	        this.inputSHA = source["inputSHA"];
	        this.buildStamp = source["buildStamp"];
	        this.howCounted = source["howCounted"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Reading_github_com_secureprospective_TheWarRoom_internal_snapshot_CapSummary_ {
	    // Go type: snapshot
	    value: any;
	    provenance: snapshot.Provenance;
	    status: string;
	    absenceReason?: string;
	    count: number;
	    threshold: number;
	    baseline: string;
	    cutoff: string;
	    method: string;
	    inputSHA: string;
	    buildStamp: string;
	    howCounted: string;
	
	    static createFrom(source: any = {}) {
	        return new Reading_github_com_secureprospective_TheWarRoom_internal_snapshot_CapSummary_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.value = this.convertValues(source["value"], null);
	        this.provenance = this.convertValues(source["provenance"], snapshot.Provenance);
	        this.status = source["status"];
	        this.absenceReason = source["absenceReason"];
	        this.count = source["count"];
	        this.threshold = source["threshold"];
	        this.baseline = source["baseline"];
	        this.cutoff = source["cutoff"];
	        this.method = source["method"];
	        this.inputSHA = source["inputSHA"];
	        this.buildStamp = source["buildStamp"];
	        this.howCounted = source["howCounted"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Reading_github_com_secureprospective_TheWarRoom_internal_snapshot_ContractTable_ {
	    // Go type: snapshot
	    value: any;
	    provenance: snapshot.Provenance;
	    status: string;
	    absenceReason?: string;
	    count: number;
	    threshold: number;
	    baseline: string;
	    cutoff: string;
	    method: string;
	    inputSHA: string;
	    buildStamp: string;
	    howCounted: string;
	
	    static createFrom(source: any = {}) {
	        return new Reading_github_com_secureprospective_TheWarRoom_internal_snapshot_ContractTable_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.value = this.convertValues(source["value"], null);
	        this.provenance = this.convertValues(source["provenance"], snapshot.Provenance);
	        this.status = source["status"];
	        this.absenceReason = source["absenceReason"];
	        this.count = source["count"];
	        this.threshold = source["threshold"];
	        this.baseline = source["baseline"];
	        this.cutoff = source["cutoff"];
	        this.method = source["method"];
	        this.inputSHA = source["inputSHA"];
	        this.buildStamp = source["buildStamp"];
	        this.howCounted = source["howCounted"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Reading_github_com_secureprospective_TheWarRoom_internal_snapshot_SalaryAdjustments_ {
	    // Go type: snapshot
	    value: any;
	    provenance: snapshot.Provenance;
	    status: string;
	    absenceReason?: string;
	    count: number;
	    threshold: number;
	    baseline: string;
	    cutoff: string;
	    method: string;
	    inputSHA: string;
	    buildStamp: string;
	    howCounted: string;
	
	    static createFrom(source: any = {}) {
	        return new Reading_github_com_secureprospective_TheWarRoom_internal_snapshot_SalaryAdjustments_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.value = this.convertValues(source["value"], null);
	        this.provenance = this.convertValues(source["provenance"], snapshot.Provenance);
	        this.status = source["status"];
	        this.absenceReason = source["absenceReason"];
	        this.count = source["count"];
	        this.threshold = source["threshold"];
	        this.baseline = source["baseline"];
	        this.cutoff = source["cutoff"];
	        this.method = source["method"];
	        this.inputSHA = source["inputSHA"];
	        this.buildStamp = source["buildStamp"];
	        this.howCounted = source["howCounted"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	

}

export namespace research {
	
	export class RawResearchScope {
	    mine: string;
	    search: string;
	    position: string;
	    team: string;
	    group: string;
	    trend: string;
	    depth: string;
	    injuries: string;
	
	    static createFrom(source: any = {}) {
	        return new RawResearchScope(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mine = source["mine"];
	        this.search = source["search"];
	        this.position = source["position"];
	        this.team = source["team"];
	        this.group = source["group"];
	        this.trend = source["trend"];
	        this.depth = source["depth"];
	        this.injuries = source["injuries"];
	    }
	}
	export class ResearchArticle {
	    id: string;
	    article: string;
	    name: string;
	    headline: string;
	
	    static createFrom(source: any = {}) {
	        return new ResearchArticle(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.article = source["article"];
	        this.name = source["name"];
	        this.headline = source["headline"];
	    }
	}
	export class ResearchPlayer {
	    id: string;
	    name: string;
	    position: string;
	    team: string;
	    franchiseId: string;
	    owner: string;
	
	    static createFrom(source: any = {}) {
	        return new ResearchPlayer(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.position = source["position"];
	        this.team = source["team"];
	        this.franchiseId = source["franchiseId"];
	        this.owner = source["owner"];
	    }
	}
	export class ResearchBoard {
	    board: report.Board;
	    news: report.Report;
	    articles: ResearchArticle[];
	    players: ResearchPlayer[];
	    teams: string[];
	    positions: string[];
	    scope: RawResearchScope;
	
	    static createFrom(source: any = {}) {
	        return new ResearchBoard(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.board = this.convertValues(source["board"], report.Board);
	        this.news = this.convertValues(source["news"], report.Report);
	        this.articles = this.convertValues(source["articles"], ResearchArticle);
	        this.players = this.convertValues(source["players"], ResearchPlayer);
	        this.teams = source["teams"];
	        this.positions = source["positions"];
	        this.scope = this.convertValues(source["scope"], RawResearchScope);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class ResearchPlayerReading {
	    id: string;
	    name: string;
	    position: string;
	    team: string;
	    franchiseId: string;
	    owner: string;
	    report: report.Report;
	    articles: ResearchArticle[];
	    howLabel: string;
	
	    static createFrom(source: any = {}) {
	        return new ResearchPlayerReading(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.position = source["position"];
	        this.team = source["team"];
	        this.franchiseId = source["franchiseId"];
	        this.owner = source["owner"];
	        this.report = this.convertValues(source["report"], report.Report);
	        this.articles = this.convertValues(source["articles"], ResearchArticle);
	        this.howLabel = source["howLabel"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace rookiedraft {
	
	export class Terms {
	    salaryCents: number;
	    years: number;
	    rfaAfter: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Terms(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.salaryCents = source["salaryCents"];
	        this.years = source["years"];
	        this.rfaAfter = source["rfaAfter"];
	    }
	}
	export class Clock {
	    round: number;
	    original: string;
	    owner: string;
	    slot: number;
	    openedAt: time.Time;
	    deadline: time.Time;
	    contract: Terms;
	    paused: boolean;
	    remaining: number;
	
	    static createFrom(source: any = {}) {
	        return new Clock(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.round = source["round"];
	        this.original = source["original"];
	        this.owner = source["owner"];
	        this.slot = source["slot"];
	        this.openedAt = this.convertValues(source["openedAt"], time.Time);
	        this.deadline = this.convertValues(source["deadline"], time.Time);
	        this.contract = this.convertValues(source["contract"], Terms);
	        this.paused = source["paused"];
	        this.remaining = source["remaining"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Result {
	    round: number;
	    original: string;
	    owner: string;
	    slot: number;
	    extra?: number;
	    player?: string;
	    skipped: boolean;
	    forfeited: boolean;
	    madeAt?: time.Time;
	    contract?: Terms;
	    how?: string;
	    changed?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Result(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.round = source["round"];
	        this.original = source["original"];
	        this.owner = source["owner"];
	        this.slot = source["slot"];
	        this.extra = source["extra"];
	        this.player = source["player"];
	        this.skipped = source["skipped"];
	        this.forfeited = source["forfeited"];
	        this.madeAt = this.convertValues(source["madeAt"], time.Time);
	        this.contract = this.convertValues(source["contract"], Terms);
	        this.how = source["how"];
	        this.changed = source["changed"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Draft {
	    start: time.Time;
	    started: boolean;
	    exhausted: boolean;
	    picks: Result[];
	    onClock?: Clock;
	
	    static createFrom(source: any = {}) {
	        return new Draft(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.start = this.convertValues(source["start"], time.Time);
	        this.started = source["started"];
	        this.exhausted = source["exhausted"];
	        this.picks = this.convertValues(source["picks"], Result);
	        this.onClock = this.convertValues(source["onClock"], Clock);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Pick {
	    round: number;
	    original: string;
	    owner: string;
	    slot: number;
	
	    static createFrom(source: any = {}) {
	        return new Pick(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.round = source["round"];
	        this.original = source["original"];
	        this.owner = source["owner"];
	        this.slot = source["slot"];
	    }
	}
	
	export class RoundOrder {
	    round: number;
	    words: string;
	    picks: Pick[];
	
	    static createFrom(source: any = {}) {
	        return new RoundOrder(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.round = source["round"];
	        this.words = source["words"];
	        this.picks = this.convertValues(source["picks"], Pick);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace snapshot {
	
	export class ContractAmount {
	    cents: number;
	    text: string;
	
	    static createFrom(source: any = {}) {
	        return new ContractAmount(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.cents = source["cents"];
	        this.text = source["text"];
	    }
	}
	export class Provenance {
	    source: string;
	    freshness: domain.Freshness;
	    kind: string;
	
	    static createFrom(source: any = {}) {
	        return new Provenance(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.source = source["source"];
	        this.freshness = this.convertValues(source["freshness"], domain.Freshness);
	        this.kind = source["kind"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Eligibility {
	    players: Record<string, eligibility.PlayerVerdicts>;
	    feeds: Record<string, Provenance>;
	
	    static createFrom(source: any = {}) {
	        return new Eligibility(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.players = this.convertValues(source["players"], eligibility.PlayerVerdicts, true);
	        this.feeds = this.convertValues(source["feeds"], Provenance, true);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Franchise {
	    id: string;
	    name?: string;
	    capUsed?: number;
	    capRoom?: number;
	
	    static createFrom(source: any = {}) {
	        return new Franchise(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.capUsed = source["capUsed"];
	        this.capRoom = source["capRoom"];
	    }
	}
	export class League {
	    season: number;
	    franchiseCount: number;
	    salaryCap?: number;
	
	    static createFrom(source: any = {}) {
	        return new League(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.season = source["season"];
	        this.franchiseCount = source["franchiseCount"];
	        this.salaryCap = source["salaryCap"];
	    }
	}
	export class Player {
	    // Go type: playerid
	    id: any;
	    name?: string;
	    position?: string;
	    team?: string;
	    birthdate?: number;
	    isRookie?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Player(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = this.convertValues(source["id"], null);
	        this.name = source["name"];
	        this.position = source["position"];
	        this.team = source["team"];
	        this.birthdate = source["birthdate"];
	        this.isRookie = source["isRookie"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class RawContractScope {
	    mine: string;
	    team: string;
	    position: string;
	    sort: string;
	    desc: boolean;
	
	    static createFrom(source: any = {}) {
	        return new RawContractScope(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mine = source["mine"];
	        this.team = source["team"];
	        this.position = source["position"];
	        this.sort = source["sort"];
	        this.desc = source["desc"];
	    }
	}
	export class RosterPlayer {
	    // Go type: playerid
	    id: any;
	    salary: number;
	    yearsRemaining?: number;
	    contractStatus: string;
	    rosterStatus: string;
	
	    static createFrom(source: any = {}) {
	        return new RosterPlayer(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = this.convertValues(source["id"], null);
	        this.salary = source["salary"];
	        this.yearsRemaining = source["yearsRemaining"];
	        this.contractStatus = source["contractStatus"];
	        this.rosterStatus = source["rosterStatus"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Roster {
	    franchiseId: string;
	    players: RosterPlayer[];
	
	    static createFrom(source: any = {}) {
	        return new Roster(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.franchiseId = source["franchiseId"];
	        this.players = this.convertValues(source["players"], RosterPlayer);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class Sourced___github_com_secureprospective_TheWarRoom_internal_snapshot_Player_ {
	    value: Player[];
	    provenance: Provenance;
	
	    static createFrom(source: any = {}) {
	        return new Sourced___github_com_secureprospective_TheWarRoom_internal_snapshot_Player_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.value = this.convertValues(source["value"], Player);
	        this.provenance = this.convertValues(source["provenance"], Provenance);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Sourced___github_com_secureprospective_TheWarRoom_internal_snapshot_Roster_ {
	    value: Roster[];
	    provenance: Provenance;
	
	    static createFrom(source: any = {}) {
	        return new Sourced___github_com_secureprospective_TheWarRoom_internal_snapshot_Roster_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.value = this.convertValues(source["value"], Roster);
	        this.provenance = this.convertValues(source["provenance"], Provenance);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Sourced___github_com_secureprospective_TheWarRoom_internal_snapshot_Franchise_ {
	    value: Franchise[];
	    provenance: Provenance;
	
	    static createFrom(source: any = {}) {
	        return new Sourced___github_com_secureprospective_TheWarRoom_internal_snapshot_Franchise_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.value = this.convertValues(source["value"], Franchise);
	        this.provenance = this.convertValues(source["provenance"], Provenance);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Sourced_github_com_secureprospective_TheWarRoom_internal_snapshot_League_ {
	    value: League;
	    provenance: Provenance;
	
	    static createFrom(source: any = {}) {
	        return new Sourced_github_com_secureprospective_TheWarRoom_internal_snapshot_League_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.value = this.convertValues(source["value"], League);
	        this.provenance = this.convertValues(source["provenance"], Provenance);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Snapshot {
	    league: Sourced_github_com_secureprospective_TheWarRoom_internal_snapshot_League_;
	    // Go type: Sourced___github_com_secureprospective_TheWarRoom_internal_snapshot_Franchise_
	    franchises: any;
	    // Go type: Sourced___github_com_secureprospective_TheWarRoom_internal_snapshot_Roster_
	    rosters: any;
	    // Go type: Sourced___github_com_secureprospective_TheWarRoom_internal_snapshot_Player_
	    players: any;
	    eligibility: Eligibility;
	
	    static createFrom(source: any = {}) {
	        return new Snapshot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.league = this.convertValues(source["league"], Sourced_github_com_secureprospective_TheWarRoom_internal_snapshot_League_);
	        this.franchises = this.convertValues(source["franchises"], null);
	        this.rosters = this.convertValues(source["rosters"], null);
	        this.players = this.convertValues(source["players"], null);
	        this.eligibility = this.convertValues(source["eligibility"], Eligibility);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Sourced___github_com_secureprospective_TheWarRoom_internal_leaguefeed_PendingTrade_ {
	    value: leaguefeed.PendingTrade[];
	    provenance: Provenance;
	
	    static createFrom(source: any = {}) {
	        return new Sourced___github_com_secureprospective_TheWarRoom_internal_leaguefeed_PendingTrade_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.value = this.convertValues(source["value"], leaguefeed.PendingTrade);
	        this.provenance = this.convertValues(source["provenance"], Provenance);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Sourced___github_com_secureprospective_TheWarRoom_internal_leaguefeed_Transaction_ {
	    value: leaguefeed.Transaction[];
	    provenance: Provenance;
	
	    static createFrom(source: any = {}) {
	        return new Sourced___github_com_secureprospective_TheWarRoom_internal_leaguefeed_Transaction_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.value = this.convertValues(source["value"], leaguefeed.Transaction);
	        this.provenance = this.convertValues(source["provenance"], Provenance);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	
	export class Sourced_github_com_secureprospective_TheWarRoom_internal_leagueclock_Reading_ {
	    value: leagueclock.Reading;
	    provenance: Provenance;
	
	    static createFrom(source: any = {}) {
	        return new Sourced_github_com_secureprospective_TheWarRoom_internal_leagueclock_Reading_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.value = this.convertValues(source["value"], leagueclock.Reading);
	        this.provenance = this.convertValues(source["provenance"], Provenance);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Sourced_github_com_secureprospective_TheWarRoom_internal_leaguefeed_Lineups_ {
	    value: leaguefeed.Lineups;
	    provenance: Provenance;
	
	    static createFrom(source: any = {}) {
	        return new Sourced_github_com_secureprospective_TheWarRoom_internal_leaguefeed_Lineups_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.value = this.convertValues(source["value"], leaguefeed.Lineups);
	        this.provenance = this.convertValues(source["provenance"], Provenance);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace sources {
	
	export class CrosswalkMiss {
	    mflId: string;
	    name: string;
	    team: string;
	    position: string;
	    franchise: string;
	    reason: string;
	
	    static createFrom(source: any = {}) {
	        return new CrosswalkMiss(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mflId = source["mflId"];
	        this.name = source["name"];
	        this.team = source["team"];
	        this.position = source["position"];
	        this.franchise = source["franchise"];
	        this.reason = source["reason"];
	    }
	}
	export class CrosswalkRate {
	    position: string;
	    rostered: number;
	    rosteredMatched: number;
	    freeAgents: number;
	    freeAgentsMatched: number;
	
	    static createFrom(source: any = {}) {
	        return new CrosswalkRate(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.position = source["position"];
	        this.rostered = source["rostered"];
	        this.rosteredMatched = source["rosteredMatched"];
	        this.freeAgents = source["freeAgents"];
	        this.freeAgentsMatched = source["freeAgentsMatched"];
	    }
	}
	export class CrosswalkReport {
	    ok: boolean;
	    error: string;
	    loadedAt: string;
	    links: number;
	    promoted: number;
	    rates: CrosswalkRate[];
	    total: CrosswalkRate;
	    unmatched: CrosswalkMiss[];
	
	    static createFrom(source: any = {}) {
	        return new CrosswalkReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.error = source["error"];
	        this.loadedAt = source["loadedAt"];
	        this.links = source["links"];
	        this.promoted = source["promoted"];
	        this.rates = this.convertValues(source["rates"], CrosswalkRate);
	        this.total = this.convertValues(source["total"], CrosswalkRate);
	        this.unmatched = this.convertValues(source["unmatched"], CrosswalkMiss);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DataActResult {
	    ok: boolean;
	    error: string;
	    summary: string;
	
	    static createFrom(source: any = {}) {
	        return new DataActResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.error = source["error"];
	        this.summary = source["summary"];
	    }
	}
	export class DataScope {
	    id: string;
	    label: string;
	    note: string;
	    action: string;
	    progress: string;
	
	    static createFrom(source: any = {}) {
	        return new DataScope(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.label = source["label"];
	        this.note = source["note"];
	        this.action = source["action"];
	        this.progress = source["progress"];
	    }
	}
	export class DataSources {
	    databaseChecks?: report.Report;
	    health: report.Report;
	    signals: report.Report;
	    facts: report.Report;
	    checks: report.Report;
	    archiveDir: string;
	    photosOn: boolean;
	    scopes: DataScope[];
	    words: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new DataSources(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.databaseChecks = this.convertValues(source["databaseChecks"], report.Report);
	        this.health = this.convertValues(source["health"], report.Report);
	        this.signals = this.convertValues(source["signals"], report.Report);
	        this.facts = this.convertValues(source["facts"], report.Report);
	        this.checks = this.convertValues(source["checks"], report.Report);
	        this.archiveDir = source["archiveDir"];
	        this.photosOn = source["photosOn"];
	        this.scopes = this.convertValues(source["scopes"], DataScope);
	        this.words = source["words"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace time {
	
	export class Time {
	
	
	    static createFrom(source: any = {}) {
	        return new Time(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}

}

export namespace trades {
	
	export class Cell {
	    text: string;
	    sub: string;
	    detail: string;
	    held: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Cell(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.text = source["text"];
	        this.sub = source["sub"];
	        this.detail = source["detail"];
	        this.held = source["held"];
	    }
	}
	export class RawTradeBuilder {
	    mine: string;
	    other: string;
	    action: string;
	    id: string;
	    give: string[];
	    get: string[];
	    rationale: string;
	
	    static createFrom(source: any = {}) {
	        return new RawTradeBuilder(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mine = source["mine"];
	        this.other = source["other"];
	        this.action = source["action"];
	        this.id = source["id"];
	        this.give = source["give"];
	        this.get = source["get"];
	        this.rationale = source["rationale"];
	    }
	}
	export class TradeAsset {
	    team?: string;
	    contract?: string;
	    now?: Cell;
	    dynasty?: Cell;
	    token: string;
	    name: string;
	    position?: string;
	
	    static createFrom(source: any = {}) {
	        return new TradeAsset(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.team = source["team"];
	        this.contract = source["contract"];
	        this.now = this.convertValues(source["now"], Cell);
	        this.dynasty = this.convertValues(source["dynasty"], Cell);
	        this.token = source["token"];
	        this.name = source["name"];
	        this.position = source["position"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Figure {
	    label: string;
	    before: string;
	    after: string;
	    change: string;
	
	    static createFrom(source: any = {}) {
	        return new Figure(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.label = source["label"];
	        this.before = source["before"];
	        this.after = source["after"];
	        this.change = source["change"];
	    }
	}
	export class TradeProjection {
	    team: string;
	    figures: Figure[];
	    roster: TradeAsset[];
	
	    static createFrom(source: any = {}) {
	        return new TradeProjection(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.team = source["team"];
	        this.figures = this.convertValues(source["figures"], Figure);
	        this.roster = this.convertValues(source["roster"], TradeAsset);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class TradeReset {
	    franchise: string;
	    label: string;
	
	    static createFrom(source: any = {}) {
	        return new TradeReset(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.franchise = source["franchise"];
	        this.label = source["label"];
	    }
	}
	export class TradeBuilderPlan {
	    id: string;
	    post: string;
	    reasons: string[];
	    resets?: TradeReset[];
	    teams: TradeProjection[];
	    receipt?: envelope.Receipt;
	
	    static createFrom(source: any = {}) {
	        return new TradeBuilderPlan(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.post = source["post"];
	        this.reasons = source["reasons"];
	        this.resets = this.convertValues(source["resets"], TradeReset);
	        this.teams = this.convertValues(source["teams"], TradeProjection);
	        this.receipt = this.convertValues(source["receipt"], envelope.Receipt);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class TradeBuilderTeam {
	    id: string;
	    name: string;
	    assets: TradeAsset[];
	
	    static createFrom(source: any = {}) {
	        return new TradeBuilderTeam(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.assets = this.convertValues(source["assets"], TradeAsset);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class TradeBuilderView {
	    teams: TradeBuilderTeam[];
	    plans: TradeBuilderPlan[];
	    // Go type: TradeBuilderPlan
	    preview?: any;
	
	    static createFrom(source: any = {}) {
	        return new TradeBuilderView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.teams = this.convertValues(source["teams"], TradeBuilderTeam);
	        this.plans = this.convertValues(source["plans"], TradeBuilderPlan);
	        this.preview = this.convertValues(source["preview"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class TradeBuilderReading {
	    // Go type: TradeBuilderView
	    value: any;
	    provenance: snapshot.Provenance;
	    status: string;
	    absenceReason?: string;
	    count: number;
	    threshold: number;
	    baseline: string;
	    cutoff: string;
	    method: string;
	    inputSHA: string;
	    buildStamp: string;
	    howCounted: string;
	
	    static createFrom(source: any = {}) {
	        return new TradeBuilderReading(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.value = this.convertValues(source["value"], null);
	        this.provenance = this.convertValues(source["provenance"], snapshot.Provenance);
	        this.status = source["status"];
	        this.absenceReason = source["absenceReason"];
	        this.count = source["count"];
	        this.threshold = source["threshold"];
	        this.baseline = source["baseline"];
	        this.cutoff = source["cutoff"];
	        this.method = source["method"];
	        this.inputSHA = source["inputSHA"];
	        this.buildStamp = source["buildStamp"];
	        this.howCounted = source["howCounted"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class TradeOffer {
	    tradeId: string;
	    direction: string;
	    otherId: string;
	    otherName: string;
	    give: TradeAsset[];
	    get: TradeAsset[];
	    expires: time.Time;
	    comments: string;
	
	    static createFrom(source: any = {}) {
	        return new TradeOffer(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tradeId = source["tradeId"];
	        this.direction = source["direction"];
	        this.otherId = source["otherId"];
	        this.otherName = source["otherName"];
	        this.give = this.convertValues(source["give"], TradeAsset);
	        this.get = this.convertValues(source["get"], TradeAsset);
	        this.expires = this.convertValues(source["expires"], time.Time);
	        this.comments = source["comments"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class TradeReading {
	    offers: TradeOffer[];
	    provenance: snapshot.Provenance;
	
	    static createFrom(source: any = {}) {
	        return new TradeReading(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.offers = this.convertValues(source["offers"], TradeOffer);
	        this.provenance = this.convertValues(source["provenance"], snapshot.Provenance);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

