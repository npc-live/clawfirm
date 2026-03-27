export namespace app {
	
	export class ChannelInfo {
	    name: string;
	    provider: string;
	    model: string;
	    sessions: number;
	
	    static createFrom(source: any = {}) {
	        return new ChannelInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.provider = source["provider"];
	        this.model = source["model"];
	        this.sessions = source["sessions"];
	    }
	}
	export class MemoryFile {
	    path: string;
	    name: string;
	    hash: string;
	    modifiedAt: number;
	    indexedAt: number;
	    chunkCount: number;
	
	    static createFrom(source: any = {}) {
	        return new MemoryFile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.name = source["name"];
	        this.hash = source["hash"];
	        this.modifiedAt = source["modifiedAt"];
	        this.indexedAt = source["indexedAt"];
	        this.chunkCount = source["chunkCount"];
	    }
	}
	export class MemorySearchResult {
	    filePath: string;
	    startLine: number;
	    endLine: number;
	    content: string;
	    score: number;
	
	    static createFrom(source: any = {}) {
	        return new MemorySearchResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.filePath = source["filePath"];
	        this.startLine = source["startLine"];
	        this.endLine = source["endLine"];
	        this.content = source["content"];
	        this.score = source["score"];
	    }
	}
	export class ProviderInfo {
	    id: string;
	    type: string;
	    hasKey: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ProviderInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.type = source["type"];
	        this.hasKey = source["hasKey"];
	    }
	}
	export class SessionInfo {
	    agentName: string;
	    channelId: string;
	    userId: string;
	
	    static createFrom(source: any = {}) {
	        return new SessionInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.agentName = source["agentName"];
	        this.channelId = source["channelId"];
	        this.userId = source["userId"];
	    }
	}
	export class SkillInfo {
	    name: string;
	    description: string;
	    filePath: string;
	    source: string;
	
	    static createFrom(source: any = {}) {
	        return new SkillInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.description = source["description"];
	        this.filePath = source["filePath"];
	        this.source = source["source"];
	    }
	}
	export class ToolExecutionInfo {
	    id: string;
	    name: string;
	    args?: any;
	    result?: string;
	    isError: boolean;
	    timestamp: number;
	
	    static createFrom(source: any = {}) {
	        return new ToolExecutionInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.args = source["args"];
	        this.result = source["result"];
	        this.isError = source["isError"];
	        this.timestamp = source["timestamp"];
	    }
	}

}

export namespace config {
	
	export class AgentConfig {
	    name: string;
	    provider: string;
	    model: string;
	    system_prompt: string;
	    max_tokens: number;
	    tools: string[];
	    skill_paths: string[];
	    workspace_dir: string;
	
	    static createFrom(source: any = {}) {
	        return new AgentConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.provider = source["provider"];
	        this.model = source["model"];
	        this.system_prompt = source["system_prompt"];
	        this.max_tokens = source["max_tokens"];
	        this.tools = source["tools"];
	        this.skill_paths = source["skill_paths"];
	        this.workspace_dir = source["workspace_dir"];
	    }
	}
	export class Schedule {
	    kind: string;
	    at?: string;
	    everyMs?: number;
	    anchorMs?: number;
	    expr?: string;
	    tz?: string;
	
