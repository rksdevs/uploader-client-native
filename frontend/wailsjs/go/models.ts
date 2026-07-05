export namespace main {
	
	export class CombatLogFileActivity {
	    fileExists: boolean;
	    currentSize: number;
	    baselineSize: number;
	    pendingBytes: number;
	    hasPendingChanges: boolean;
	    lastModified?: string;
	    lastModifiedUnix?: number;
	    lastLinePreview?: string;
	    wowRunning: boolean;
	    wowClosedReady: boolean;
	    wowClosedDetail?: string;
	    fileStable: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CombatLogFileActivity(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.fileExists = source["fileExists"];
	        this.currentSize = source["currentSize"];
	        this.baselineSize = source["baselineSize"];
	        this.pendingBytes = source["pendingBytes"];
	        this.hasPendingChanges = source["hasPendingChanges"];
	        this.lastModified = source["lastModified"];
	        this.lastModifiedUnix = source["lastModifiedUnix"];
	        this.lastLinePreview = source["lastLinePreview"];
	        this.wowRunning = source["wowRunning"];
	        this.wowClosedReady = source["wowClosedReady"];
	        this.wowClosedDetail = source["wowClosedDetail"];
	        this.fileStable = source["fileStable"];
	    }
	}
	export class AutoUploadWatcherStatus {
	    running: boolean;
	    status: string;
	    detail?: string;
	    lastStagingBytes?: number;
	    lastStagingAt?: string;
	    stagingPath?: string;
	    fileActivity: CombatLogFileActivity;
	
	    static createFrom(source: any = {}) {
	        return new AutoUploadWatcherStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.running = source["running"];
	        this.status = source["status"];
	        this.detail = source["detail"];
	        this.lastStagingBytes = source["lastStagingBytes"];
	        this.lastStagingAt = source["lastStagingAt"];
	        this.stagingPath = source["stagingPath"];
	        this.fileActivity = this.convertValues(source["fileActivity"], CombatLogFileActivity);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class AutoUploadSettingsResponse {
	    enabled: boolean;
	    defaultServer: string;
	    deviceId: string;
	    hasBaseline: boolean;
	    baselineEstablishedAt?: string;
	    tailFingerprint?: string[];
	    logDirectory: string;
	    hasApiToken: boolean;
	    canEnable: boolean;
	    serverAllowed: boolean;
	    blockReason?: string;
	    watcher: AutoUploadWatcherStatus;
	    minimizeToTray: boolean;
	
	    static createFrom(source: any = {}) {
	        return new AutoUploadSettingsResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.defaultServer = source["defaultServer"];
	        this.deviceId = source["deviceId"];
	        this.hasBaseline = source["hasBaseline"];
	        this.baselineEstablishedAt = source["baselineEstablishedAt"];
	        this.tailFingerprint = source["tailFingerprint"];
	        this.logDirectory = source["logDirectory"];
	        this.hasApiToken = source["hasApiToken"];
	        this.canEnable = source["canEnable"];
	        this.serverAllowed = source["serverAllowed"];
	        this.blockReason = source["blockReason"];
	        this.watcher = this.convertValues(source["watcher"], AutoUploadWatcherStatus);
	        this.minimizeToTray = source["minimizeToTray"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class BaselinePreview {
	    lines: string[];
	    baselineEstablishedAt: string;
	    sourceFileSize: number;
	    lastByteOffset: number;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new BaselinePreview(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.lines = source["lines"];
	        this.baselineEstablishedAt = source["baselineEstablishedAt"];
	        this.sourceFileSize = source["sourceFileSize"];
	        this.lastByteOffset = source["lastByteOffset"];
	        this.message = source["message"];
	    }
	}
	
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

