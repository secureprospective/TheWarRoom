export namespace main {
	
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
	export class CrosswalkMiss {
	    mflId: string;
	    name: string;
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
	export class PositionShare {
	    position: string;
	    rostered: number;
	    withData: number;
	
	    static createFrom(source: any = {}) {
	        return new PositionShare(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.position = source["position"];
	        this.rostered = source["rostered"];
	        this.withData = source["withData"];
	    }
	}
	export class SeasonCount {
	    season: number;
	    players: number;
	    waiting: number;
	
	    static createFrom(source: any = {}) {
	        return new SeasonCount(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.season = source["season"];
	        this.players = source["players"];
	        this.waiting = source["waiting"];
	    }
	}
	export class FeedView {
	    feed: string;
	    source: string;
	    lastLoaded: string;
	    fresh: boolean;
	    seasons: SeasonCount[];
	    coverageSeason: number;
	    coverage: PositionShare[];
	
	    static createFrom(source: any = {}) {
	        return new FeedView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.feed = source["feed"];
	        this.source = source["source"];
	        this.lastLoaded = source["lastLoaded"];
	        this.fresh = source["fresh"];
	        this.seasons = this.convertValues(source["seasons"], SeasonCount);
	        this.coverageSeason = source["coverageSeason"];
	        this.coverage = this.convertValues(source["coverage"], PositionShare);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
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
	export class ParamView {
	    key: string;
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
	    freshness: Freshness;
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
	        this.freshness = this.convertValues(source["freshness"], Freshness);
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
	    freshness: Freshness;
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
	        this.freshness = this.convertValues(source["freshness"], Freshness);
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
	export class SignalLoad {
	    feed: string;
	    season: number;
	    status: string;
	    rows: number;
	    facts: number;
	    added: number;
	    unresolved: number;
	    missing: string[];
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new SignalLoad(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.feed = source["feed"];
	        this.season = source["season"];
	        this.status = source["status"];
	        this.rows = source["rows"];
	        this.facts = source["facts"];
	        this.added = source["added"];
	        this.unresolved = source["unresolved"];
	        this.missing = source["missing"];
	        this.error = source["error"];
	    }
	}
	export class SignalsReport {
	    ok: boolean;
	    error: string;
	    startedAt: string;
	    finishedAt: string;
	    loads: SignalLoad[];
	
	    static createFrom(source: any = {}) {
	        return new SignalsReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.error = source["error"];
	        this.startedAt = source["startedAt"];
	        this.finishedAt = source["finishedAt"];
	        this.loads = this.convertValues(source["loads"], SignalLoad);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SourceView {
	    source: string;
	    name: string;
	    state: string;
	    lastSuccess: string;
	    lastError: string;
	
	    static createFrom(source: any = {}) {
	        return new SourceView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.source = source["source"];
	        this.name = source["name"];
	        this.state = source["state"];
	        this.lastSuccess = source["lastSuccess"];
	        this.lastError = source["lastError"];
	    }
	}
	export class SignalsView {
	    ok: boolean;
	    error: string;
	    season: number;
	    sources: SourceView[];
	    feeds: FeedView[];
	    lastLoad: SignalsReport;
	
	    static createFrom(source: any = {}) {
	        return new SignalsView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.error = source["error"];
	        this.season = source["season"];
	        this.sources = this.convertValues(source["sources"], SourceView);
	        this.feeds = this.convertValues(source["feeds"], FeedView);
	        this.lastLoad = this.convertValues(source["lastLoad"], SignalsReport);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
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

