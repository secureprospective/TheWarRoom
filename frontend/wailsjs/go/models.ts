export namespace engine {
	
	export class Layer4Output {
	    FilmEffective: number;
	    FilmRaw: number;
	    RASEffective: number;
	    BreakoutEffective: number;
	    Combined: number;
	
	    static createFrom(source: any = {}) {
	        return new Layer4Output(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.FilmEffective = source["FilmEffective"];
	        this.FilmRaw = source["FilmRaw"];
	        this.RASEffective = source["RASEffective"];
	        this.BreakoutEffective = source["BreakoutEffective"];
	        this.Combined = source["Combined"];
	    }
	}
	export class TiebreakerKey {
	    IsVeteran: boolean;
	    RAS: number;
	    ScarcityRank: number;
	
	    static createFrom(source: any = {}) {
	        return new TiebreakerKey(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.IsVeteran = source["IsVeteran"];
	        this.RAS = source["RAS"];
	        this.ScarcityRank = source["ScarcityRank"];
	    }
	}
	export class Result {
	    BasePoints: number;
	    AgePull: number;
	    Layer4Output: Layer4Output;
	    ScoutingAdjusted: number;
	    CapMultiplier: number;
	    CapTier: string;
	    AdjustedScore: number;
	    Tiebreaker: TiebreakerKey;
	
	    static createFrom(source: any = {}) {
	        return new Result(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.BasePoints = source["BasePoints"];
	        this.AgePull = source["AgePull"];
	        this.Layer4Output = this.convertValues(source["Layer4Output"], Layer4Output);
	        this.ScoutingAdjusted = source["ScoutingAdjusted"];
	        this.CapMultiplier = source["CapMultiplier"];
	        this.CapTier = source["CapTier"];
	        this.AdjustedScore = source["AdjustedScore"];
	        this.Tiebreaker = this.convertValues(source["Tiebreaker"], TiebreakerKey);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
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

export namespace harness {
	
	export class CaseResult {
	    id: string;
	    name: string;
	    state: string;
	    detail: string;
	    b5bBlock: string;
	
	    static createFrom(source: any = {}) {
	        return new CaseResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.state = source["state"];
	        this.detail = source["detail"];
	        this.b5bBlock = source["b5bBlock"];
	    }
	}
	export class RookieRow {
	    mflID: string;
	    name: string;
	    position: string;
	    age: number;
	    rasImputed: boolean;
	    result: engine.Result;
	    err: string;
	
	    static createFrom(source: any = {}) {
	        return new RookieRow(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mflID = source["mflID"];
	        this.name = source["name"];
	        this.position = source["position"];
	        this.age = source["age"];
	        this.rasImputed = source["rasImputed"];
	        this.result = this.convertValues(source["result"], engine.Result);
	        this.err = source["err"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Summary {
	    pass: number;
	    fail: number;
	    pending: number;
	    total: number;
	
	    static createFrom(source: any = {}) {
	        return new Summary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pass = source["pass"];
	        this.fail = source["fail"];
	        this.pending = source["pending"];
	        this.total = source["total"];
	    }
	}

}

export namespace main {
	
	export class AppInfo {
	    version: string;
	    commit: string;
	    buildDate: string;
	
	    static createFrom(source: any = {}) {
	        return new AppInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.commit = source["commit"];
	        this.buildDate = source["buildDate"];
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
	export class FranchisePlayerDTO {
	    mflID: string;
	    rosterStatus: string;
	    salary: number;
	    capSalary: number;
	
	    static createFrom(source: any = {}) {
	        return new FranchisePlayerDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mflID = source["mflID"];
	        this.rosterStatus = source["rosterStatus"];
	        this.salary = source["salary"];
	        this.capSalary = source["capSalary"];
	    }
	}
	export class FranchiseStateResult {
	    ok: boolean;
	    franchiseID: string;
	    capUsed: number;
	    players: FranchisePlayerDTO[];
	    detail: string;
	
	    static createFrom(source: any = {}) {
	        return new FranchiseStateResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.franchiseID = source["franchiseID"];
	        this.capUsed = source["capUsed"];
	        this.players = this.convertValues(source["players"], FranchisePlayerDTO);
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
	export class FreeAgentsResult {
	    ok: boolean;
	    mflIDs: string[];
	    detail: string;
	
	    static createFrom(source: any = {}) {
	        return new FreeAgentsResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.mflIDs = source["mflIDs"];
	        this.detail = source["detail"];
	    }
	}
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
	    freshness: Freshness;
	    detail: string;
	
	    static createFrom(source: any = {}) {
	        return new LeagueScheduleResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.weeks = this.convertValues(source["weeks"], ScheduleWeekDTO);
	        this.freshness = this.convertValues(source["freshness"], Freshness);
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
	export class ParamsResult {
	    ok: boolean;
	    error: string;
	    params: params.ParamDef[];
	
	    static createFrom(source: any = {}) {
	        return new ParamsResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.error = source["error"];
	        this.params = this.convertValues(source["params"], params.ParamDef);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
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
	export class PingResult {
	    ok: boolean;
	    message: string;
	    journalMode: string;
	    detail: string;
	
	    static createFrom(source: any = {}) {
	        return new PingResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.message = source["message"];
	        this.journalMode = source["journalMode"];
	        this.detail = source["detail"];
	    }
	}
	export class PlayerScoreDTO {
	    mflID: string;
	    name: string;
	    position: string;
	    franchiseID: string;
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
	    powerScore: number;
	    scoutingZ: number;
	    mflPerfZ: number;
	    scoutingScore: number;
	    allPlayWinPct: number;
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
	        this.powerScore = source["powerScore"];
	        this.scoutingZ = source["scoutingZ"];
	        this.mflPerfZ = source["mflPerfZ"];
	        this.scoutingScore = source["scoutingScore"];
	        this.allPlayWinPct = source["allPlayWinPct"];
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
	    weight: number;
	    aggMode: string;
	    starterN: number;
	    freshness: Freshness;
	    phase: string;
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
	        this.weight = source["weight"];
	        this.aggMode = source["aggMode"];
	        this.starterN = source["starterN"];
	        this.freshness = this.convertValues(source["freshness"], Freshness);
	        this.phase = source["phase"];
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
	
	export class RankRow {
	    rank: number;
	    mflID: string;
	    name: string;
	    position: string;
	    franchiseID: string;
	    salary: number;
	    basePoints: number;
	    adjustedScore: number;
	    capEff: number;
	    capEffOK: boolean;
	    rankDelta: number;
	    deltaOK: boolean;
	
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
	        this.salary = source["salary"];
	        this.basePoints = source["basePoints"];
	        this.adjustedScore = source["adjustedScore"];
	        this.capEff = source["capEff"];
	        this.capEffOK = source["capEffOK"];
	        this.rankDelta = source["rankDelta"];
	        this.deltaOK = source["deltaOK"];
	    }
	}
	export class RankingsResult {
	    ok: boolean;
	    error: string;
	    warning: string;
	    label: string;
	    season: number;
	    configVersion: number;
	    freshness: Freshness;
	    rows: RankRow[];
	
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
	        this.configVersion = source["configVersion"];
	        this.freshness = this.convertValues(source["freshness"], Freshness);
	        this.rows = this.convertValues(source["rows"], RankRow);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class RookiesResult {
	    ok: boolean;
	    error: string;
	    l4Mode: string;
	    rows: harness.RookieRow[];
	
	    static createFrom(source: any = {}) {
	        return new RookiesResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.error = source["error"];
	        this.l4Mode = source["l4Mode"];
	        this.rows = this.convertValues(source["rows"], harness.RookieRow);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
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
	    label: string;
	    report: rankings.Report;
	
	    static createFrom(source: any = {}) {
	        return new ScoreLeagueResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.error = source["error"];
	        this.label = source["label"];
	        this.report = this.convertValues(source["report"], rankings.Report);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
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
	export class ValidationResult {
	    ok: boolean;
	    cases: harness.CaseResult[];
	    summary: harness.Summary;
	
	    static createFrom(source: any = {}) {
	        return new ValidationResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.cases = this.convertValues(source["cases"], harness.CaseResult);
	        this.summary = this.convertValues(source["summary"], harness.Summary);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
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

export namespace params {
	
	export class ParamDef {
	    Key: string;
	    Position: string;
	    Type: string;
	    Default: number;
	    Min: number;
	    Max: number;
	    IsCalibrated: boolean;
	    Description: string;
	
	    static createFrom(source: any = {}) {
	        return new ParamDef(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Key = source["Key"];
	        this.Position = source["Position"];
	        this.Type = source["Type"];
	        this.Default = source["Default"];
	        this.Min = source["Min"];
	        this.Max = source["Max"];
	        this.IsCalibrated = source["IsCalibrated"];
	        this.Description = source["Description"];
	    }
	}

}

export namespace rankings {
	
	export class Exclusion {
	    mflID: string;
	    name: string;
	    franchiseID: string;
	    reason: string;
	
	    static createFrom(source: any = {}) {
	        return new Exclusion(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mflID = source["mflID"];
	        this.name = source["name"];
	        this.franchiseID = source["franchiseID"];
	        this.reason = source["reason"];
	    }
	}
	export class Report {
	    season: number;
	    configVersion: number;
	    scored: number;
	    skippedExisting: boolean;
	    existing: number;
	    zeroBase: number;
	    negativeBase: number;
	    excluded: Exclusion[];
	
	    static createFrom(source: any = {}) {
	        return new Report(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.season = source["season"];
	        this.configVersion = source["configVersion"];
	        this.scored = source["scored"];
	        this.skippedExisting = source["skippedExisting"];
	        this.existing = source["existing"];
	        this.zeroBase = source["zeroBase"];
	        this.negativeBase = source["negativeBase"];
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

