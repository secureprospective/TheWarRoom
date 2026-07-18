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
	export class RankRow {
	    rank: number;
	    mflID: string;
	    name: string;
	    position: string;
	    franchiseID: string;
	    salary: number;
	    basePoints: number;
	    agePull: number;
	    l4Combined: number;
	    capTier: string;
	    adjustedScore: number;
	    capEff: number;
	    capEffOK: boolean;
	
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
	        this.agePull = source["agePull"];
	        this.l4Combined = source["l4Combined"];
	        this.capTier = source["capTier"];
	        this.adjustedScore = source["adjustedScore"];
	        this.capEff = source["capEff"];
	        this.capEffOK = source["capEffOK"];
	    }
	}
	export class RankingsResult {
	    ok: boolean;
	    error: string;
	    warning: string;
	    label: string;
	    season: number;
	    configVersion: number;
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
	
	    static createFrom(source: any = {}) {
	        return new TransactionRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.moves = this.convertValues(source["moves"], MoveDTO);
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
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
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

