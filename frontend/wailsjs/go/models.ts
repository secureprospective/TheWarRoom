export namespace main {
	
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

}