	    static createFrom(source: any = {}) {
	        return new Schedule(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.at = source["at"];
	        this.everyMs = source["everyMs"];
	        this.anchorMs = source["anchorMs"];
	        this.expr = source["expr"];
	        this.tz = source["tz"];
	    }
	}
	export class CronJobConfig {
	    name: string;
	    schedule: Schedule;
	    agent_name: string;
	    prompt: string;
	    enabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CronJobConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.schedule = this.convertValues(source["schedule"], Schedule);
	        this.agent_name = source["agent_name"];
	        this.prompt = source["prompt"];
	        this.enabled = source["enabled"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class WhipflowCliProvider {
	    name?: string;
	    bin?: string;
	    prompt_mode?: string;
	    args?: string[];
	    stdin_args?: string[];
	    timeout?: number;
	    output_format?: string;
	    raw_prompt?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new WhipflowCliProvider(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.bin = source["bin"];
	        this.prompt_mode = source["prompt_mode"];
	        this.args = source["args"];
	        this.stdin_args = source["stdin_args"];
	        this.timeout = source["timeout"];
	        this.output_format = source["output_format"];
	        this.raw_prompt = source["raw_prompt"];
	    }
	}
	export class WhipflowConfig {
	    cli_providers?: Record<string, WhipflowCliProvider>;
	    default_provider?: string;
	    tools_dir?: string;
	    tools?: string[];
	
	    static createFrom(source: any = {}) {
	        return new WhipflowConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.cli_providers = this.convertValues(source["cli_providers"], WhipflowCliProvider, true);
	        this.default_provider = source["default_provider"];
	        this.tools_dir = source["tools_dir"];
	        this.tools = source["tools"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class WhatsAppConfig {
	    Enabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new WhatsAppConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Enabled = source["Enabled"];
	    }
	}
	export class FeishuConfig {
	    AppID: string;
	    AppSecret: string;
	
	    static createFrom(source: any = {}) {
	        return new FeishuConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.AppID = source["AppID"];
	        this.AppSecret = source["AppSecret"];
	    }
	}
	export class ProviderConfig {
	    type: string;
	    api_key: string;
	    base_url: string;
	
	    static createFrom(source: any = {}) {
	        return new ProviderConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.api_key = source["api_key"];
	        this.base_url = source["base_url"];
	    }
	}
	export class Config {
	    providers: Record<string, ProviderConfig>;
	    agents: AgentConfig[];
	    default_agent: string;
	    DefaultProvider: string;
	    DefaultModel: string;
	    Feishu: FeishuConfig;
	    WhatsApp: WhatsAppConfig;
	    whipflow: WhipflowConfig;
	    cron_jobs: CronJobConfig[];
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.providers = this.convertValues(source["providers"], ProviderConfig, true);
	        this.agents = this.convertValues(source["agents"], AgentConfig);
	        this.default_agent = source["default_agent"];
	        this.DefaultProvider = source["DefaultProvider"];
	        this.DefaultModel = source["DefaultModel"];
	        this.Feishu = this.convertValues(source["Feishu"], FeishuConfig);
	        this.WhatsApp = this.convertValues(source["WhatsApp"], WhatsAppConfig);
	        this.whipflow = this.convertValues(source["whipflow"], WhipflowConfig);
	        this.cron_jobs = this.convertValues(source["cron_jobs"], CronJobConfig);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
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

export namespace store {
	
	export class ScheduleData {
	    at?: string;
	    everyMs?: number;
	    anchorMs?: number;
	    expr?: string;
	    tz?: string;
	
	    static createFrom(source: any = {}) {
	        return new ScheduleData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.at = source["at"];
	        this.everyMs = source["everyMs"];
	        this.anchorMs = source["anchorMs"];
	        this.expr = source["expr"];
	        this.tz = source["tz"];
	    }
	}
	export class CronJob {
	    id: string;
	    name: string;
	    scheduleKind: string;
	    schedule: ScheduleData;
	    agentName: string;
	    prompt: string;
	    enabled: boolean;
	    // Go type: time
	    createdAt?: any;
	    // Go type: time
	    updatedAt?: any;
	
	    static createFrom(source: any = {}) {
	        return new CronJob(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.scheduleKind = source["scheduleKind"];
	        this.schedule = this.convertValues(source["schedule"], ScheduleData);
	        this.agentName = source["agentName"];
	        this.prompt = source["prompt"];
	        this.enabled = source["enabled"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CronJobHistory {
	    id: number;
	    jobId: string;
	    // Go type: time
	    startedAt: any;
	    // Go type: time
	    finishedAt?: any;
	    status: string;
	    resultText: string;
	    errorText: string;
	
	    static createFrom(source: any = {}) {
	        return new CronJobHistory(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.jobId = source["jobId"];
	        this.startedAt = this.convertValues(source["startedAt"], null);
	        this.finishedAt = this.convertValues(source["finishedAt"], null);
	        this.status = source["status"];
	        this.resultText = source["resultText"];
	        this.errorText = source["errorText"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class VaultEntry {
	    key: string;
	    value: string;
	    updated_at: number;
	
	    static createFrom(source: any = {}) {
	        return new VaultEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.value = source["value"];
	        this.updated_at = source["updated_at"];
	    }
	}

}

