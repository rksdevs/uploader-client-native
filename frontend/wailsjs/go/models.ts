export namespace main {
	
	export class InstancePreview {
	    loggedBy: string;
	    bosses: string[];
	    formattedStartTime: string;
	    formattedEndTime: string;
	    detectedServerName?: string;
	    detectedGuidPrefix?: string;
	
	    static createFrom(source: any = {}) {
	        return new InstancePreview(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.loggedBy = source["loggedBy"];
	        this.bosses = source["bosses"];
	        this.formattedStartTime = source["formattedStartTime"];
	        this.formattedEndTime = source["formattedEndTime"];
	        this.detectedServerName = source["detectedServerName"];
	        this.detectedGuidPrefix = source["detectedGuidPrefix"];
	    }
	}
	export class Instance {
	    name: string;
	    encounterStartTime: string;
	    startMs: number;
	    endMs: number;
	    lineStart: number;
	    lineEnd: number;
	    serverName?: string;
	    serverVerified?: boolean;
	    preview?: InstancePreview;
	
	    static createFrom(source: any = {}) {
	        return new Instance(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.encounterStartTime = source["encounterStartTime"];
	        this.startMs = source["startMs"];
	        this.endMs = source["endMs"];
	        this.lineStart = source["lineStart"];
	        this.lineEnd = source["lineEnd"];
	        this.serverName = source["serverName"];
	        this.serverVerified = source["serverVerified"];
	        this.preview = this.convertValues(source["preview"], InstancePreview);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class PreprocessResponse {
	    message: string;
	    preprocessId: number;
	    instances: Instance[];
	    autoQueued: boolean;
	    hasMultipleDetectedServers: boolean;
	    viewLogURL: string;
	
	    static createFrom(source: any = {}) {
	        return new PreprocessResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.message = source["message"];
	        this.preprocessId = source["preprocessId"];
	        this.instances = this.convertValues(source["instances"], Instance);
	        this.autoQueued = source["autoQueued"];
	        this.hasMultipleDetectedServers = source["hasMultipleDetectedServers"];
	        this.viewLogURL = source["viewLogURL"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class UploaderServer {
	    id: number;
	    value: string;
	    label: string;
	
	    static createFrom(source: any = {}) {
	        return new UploaderServer(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.value = source["value"];
	        this.label = source["label"];
	    }
	}

}

