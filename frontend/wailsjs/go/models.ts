export namespace main {
	
	export class AddonDatasetSync {
	    dataset: string;
	    label: string;
	    rows: number;
	    realm?: string;
	    season?: number;
	    syncedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new AddonDatasetSync(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.dataset = source["dataset"];
	        this.label = source["label"];
	        this.rows = source["rows"];
	        this.realm = source["realm"];
	        this.season = source["season"];
	        this.syncedAt = source["syncedAt"];
	    }
	}
	export class AddonStatus {
	    wowDirectory: string;
	    folderFound: boolean;
	    addonInstalled: boolean;
	    installedVersion?: string;
	    latestVersion?: string;
	    updateAvailable: boolean;
	    downloadUrl: string;
	    datasets: AddonDatasetSync[];
	
	    static createFrom(source: any = {}) {
	        return new AddonStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.wowDirectory = source["wowDirectory"];
	        this.folderFound = source["folderFound"];
	        this.addonInstalled = source["addonInstalled"];
	        this.installedVersion = source["installedVersion"];
	        this.latestVersion = source["latestVersion"];
	        this.updateAvailable = source["updateAvailable"];
	        this.downloadUrl = source["downloadUrl"];
	        this.datasets = this.convertValues(source["datasets"], AddonDatasetSync);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
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
	
	export class DesktopAccount {
	    signedIn: boolean;
	    source: string;
	    tokenType: string;
	    isPremium: boolean;
	    reason?: string;
	    userName?: string;
	    avatarUrl?: string;
	
	    static createFrom(source: any = {}) {
	        return new DesktopAccount(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.signedIn = source["signedIn"];
	        this.source = source["source"];
	        this.tokenType = source["tokenType"];
	        this.isPremium = source["isPremium"];
	        this.reason = source["reason"];
	        this.userName = source["userName"];
	        this.avatarUrl = source["avatarUrl"];
	    }
	}
	export class DesktopSettings {
	    minimizeToTray: boolean;
	    startWithWindows: boolean;
	    desktopNotifications: boolean;
	
	    static createFrom(source: any = {}) {
	        return new DesktopSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.minimizeToTray = source["minimizeToTray"];
	        this.startWithWindows = source["startWithWindows"];
	        this.desktopNotifications = source["desktopNotifications"];
	    }
	}
	export class HistoryEntry {
	    preprocessId: number;
	    source: string;
	    serverName: string;
	    raids: string[];
	    bosses: string[];
	    startedAt: string;
	    durationMs: number;
	    status: string;
	    message?: string;
	    logIds: number[];
	    unlisted?: boolean;
	    retryPath?: string;
	
	    static createFrom(source: any = {}) {
	        return new HistoryEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.preprocessId = source["preprocessId"];
	        this.source = source["source"];
	        this.serverName = source["serverName"];
	        this.raids = source["raids"];
	        this.bosses = source["bosses"];
	        this.startedAt = source["startedAt"];
	        this.durationMs = source["durationMs"];
	        this.status = source["status"];
	        this.message = source["message"];
	        this.logIds = source["logIds"];
	        this.unlisted = source["unlisted"];
	        this.retryPath = source["retryPath"];
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
	    unlisted?: boolean;
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
	        this.unlisted = source["unlisted"];
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
	
	export class MergeFile {
	    path: string;
	    name: string;
	    sizeBytes: number;
	    firstMs: number;
	    lastMs: number;
	    bosses: string[];
	    realms: string[];
	    header?: string;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new MergeFile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.name = source["name"];
	        this.sizeBytes = source["sizeBytes"];
	        this.firstMs = source["firstMs"];
	        this.lastMs = source["lastMs"];
	        this.bosses = source["bosses"];
	        this.realms = source["realms"];
	        this.header = source["header"];
	        this.error = source["error"];
	    }
	}
	export class MergeLink {
	    gapMs: number;
	    overlap: boolean;
	    nextDay: boolean;
	
	    static createFrom(source: any = {}) {
	        return new MergeLink(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.gapMs = source["gapMs"];
	        this.overlap = source["overlap"];
	        this.nextDay = source["nextDay"];
	    }
	}
	export class MergePlan {
	    files: MergeFile[];
	    links: MergeLink[];
	    bosses: string[];
	    totalBytes: number;
	    activeMs: number;
	    spanMs: number;
	    multiDay: boolean;
	    warnings: string[];
	    canMerge: boolean;
	
	    static createFrom(source: any = {}) {
	        return new MergePlan(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.files = this.convertValues(source["files"], MergeFile);
	        this.links = this.convertValues(source["links"], MergeLink);
	        this.bosses = source["bosses"];
	        this.totalBytes = source["totalBytes"];
	        this.activeMs = source["activeMs"];
	        this.spanMs = source["spanMs"];
	        this.multiDay = source["multiDay"];
	        this.warnings = source["warnings"];
	        this.canMerge = source["canMerge"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
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
	export class UpdateInfo {
	    available: boolean;
	    required: boolean;
	    current: string;
	    version: string;
	    notes?: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.available = source["available"];
	        this.required = source["required"];
	        this.current = source["current"];
	        this.version = source["version"];
	        this.notes = source["notes"];
	    }
	}
	export class UploadPreview {
	    path: string;
	    exists: boolean;
	    sizeBytes: number;
	    modifiedAt?: string;
	    hasPrevious: boolean;
	    newBytes: number;
	    wowRunning: boolean;
	    tooLarge: boolean;
	
	    static createFrom(source: any = {}) {
	        return new UploadPreview(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.exists = source["exists"];
	        this.sizeBytes = source["sizeBytes"];
	        this.modifiedAt = source["modifiedAt"];
	        this.hasPrevious = source["hasPrevious"];
	        this.newBytes = source["newBytes"];
	        this.wowRunning = source["wowRunning"];
	        this.tooLarge = source["tooLarge"];
	    }
	}
	export class UploadRequest {
	    path: string;
	    logDirectory: string;
	    serverName: string;
	    unlisted: boolean;
	    newContentOnly: boolean;
	
	    static createFrom(source: any = {}) {
	        return new UploadRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.logDirectory = source["logDirectory"];
	        this.serverName = source["serverName"];
	        this.unlisted = source["unlisted"];
	        this.newContentOnly = source["newContentOnly"];
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

