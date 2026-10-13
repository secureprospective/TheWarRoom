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
	
	export class AuditEntry {
	    // Go type: time
	    at: any;
	    from: string;
	    event: string;
	    to: string;
	    note: string;
	
	    static createFrom(source: any = {}) {
	        return new AuditEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.at = this.convertValues(source["at"], null);
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
	    // Go type: time
	    deadline?: any;
	
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
	        this.deadline = this.convertValues(source["deadline"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
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
	    correlationId: string;
	    spec: Spec;
	    state: string;
	    audit: AuditEntry[];
	
	    static createFrom(source: any = {}) {
	        return new Receipt(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
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
	
	export class Deadline {
	    id: string;
	    label: string;
	    // Go type: time
	    at?: any;
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
	        this.at = this.convertValues(source["at"], null);
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
	    // Go type: time
	    at?: any;
	    moved: boolean;
	    reason: string;
	
	    static createFrom(source: any = {}) {
	        return new Window(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.status = source["status"];
	        this.at = this.convertValues(source["at"], null);
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
	    // Go type: time
	    proposed: any;
	    // Go type: time
	    expires: any;
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
	        this.proposed = this.convertValues(source["proposed"], null);
	        this.expires = this.convertValues(source["expires"], null);
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
	    // Go type: time
	    expires: any;
	
	    static createFrom(source: any = {}) {
	        return new Trade(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.counterparty = source["counterparty"];
	        this.gave = this.convertValues(source["gave"], Asset);
	        this.counterpartyGave = this.convertValues(source["counterpartyGave"], Asset);
	        this.comments = source["comments"];
	        this.expires = this.convertValues(source["expires"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
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
	    // Go type: time
	    time: any;
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
	        this.time = this.convertValues(source["time"], null);
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

export namespace main {
	
	export class Alert {
	    kind: string;
	    urgency: string;
	    title: string;
	    detail: string;
	    node: string;
	    workspace: string;
	    // Go type: time
	    at: any;
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
	        this.at = this.convertValues(source["at"], null);
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
	export class AppInfo {
	    version: string;
	    commit: string;
	    buildDate: string;
	    startupError?: string;
	
	    static createFrom(source: any = {}) {
	        return new AppInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.commit = source["commit"];
	        this.buildDate = source["buildDate"];
	        this.startupError = source["startupError"];
	    }
	}
	export class RaceChoice {
	    value: string;
	    label: string;
	
	    static createFrom(source: any = {}) {
	        return new RaceChoice(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.value = source["value"];
	        this.label = source["label"];
	    }
	}
	export class ReportLink {
	    label: string;
	    endpoint: string;
	
	    static createFrom(source: any = {}) {
	        return new ReportLink(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.label = source["label"];
	        this.endpoint = source["endpoint"];
	    }
	}
	export class MarketRange {
	    max?: number;
	    value: number;
	    lo: number;
	    hi: number;
	    text: string;
	
	    static createFrom(source: any = {}) {
	        return new MarketRange(source);
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
	export class ReportRow {
	    sort?: number[];
	    titles?: string[];
	    cells: string[];
	    range?: MarketRange;
	    mine?: boolean;
	    bars?: number[];
	    group?: string;
	
	    static createFrom(source: any = {}) {
	        return new ReportRow(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sort = source["sort"];
	        this.titles = source["titles"];
	        this.cells = source["cells"];
	        this.range = this.convertValues(source["range"], MarketRange);
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
	export class ReportEntity {
	    text: string;
	    kind: string;
	    id: string;
	
	    static createFrom(source: any = {}) {
	        return new ReportEntity(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.text = source["text"];
	        this.kind = source["kind"];
	        this.id = source["id"];
	    }
	}
	export class Report {
	    entities?: ReportEntity[];
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
	    rows: ReportRow[];
	    summary: ReportRow[];
	    link?: ReportLink;
	    barMax?: number;
	    barLabels?: string[];
	
	    static createFrom(source: any = {}) {
	        return new Report(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.entities = this.convertValues(source["entities"], ReportEntity);
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
	        this.rows = this.convertValues(source["rows"], ReportRow);
	        this.summary = this.convertValues(source["summary"], ReportRow);
	        this.link = this.convertValues(source["link"], ReportLink);
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
	export class ReportBoard {
	    title: string;
	    loading: string;
	    refresh: string;
	    howLabel: string;
	    mflLabel: string;
	    mflUrl: string;
	    reports: Report[];
	
	    static createFrom(source: any = {}) {
	        return new ReportBoard(source);
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
	export class ArchiveBoard {
	    board: ReportBoard;
	    season: number;
	    seasons: RaceChoice[];
	    team: string;
	    other: string;
	    teams: RaceChoice[];
	
	    static createFrom(source: any = {}) {
	        return new ArchiveBoard(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.board = this.convertValues(source["board"], ReportBoard);
	        this.season = source["season"];
	        this.seasons = this.convertValues(source["seasons"], RaceChoice);
	        this.team = source["team"];
	        this.other = source["other"];
	        this.teams = this.convertValues(source["teams"], RaceChoice);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
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
	export class CalendarEventDTO {
	    eventID: string;
	    kind: string;
	    scheduledAt: string;
	    payload: string;
	    status: string;
	    note: string;
	    createdAt: string;
	
	    static createFrom(source: any = {}) {
	        return new CalendarEventDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.eventID = source["eventID"];
	        this.kind = source["kind"];
	        this.scheduledAt = source["scheduledAt"];
	        this.payload = source["payload"];
	        this.status = source["status"];
	        this.note = source["note"];
	        this.createdAt = source["createdAt"];
	    }
	}
	export class CalendarEventsResult {
	    ok: boolean;
	    events: CalendarEventDTO[];
	    detail: string;
	
	    static createFrom(source: any = {}) {
	        return new CalendarEventsResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.events = this.convertValues(source["events"], CalendarEventDTO);
	        this.detail = source["detail"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CapDeltaDTO {
	    franchiseID: string;
	    franchiseName: string;
	    amount: string;
	    cents: number;
	    reason: string;
	
	    static createFrom(source: any = {}) {
	        return new CapDeltaDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.franchiseID = source["franchiseID"];
	        this.franchiseName = source["franchiseName"];
	        this.amount = source["amount"];
	        this.cents = source["cents"];
	        this.reason = source["reason"];
	    }
	}
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
	    feed: Report;
	    transactions: Report;
	    events: CommsEvent[];
	    boardNote: string;
	    leagueTalk: string;
	    boardPosts: Report[];
	    team: string;
	    kind: string;
	    teams: RaceChoice[];
	    kinds: RaceChoice[];
	
	    static createFrom(source: any = {}) {
	        return new CommsBoard(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.feed = this.convertValues(source["feed"], Report);
	        this.transactions = this.convertValues(source["transactions"], Report);
	        this.events = this.convertValues(source["events"], CommsEvent);
	        this.boardNote = source["boardNote"];
	        this.leagueTalk = source["leagueTalk"];
	        this.boardPosts = this.convertValues(source["boardPosts"], Report);
	        this.team = source["team"];
	        this.kind = source["kind"];
	        this.teams = this.convertValues(source["teams"], RaceChoice);
	        this.kinds = this.convertValues(source["kinds"], RaceChoice);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
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
	export class ContractsReadings {
	    table: Reading_github_com_secureprospective_TheWarRoom_internal_snapshot_ContractTable_;
	    cap: Reading_github_com_secureprospective_TheWarRoom_internal_snapshot_CapSummary_;
	    adjustments: Reading_github_com_secureprospective_TheWarRoom_internal_snapshot_SalaryAdjustments_;
	
	    static createFrom(source: any = {}) {
	        return new ContractsReadings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.table = this.convertValues(source["table"], Reading_github_com_secureprospective_TheWarRoom_internal_snapshot_ContractTable_);
	        this.cap = this.convertValues(source["cap"], Reading_github_com_secureprospective_TheWarRoom_internal_snapshot_CapSummary_);
	        this.adjustments = this.convertValues(source["adjustments"], Reading_github_com_secureprospective_TheWarRoom_internal_snapshot_SalaryAdjustments_);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
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
	    health: Report;
	    signals: Report;
	    facts: Report;
	    checks: Report;
	    archiveDir: string;
	    photosOn: boolean;
	    scopes: DataScope[];
	    words: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new DataSources(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.health = this.convertValues(source["health"], Report);
	        this.signals = this.convertValues(source["signals"], Report);
	        this.facts = this.convertValues(source["facts"], Report);
	        this.checks = this.convertValues(source["checks"], Report);
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
	export class FeedEventDTO {
	    stableKey: string;
	    source: string;
	    id: string;
	    kind: string;
	    timestamp: string;
	    mflID: string;
	    playerName: string;
	    playerPosition: string;
	    playerUnknown: boolean;
	    franchiseIDs: string[];
	    franchiseNames: string[];
	    reason: string;
	    provenance: string;
	    tradeRationale?: string;
	    tradePicksNote?: string;
	    txID: string;
	    correctionStatus?: string;
	    correctionReason?: string;
	    correctionNote?: string;
	    correctedBy?: string;
	    correctedAt?: string;
	
	    static createFrom(source: any = {}) {
	        return new FeedEventDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.stableKey = source["stableKey"];
	        this.source = source["source"];
	        this.id = source["id"];
	        this.kind = source["kind"];
	        this.timestamp = source["timestamp"];
	        this.mflID = source["mflID"];
	        this.playerName = source["playerName"];
	        this.playerPosition = source["playerPosition"];
	        this.playerUnknown = source["playerUnknown"];
	        this.franchiseIDs = source["franchiseIDs"];
	        this.franchiseNames = source["franchiseNames"];
	        this.reason = source["reason"];
	        this.provenance = source["provenance"];
	        this.tradeRationale = source["tradeRationale"];
	        this.tradePicksNote = source["tradePicksNote"];
	        this.txID = source["txID"];
	        this.correctionStatus = source["correctionStatus"];
	        this.correctionReason = source["correctionReason"];
	        this.correctionNote = source["correctionNote"];
	        this.correctedBy = source["correctedBy"];
	        this.correctedAt = source["correctedAt"];
	    }
	}
	export class FeedResult {
	    ok: boolean;
	    events: FeedEventDTO[];
	    detail: string;
	    directoryWarning?: string;
	
	    static createFrom(source: any = {}) {
	        return new FeedResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.events = this.convertValues(source["events"], FeedEventDTO);
	        this.detail = source["detail"];
	        this.directoryWarning = source["directoryWarning"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class M4Franchise {
	    franchiseID: string;
	    name: string;
	    playerCount: number;
	
	    static createFrom(source: any = {}) {
	        return new M4Franchise(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.franchiseID = source["franchiseID"];
	        this.name = source["name"];
	        this.playerCount = source["playerCount"];
	    }
	}
	export class FranchisesResult {
	    ok: boolean;
	    franchises: M4Franchise[];
	    detail: string;
	
	    static createFrom(source: any = {}) {
	        return new FranchisesResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.franchises = this.convertValues(source["franchises"], M4Franchise);
	        this.detail = source["detail"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class M4Player {
	    mflID: string;
	    name: string;
	    position: string;
	    rosterStatus: string;
	    salary: number;
	    capSalary: number;
	
	    static createFrom(source: any = {}) {
	        return new M4Player(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mflID = source["mflID"];
	        this.name = source["name"];
	        this.position = source["position"];
	        this.rosterStatus = source["rosterStatus"];
	        this.salary = source["salary"];
	        this.capSalary = source["capSalary"];
	    }
	}
	export class FreeAgentPoolResult {
	    ok: boolean;
	    players: M4Player[];
	    warning: string;
	    detail: string;
	
	    static createFrom(source: any = {}) {
	        return new FreeAgentPoolResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.players = this.convertValues(source["players"], M4Player);
	        this.warning = source["warning"];
	        this.detail = source["detail"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ScheduleMatchupDTO {
	    homeFranchiseID: string;
	    homeFranchiseName: string;
	    homeScore: string;
	    awayFranchiseID: string;
	    awayFranchiseName: string;
	    awayScore: string;
	
	    static createFrom(source: any = {}) {
	        return new ScheduleMatchupDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.homeFranchiseID = source["homeFranchiseID"];
	        this.homeFranchiseName = source["homeFranchiseName"];
	        this.homeScore = source["homeScore"];
	        this.awayFranchiseID = source["awayFranchiseID"];
	        this.awayFranchiseName = source["awayFranchiseName"];
	        this.awayScore = source["awayScore"];
	    }
	}
	export class ScheduleWeekDTO {
	    week: number;
	    matchups: ScheduleMatchupDTO[];
	
	    static createFrom(source: any = {}) {
	        return new ScheduleWeekDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.week = source["week"];
	        this.matchups = this.convertValues(source["matchups"], ScheduleMatchupDTO);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class LeagueScheduleResult {
	    ok: boolean;
	    weeks: ScheduleWeekDTO[];
	    freshness: domain.Freshness;
	    detail: string;
	
	    static createFrom(source: any = {}) {
	        return new LeagueScheduleResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.weeks = this.convertValues(source["weeks"], ScheduleWeekDTO);
	        this.freshness = this.convertValues(source["freshness"], domain.Freshness);
	        this.detail = source["detail"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class LeagueSettingResult {
	    ok: boolean;
	    error: string;
	    value: string;
	
	    static createFrom(source: any = {}) {
	        return new LeagueSettingResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.error = source["error"];
	        this.value = source["value"];
	    }
	}
	export class LegalOpsResult {
	    ok: boolean;
	    phase: string;
	    kinds: string[];
	    detail: string;
	
	    static createFrom(source: any = {}) {
	        return new LegalOpsResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.phase = source["phase"];
	        this.kinds = source["kinds"];
	        this.detail = source["detail"];
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
	    box: Report;
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
	        this.box = this.convertValues(source["box"], Report);
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
	
	
	export class MFLKeyStatus {
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
	        this.state = source["state"];
	        this.league = source["league"];
	        this.season = source["season"];
	        this.verifiedAt = source["verifiedAt"];
	        this.detail = source["detail"];
	    }
	}
	
	export class MissingMeasure {
	    name: string;
	    meaning: string;
	
	    static createFrom(source: any = {}) {
	        return new MissingMeasure(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.meaning = source["meaning"];
	    }
	}
	export class MoveDTO {
	    mflID: string;
	    toFranchiseID: string;
	
	    static createFrom(source: any = {}) {
	        return new MoveDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mflID = source["mflID"];
	        this.toFranchiseID = source["toFranchiseID"];
	    }
	}
	export class NowBoard {
	    reportWeek: number;
	    reportWeeks: RaceChoice[];
	    week: number;
	    weeks: RaceChoice[];
	    matchup: Report;
	    board: ReportBoard;
	
	    static createFrom(source: any = {}) {
	        return new NowBoard(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.reportWeek = source["reportWeek"];
	        this.reportWeeks = this.convertValues(source["reportWeeks"], RaceChoice);
	        this.week = source["week"];
	        this.weeks = this.convertValues(source["weeks"], RaceChoice);
	        this.matchup = this.convertValues(source["matchup"], Report);
	        this.board = this.convertValues(source["board"], ReportBoard);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
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
	    bait: Report;
	
	    static createFrom(source: any = {}) {
	        return new PersonalReading(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.entries = this.convertValues(source["entries"], personal.Entry);
	        this.players = this.convertValues(source["players"], PersonalPlayer);
	        this.bait = this.convertValues(source["bait"], Report);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PhaseResult {
	    ok: boolean;
	    phase: string;
	    detail: string;
	
	    static createFrom(source: any = {}) {
	        return new PhaseResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.phase = source["phase"];
	        this.detail = source["detail"];
	    }
	}
	
	export class PlayerScoreDTO {
	    mflID: string;
	    name: string;
	    position: string;
	    franchiseID: string;
	    franchiseName: string;
	    basePoints: number;
	    agePull: number;
	    filmEffective: number;
	    rasEffective: number;
	    breakoutEffective: number;
	    l4Combined: number;
	    scoutingAdjusted: number;
	    adjustedScore: number;
	    salary: number;
	    capMultiplier: number;
	    capTier: string;
	    capEff: number;
	    capEffOK: boolean;
	    isVeteran: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PlayerScoreDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mflID = source["mflID"];
	        this.name = source["name"];
	        this.position = source["position"];
	        this.franchiseID = source["franchiseID"];
	        this.franchiseName = source["franchiseName"];
	        this.basePoints = source["basePoints"];
	        this.agePull = source["agePull"];
	        this.filmEffective = source["filmEffective"];
	        this.rasEffective = source["rasEffective"];
	        this.breakoutEffective = source["breakoutEffective"];
	        this.l4Combined = source["l4Combined"];
	        this.scoutingAdjusted = source["scoutingAdjusted"];
	        this.adjustedScore = source["adjustedScore"];
	        this.salary = source["salary"];
	        this.capMultiplier = source["capMultiplier"];
	        this.capTier = source["capTier"];
	        this.capEff = source["capEff"];
	        this.capEffOK = source["capEffOK"];
	        this.isVeteran = source["isVeteran"];
	    }
	}
	export class PlayerScoreResult {
	    ok: boolean;
	    found: boolean;
	    error: string;
	    warning: string;
	    label: string;
	    player: PlayerScoreDTO;
	
	    static createFrom(source: any = {}) {
	        return new PlayerScoreResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.found = source["found"];
	        this.error = source["error"];
	        this.warning = source["warning"];
	        this.label = source["label"];
	        this.player = this.convertValues(source["player"], PlayerScoreDTO);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PowerRow {
	    rank: number;
	    franchiseID: string;
	    name: string;
	    rankDelta: number;
	    deltaOK: boolean;
	    powerScore: number;
	    rosterZ: number;
	    mflPerfZ: number;
	    ageZ: number;
	    rosterValue: number;
	    results: number;
	    age: number;
	    capRoom: number;
	    deadCap: number;
	    hasCap: boolean;
	    luck: number;
	    hasLuck: boolean;
	    projW: number;
	    projL: number;
	    hasProj: boolean;
	    h2hW: number;
	    h2hL: number;
	    h2hT: number;
	    allPlayW: number;
	    allPlayL: number;
	    allPlayT: number;
	    pf: number;
	    pa: number;
	    pp: number;
	    pwr: number;
	    altPwr: number;
	
	    static createFrom(source: any = {}) {
	        return new PowerRow(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rank = source["rank"];
	        this.franchiseID = source["franchiseID"];
	        this.name = source["name"];
	        this.rankDelta = source["rankDelta"];
	        this.deltaOK = source["deltaOK"];
	        this.powerScore = source["powerScore"];
	        this.rosterZ = source["rosterZ"];
	        this.mflPerfZ = source["mflPerfZ"];
	        this.ageZ = source["ageZ"];
	        this.rosterValue = source["rosterValue"];
	        this.results = source["results"];
	        this.age = source["age"];
	        this.capRoom = source["capRoom"];
	        this.deadCap = source["deadCap"];
	        this.hasCap = source["hasCap"];
	        this.luck = source["luck"];
	        this.hasLuck = source["hasLuck"];
	        this.projW = source["projW"];
	        this.projL = source["projL"];
	        this.hasProj = source["hasProj"];
	        this.h2hW = source["h2hW"];
	        this.h2hL = source["h2hL"];
	        this.h2hT = source["h2hT"];
	        this.allPlayW = source["allPlayW"];
	        this.allPlayL = source["allPlayL"];
	        this.allPlayT = source["allPlayT"];
	        this.pf = source["pf"];
	        this.pa = source["pa"];
	        this.pp = source["pp"];
	        this.pwr = source["pwr"];
	        this.altPwr = source["altPwr"];
	    }
	}
	export class PowerRankingsResult {
	    ok: boolean;
	    error: string;
	    label: string;
	    season: number;
	    view: string;
	    weight: number;
	    weightAuto: boolean;
	    ageWeight: number;
	    modelRunID: number;
	    previousRunID: number;
	    aggMode: string;
	    starterN: number;
	    freshness: domain.Freshness;
	    performance: string;
	    weeksScored: number;
	    seasonWeeks: number;
	    rows: PowerRow[];
	
	    static createFrom(source: any = {}) {
	        return new PowerRankingsResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.error = source["error"];
	        this.label = source["label"];
	        this.season = source["season"];
	        this.view = source["view"];
	        this.weight = source["weight"];
	        this.weightAuto = source["weightAuto"];
	        this.ageWeight = source["ageWeight"];
	        this.modelRunID = source["modelRunID"];
	        this.previousRunID = source["previousRunID"];
	        this.aggMode = source["aggMode"];
	        this.starterN = source["starterN"];
	        this.freshness = this.convertValues(source["freshness"], domain.Freshness);
	        this.performance = source["performance"];
	        this.weeksScored = source["weeksScored"];
	        this.seasonWeeks = source["seasonWeeks"];
	        this.rows = this.convertValues(source["rows"], PowerRow);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class RaceBoard {
	    standings: Report;
	    board: ReportBoard;
	    week: number;
	    weeks: RaceChoice[];
	    team: string;
	    teams: RaceChoice[];
	
	    static createFrom(source: any = {}) {
	        return new RaceBoard(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.standings = this.convertValues(source["standings"], Report);
	        this.board = this.convertValues(source["board"], ReportBoard);
	        this.week = source["week"];
	        this.weeks = this.convertValues(source["weeks"], RaceChoice);
	        this.team = source["team"];
	        this.teams = this.convertValues(source["teams"], RaceChoice);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class RankRow {
	    rank: number;
	    mflID: string;
	    name: string;
	    position: string;
	    franchiseID: string;
	    franchiseName: string;
	    salary: number;
	    basePoints: number;
	    adjustedScore: number;
	    capEff: number;
	    capEffOK: boolean;
	    rankDelta: number;
	    deltaOK: boolean;
	    nowPPG: number;
	    dynastyPPG: number;
	    modelOK: boolean;
	
	    static createFrom(source: any = {}) {
	        return new RankRow(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rank = source["rank"];
	        this.mflID = source["mflID"];
	        this.name = source["name"];
	        this.position = source["position"];
	        this.franchiseID = source["franchiseID"];
	        this.franchiseName = source["franchiseName"];
	        this.salary = source["salary"];
	        this.basePoints = source["basePoints"];
	        this.adjustedScore = source["adjustedScore"];
	        this.capEff = source["capEff"];
	        this.capEffOK = source["capEffOK"];
	        this.rankDelta = source["rankDelta"];
	        this.deltaOK = source["deltaOK"];
	        this.nowPPG = source["nowPPG"];
	        this.dynastyPPG = source["dynastyPPG"];
	        this.modelOK = source["modelOK"];
	    }
	}
	export class RankingsResult {
	    ok: boolean;
	    error: string;
	    warning: string;
	    label: string;
	    season: number;
	    runID: number;
	    asOf: string;
	    missingMeasures: MissingMeasure[];
	    freshness: domain.Freshness;
	    rows: RankRow[];
	    modelRunID: number;
	
	    static createFrom(source: any = {}) {
	        return new RankingsResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.error = source["error"];
	        this.warning = source["warning"];
	        this.label = source["label"];
	        this.season = source["season"];
	        this.runID = source["runID"];
	        this.asOf = source["asOf"];
	        this.missingMeasures = this.convertValues(source["missingMeasures"], MissingMeasure);
	        this.freshness = this.convertValues(source["freshness"], domain.Freshness);
	        this.rows = this.convertValues(source["rows"], RankRow);
	        this.modelRunID = source["modelRunID"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
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
	export class RawBidInput {
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
	export class Reading___main_ContractActView_ {
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
	        return new Reading___main_ContractActView_(source);
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
	export class Reading___main_ReleasePlanView_ {
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
	        return new Reading___main_ReleasePlanView_(source);
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
	
	
	export class Reading_github_com_secureprospective_TheWarRoom_internal_snapshot_PlayerContract_ {
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
	        return new Reading_github_com_secureprospective_TheWarRoom_internal_snapshot_PlayerContract_(source);
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
	
	export class TradeAsset {
	    team?: string;
	    contract?: string;
	    now?: AssetCell;
	    dynasty?: AssetCell;
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
	        this.now = this.convertValues(source["now"], AssetCell);
	        this.dynasty = this.convertValues(source["dynasty"], AssetCell);
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
	    cuts: TradeAsset[];
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
	        this.cuts = this.convertValues(source["cuts"], TradeAsset);
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
	export class Reading_main_AcquisitionView_ {
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
	        return new Reading_main_AcquisitionView_(source);
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
	export class BidBoard {
	    plans: BidPlan[];
	    summary: string;
	
	    static createFrom(source: any = {}) {
	        return new BidBoard(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.plans = this.convertValues(source["plans"], BidPlan);
	        this.summary = source["summary"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Reading_main_BidBoard_ {
	    // Go type: BidBoard
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
	        return new Reading_main_BidBoard_(source);
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
	export class Reading_main_Counterparties_ {
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
	        return new Reading_main_Counterparties_(source);
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
	export class MarketLine {
	    key: string;
	    label: string;
	    detail: string;
	    count: number;
	    range?: MarketRange;
	    mark: string;
	
	    static createFrom(source: any = {}) {
	        return new MarketLine(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.label = source["label"];
	        this.detail = source["detail"];
	        this.count = source["count"];
	        this.range = this.convertValues(source["range"], MarketRange);
	        this.mark = source["mark"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class MarketBreakdown {
	    title: string;
	    count: number;
	    rows: MarketLine[];
	
	    static createFrom(source: any = {}) {
	        return new MarketBreakdown(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.count = source["count"];
	        this.rows = this.convertValues(source["rows"], MarketLine);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class MarketSection {
	    breakdowns?: MarketBreakdown[];
	    id: string;
	    title: string;
	    note: string;
	    rows: MarketLine[];
	
	    static createFrom(source: any = {}) {
	        return new MarketSection(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.breakdowns = this.convertValues(source["breakdowns"], MarketBreakdown);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.note = source["note"];
	        this.rows = this.convertValues(source["rows"], MarketLine);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class MarketSurface {
	    summary: string;
	    sections: MarketSection[];
	
	    static createFrom(source: any = {}) {
	        return new MarketSurface(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.summary = source["summary"];
	        this.sections = this.convertValues(source["sections"], MarketSection);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Reading_main_MarketSurface_ {
	    // Go type: MarketSurface
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
	        return new Reading_main_MarketSurface_(source);
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
	export class OfferOutcomeView {
	    id: string;
	    label: string;
	    text: string;
	    width: number;
	
	    static createFrom(source: any = {}) {
	        return new OfferOutcomeView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.label = source["label"];
	        this.text = source["text"];
	        this.width = source["width"];
	    }
	}
	export class OfferStretchView {
	    id: string;
	    label: string;
	    countText: string;
	    acceptance: string;
	    finding: string;
	    outcomes: OfferOutcomeView[];
	
	    static createFrom(source: any = {}) {
	        return new OfferStretchView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.label = source["label"];
	        this.countText = source["countText"];
	        this.acceptance = source["acceptance"];
	        this.finding = source["finding"];
	        this.outcomes = this.convertValues(source["outcomes"], OfferOutcomeView);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class OfferChoice {
	    id: string;
	    label: string;
	    from: number;
	    to: number;
	
	    static createFrom(source: any = {}) {
	        return new OfferChoice(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.label = source["label"];
	        this.from = source["from"];
	        this.to = source["to"];
	    }
	}
	export class OfferWords {
	    heading: string;
	    direction: string;
	    seasons: string;
	    window: string;
	    team: string;
	    retry: string;
	    pending: string;
	
	    static createFrom(source: any = {}) {
	        return new OfferWords(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.heading = source["heading"];
	        this.direction = source["direction"];
	        this.seasons = source["seasons"];
	        this.window = source["window"];
	        this.team = source["team"];
	        this.retry = source["retry"];
	        this.pending = source["pending"];
	    }
	}
	export class OffersView {
	    // Go type: OfferWords
	    words: any;
	    disclosure: string;
	    title: string;
	    summary: string;
	    countText: string;
	    directions: OfferChoice[];
	    windows: OfferChoice[];
	    ranges: OfferChoice[];
	    teams: TeamOption[];
	    stretches: OfferStretchView[];
	
	    static createFrom(source: any = {}) {
	        return new OffersView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.words = this.convertValues(source["words"], null);
	        this.disclosure = source["disclosure"];
	        this.title = source["title"];
	        this.summary = source["summary"];
	        this.countText = source["countText"];
	        this.directions = this.convertValues(source["directions"], OfferChoice);
	        this.windows = this.convertValues(source["windows"], OfferChoice);
	        this.ranges = this.convertValues(source["ranges"], OfferChoice);
	        this.teams = this.convertValues(source["teams"], TeamOption);
	        this.stretches = this.convertValues(source["stretches"], OfferStretchView);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Reading_main_OffersView_ {
	    // Go type: OffersView
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
	        return new Reading_main_OffersView_(source);
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
	export class MarketFact {
	    label: string;
	    text: string;
	    reason: string;
	    range?: MarketRange;
	
	    static createFrom(source: any = {}) {
	        return new MarketFact(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.label = source["label"];
	        this.text = source["text"];
	        this.reason = source["reason"];
	        this.range = this.convertValues(source["range"], MarketRange);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PoolRow {
	    engine?: number;
	    ppg?: number;
	    lab?: number;
	    engineText: string;
	    ppgText: string;
	    labText: string;
	    marketLines: string[];
	    market: MarketFact[];
	    id: string;
	    name: string;
	    position: string;
	    team: string;
	    age: string;
	    experience: string;
	    minimum: snapshot.ContractAmount;
	    plan: string;
	    planId: string;
	
	    static createFrom(source: any = {}) {
	        return new PoolRow(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.engine = source["engine"];
	        this.ppg = source["ppg"];
	        this.lab = source["lab"];
	        this.engineText = source["engineText"];
	        this.ppgText = source["ppgText"];
	        this.labText = source["labText"];
	        this.marketLines = source["marketLines"];
	        this.market = this.convertValues(source["market"], MarketFact);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.position = source["position"];
	        this.team = source["team"];
	        this.age = source["age"];
	        this.experience = source["experience"];
	        this.minimum = this.convertValues(source["minimum"], snapshot.ContractAmount);
	        this.plan = source["plan"];
	        this.planId = source["planId"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PoolSort {
	    key: string;
	    label: string;
	
	    static createFrom(source: any = {}) {
	        return new PoolSort(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.label = source["label"];
	    }
	}
	export class Pool {
	    engineNote: string;
	    ppgNote: string;
	    sorts: PoolSort[];
	    rows: PoolRow[];
	    positions: string[];
	    summary: string;
	
	    static createFrom(source: any = {}) {
	        return new Pool(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.engineNote = source["engineNote"];
	        this.ppgNote = source["ppgNote"];
	        this.sorts = this.convertValues(source["sorts"], PoolSort);
	        this.rows = this.convertValues(source["rows"], PoolRow);
	        this.positions = source["positions"];
	        this.summary = source["summary"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Reading_main_Pool_ {
	    // Go type: Pool
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
	        return new Reading_main_Pool_(source);
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
	export class Reading_main_TeamCounterpart_ {
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
	        return new Reading_main_TeamCounterpart_(source);
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
	export class TradeProjection {
	    team: string;
	    figures: WhatIfFigure[];
	    roster: TradeAsset[];
	
	    static createFrom(source: any = {}) {
	        return new TradeProjection(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.team = source["team"];
	        this.figures = this.convertValues(source["figures"], WhatIfFigure);
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
	export class Reading_main_TradeBuilderView_ {
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
	        return new Reading_main_TradeBuilderView_(source);
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
	export class WhatIfMoveView {
	    id: string;
	    player: string;
	    tag: string;
	    status: string;
	    label: string;
	    charge: snapshot.ContractAmount;
	    explanation: string;
	    plan: string;
	
	    static createFrom(source: any = {}) {
	        return new WhatIfMoveView(source);
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
	    moves: WhatIfMoveView[];
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
	        this.moves = this.convertValues(source["moves"], WhatIfMoveView);
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
	export class Reading_main_WhatIfView_ {
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
	        return new Reading_main_WhatIfView_(source);
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
	export class RefreshResult {
	    ok: boolean;
	    error: string;
	    season: number;
	    rulesChanged: boolean;
	    changed: boolean;
	    players: number;
	
	    static createFrom(source: any = {}) {
	        return new RefreshResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.error = source["error"];
	        this.season = source["season"];
	        this.rulesChanged = source["rulesChanged"];
	        this.changed = source["changed"];
	        this.players = source["players"];
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
	    board: ReportBoard;
	    news: Report;
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
	        this.board = this.convertValues(source["board"], ReportBoard);
	        this.news = this.convertValues(source["news"], Report);
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
	    report: Report;
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
	        this.report = this.convertValues(source["report"], Report);
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
	export class RosterResult {
	    ok: boolean;
	    franchiseID: string;
	    capUsed: number;
	    players: M4Player[];
	    warning: string;
	    detail: string;
	
	    static createFrom(source: any = {}) {
	        return new RosterResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.franchiseID = source["franchiseID"];
	        this.capUsed = source["capUsed"];
	        this.players = this.convertValues(source["players"], M4Player);
	        this.warning = source["warning"];
	        this.detail = source["detail"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	export class ScoreLeagueResult {
	    ok: boolean;
	    error: string;
	    warning: string;
	    label: string;
	    report: rankings.Report;
	    model: modelrun.Report;
	
	    static createFrom(source: any = {}) {
	        return new ScoreLeagueResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.error = source["error"];
	        this.warning = source["warning"];
	        this.label = source["label"];
	        this.report = this.convertValues(source["report"], rankings.Report);
	        this.model = this.convertValues(source["model"], modelrun.Report);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
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
	export class SetLeagueSettingResult {
	    ok: boolean;
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new SetLeagueSettingResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.error = source["error"];
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
	
	
	
	export class TradeOffer {
	    tradeId: string;
	    direction: string;
	    otherId: string;
	    otherName: string;
	    give: TradeAsset[];
	    get: TradeAsset[];
	    // Go type: time
	    expires: any;
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
	        this.expires = this.convertValues(source["expires"], null);
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
	export class TransactionRequest {
	    kind: string;
	    moves: MoveDTO[];
	    picksNote: string;
	    rationale: string;
	    mflID: string;
	    status: string;
	    moveMillions: string;
	    addedYears: number;
	    toPhase: string;
	    note: string;
	    franchiseID: string;
	    amountMillions: string;
	    reason: string;
	    salaryMillions: string;
	    years: number;
	    windowOpen: boolean;
	    tradeDeadline: string;
	    eventID: string;
	    eventKind: string;
	    scheduledAt: string;
	    payload: string;
	
	    static createFrom(source: any = {}) {
	        return new TransactionRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.moves = this.convertValues(source["moves"], MoveDTO);
	        this.picksNote = source["picksNote"];
	        this.rationale = source["rationale"];
	        this.mflID = source["mflID"];
	        this.status = source["status"];
	        this.moveMillions = source["moveMillions"];
	        this.addedYears = source["addedYears"];
	        this.toPhase = source["toPhase"];
	        this.note = source["note"];
	        this.franchiseID = source["franchiseID"];
	        this.amountMillions = source["amountMillions"];
	        this.reason = source["reason"];
	        this.salaryMillions = source["salaryMillions"];
	        this.years = source["years"];
	        this.windowOpen = source["windowOpen"];
	        this.tradeDeadline = source["tradeDeadline"];
	        this.eventID = source["eventID"];
	        this.eventKind = source["eventKind"];
	        this.scheduledAt = source["scheduledAt"];
	        this.payload = source["payload"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class TransactionResult {
	    ok: boolean;
	    kind: string;
	    playersAffected: number;
	    at: string;
	    detail: string;
	    capDeltas: CapDeltaDTO[];
	
	    static createFrom(source: any = {}) {
	        return new TransactionResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.kind = source["kind"];
	        this.playersAffected = source["playersAffected"];
	        this.at = source["at"];
	        this.detail = source["detail"];
	        this.capDeltas = this.convertValues(source["capDeltas"], CapDeltaDTO);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
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

export namespace modelrun {
	
	export class Exclusion {
	    mflID: string;
	    name: string;
	    reason: string;
	
	    static createFrom(source: any = {}) {
	        return new Exclusion(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mflID = source["mflID"];
	        this.name = source["name"];
	        this.reason = source["reason"];
	    }
	}
	export class Report {
	    runID: number;
	    unchanged: boolean;
	    scored: number;
	    rosteredScored: number;
	    freeAgentScored: number;
	    rosteredExcluded: number;
	    freeAgentExcluded: number;
	    rookies: number;
	    excluded: Exclusion[];
	    mislinked: Exclusion[];
	
	    static createFrom(source: any = {}) {
	        return new Report(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.runID = source["runID"];
	        this.unchanged = source["unchanged"];
	        this.scored = source["scored"];
	        this.rosteredScored = source["rosteredScored"];
	        this.freeAgentScored = source["freeAgentScored"];
	        this.rosteredExcluded = source["rosteredExcluded"];
	        this.freeAgentExcluded = source["freeAgentExcluded"];
	        this.rookies = source["rookies"];
	        this.excluded = this.convertValues(source["excluded"], Exclusion);
	        this.mislinked = this.convertValues(source["mislinked"], Exclusion);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
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

export namespace rankings {
	
	export class Exclusion {
	    mflID: string;
	    name: string;
	    franchiseID: string;
	    franchiseName: string;
	    reason: string;
	
	    static createFrom(source: any = {}) {
	        return new Exclusion(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mflID = source["mflID"];
	        this.name = source["name"];
	        this.franchiseID = source["franchiseID"];
	        this.franchiseName = source["franchiseName"];
	        this.reason = source["reason"];
	    }
	}
	export class Report {
	    season: number;
	    runID: number;
	    unchanged: boolean;
	    scored: number;
	    zeroBase: number;
	    negativeBase: number;
	    missingMeasures: string[];
	    excluded: Exclusion[];
	
	    static createFrom(source: any = {}) {
	        return new Report(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.season = source["season"];
	        this.runID = source["runID"];
	        this.unchanged = source["unchanged"];
	        this.scored = source["scored"];
	        this.zeroBase = source["zeroBase"];
	        this.negativeBase = source["negativeBase"];
	        this.missingMeasures = source["missingMeasures"];
	        this.excluded = this.convertValues(source["excluded"], Exclusion);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
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

