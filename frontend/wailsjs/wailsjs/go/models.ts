export namespace app {
	
	export class AppInfo {
	    appName: string;
	    version: string;
	    goVersion: string;
	    platform: string;
	
	    static createFrom(source: any = {}) {
	        return new AppInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.appName = source["appName"];
	        this.version = source["version"];
	        this.goVersion = source["goVersion"];
	        this.platform = source["platform"];
	    }
	}
	export class BrowserState {
	    name: string;
	    ready: boolean;
	    url: string;
	    profile: string;
	
	    static createFrom(source: any = {}) {
	        return new BrowserState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.ready = source["ready"];
	        this.url = source["url"];
	        this.profile = source["profile"];
	    }
	}
	export class RuntimeStatus {
	    pythonPath: string;
	    ytDlpPath: string;
	    ytDlpVersion: string;
	    ffmpegPath: string;
	    ffmpegOK: boolean;
	    dbPath: string;
	    dbOK: boolean;
	
	    static createFrom(source: any = {}) {
	        return new RuntimeStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pythonPath = source["pythonPath"];
	        this.ytDlpPath = source["ytDlpPath"];
	        this.ytDlpVersion = source["ytDlpVersion"];
	        this.ffmpegPath = source["ffmpegPath"];
	        this.ffmpegOK = source["ffmpegOK"];
	        this.dbPath = source["dbPath"];
	        this.dbOK = source["dbOK"];
	    }
	}

}

export namespace config {
	
	export class RuntimeConfig {
	    ytDlpPath: string;
	    ffmpegPath: string;
	    pythonPath: string;
	
	    static createFrom(source: any = {}) {
	        return new RuntimeConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ytDlpPath = source["ytDlpPath"];
	        this.ffmpegPath = source["ffmpegPath"];
	        this.pythonPath = source["pythonPath"];
	    }
	}
	export class NetworkConfig {
	    proxy: string;
	    cookieFile: string;
	    userAgent: string;
	
	    static createFrom(source: any = {}) {
	        return new NetworkConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.proxy = source["proxy"];
	        this.cookieFile = source["cookieFile"];
	        this.userAgent = source["userAgent"];
	    }
	}
	export class DownloadConfig {
	    maxConcurrency: number;
	    retryCount: number;
	    downloadDirectory: string;
	
	    static createFrom(source: any = {}) {
	        return new DownloadConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.maxConcurrency = source["maxConcurrency"];
	        this.retryCount = source["retryCount"];
	        this.downloadDirectory = source["downloadDirectory"];
	    }
	}
	export class Config {
	    download: DownloadConfig;
	    network: NetworkConfig;
	    runtime: RuntimeConfig;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.download = this.convertValues(source["download"], DownloadConfig);
	        this.network = this.convertValues(source["network"], NetworkConfig);
	        this.runtime = this.convertValues(source["runtime"], RuntimeConfig);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
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

export namespace tiktok {
	
	export class SearchOptions {
	    maxResults: number;
	    maxNoNewRounds: number;
	    scrollDelayMs: number;
	    pageReadyTimeoutMs: number;
	
	    static createFrom(source: any = {}) {
	        return new SearchOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.maxResults = source["maxResults"];
	        this.maxNoNewRounds = source["maxNoNewRounds"];
	        this.scrollDelayMs = source["scrollDelayMs"];
	        this.pageReadyTimeoutMs = source["pageReadyTimeoutMs"];
	    }
	}
	export class VideoCandidate {
	    videoId: string;
	    url: string;
	    authorId: string;
	    authorName: string;
	    title: string;
	    publishTime: number;
	    source: string;
	    keyword: string;
	
	    static createFrom(source: any = {}) {
	        return new VideoCandidate(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.videoId = source["videoId"];
	        this.url = source["url"];
	        this.authorId = source["authorId"];
	        this.authorName = source["authorName"];
	        this.title = source["title"];
	        this.publishTime = source["publishTime"];
	        this.source = source["source"];
	        this.keyword = source["keyword"];
	    }
	}

}

