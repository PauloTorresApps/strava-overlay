export namespace handlers {
	
	export class AuthStatus {
	    is_authenticated: boolean;
	    message: string;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new AuthStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.is_authenticated = source["is_authenticated"];
	        this.message = source["message"];
	        this.error = source["error"];
	    }
	}
	export class FrontendActivity {
	    id: number;
	    name: string;
	    type: string;
	    start_date: string;
	    distance: number;
	    moving_time: number;
	    max_speed: number;
	    start_latlng: number[];
	    end_latlng: number[];
	    map: strava.Map;
	    has_gps: boolean;
	
	    static createFrom(source: any = {}) {
	        return new FrontendActivity(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.type = source["type"];
	        this.start_date = source["start_date"];
	        this.distance = source["distance"];
	        this.moving_time = source["moving_time"];
	        this.max_speed = source["max_speed"];
	        this.start_latlng = source["start_latlng"];
	        this.end_latlng = source["end_latlng"];
	        this.map = this.convertValues(source["map"], strava.Map);
	        this.has_gps = source["has_gps"];
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
	export class FrontendGPSPoint {
	    time: string;
	    lat: number;
	    lng: number;
	    velocity: number;
	    altitude: number;
	    bearing: number;
	    gForce: number;
	
	    static createFrom(source: any = {}) {
	        return new FrontendGPSPoint(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.time = source["time"];
	        this.lat = source["lat"];
	        this.lng = source["lng"];
	        this.velocity = source["velocity"];
	        this.altitude = source["altitude"];
	        this.bearing = source["bearing"];
	        this.gForce = source["gForce"];
	    }
	}
	export class PaginatedActivities {
	    activities: FrontendActivity[];
	    page: number;
	    per_page: number;
	    has_more: boolean;
	    total_loaded: number;
	
	    static createFrom(source: any = {}) {
	        return new PaginatedActivities(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.activities = this.convertValues(source["activities"], FrontendActivity);
	        this.page = source["page"];
	        this.per_page = source["per_page"];
	        this.has_more = source["has_more"];
	        this.total_loaded = source["total_loaded"];
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

export namespace strava {
	
	export class Map {
	    id: string;
	    polyline: string;
	    summary_polyline: string;
	
	    static createFrom(source: any = {}) {
	        return new Map(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.polyline = source["polyline"];
	        this.summary_polyline = source["summary_polyline"];
	    }
	}
	export class ActivityDetail {
	    id: number;
	    name: string;
	    type: string;
	    // Go type: time
	    start_date: any;
	    timezone: string;
	    distance: number;
	    moving_time: number;
	    max_speed: number;
	    has_heartrate: boolean;
	    start_latlng: number[];
	    end_latlng: number[];
	    map: Map;
	    calories: number;
	    total_elevation_gain: number;
	
	    static createFrom(source: any = {}) {
	        return new ActivityDetail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.type = source["type"];
	        this.start_date = this.convertValues(source["start_date"], null);
	        this.timezone = source["timezone"];
	        this.distance = source["distance"];
	        this.moving_time = source["moving_time"];
	        this.max_speed = source["max_speed"];
	        this.has_heartrate = source["has_heartrate"];
	        this.start_latlng = source["start_latlng"];
	        this.end_latlng = source["end_latlng"];
	        this.map = this.convertValues(source["map"], Map);
	        this.calories = source["calories"];
	        this.total_elevation_gain = source["total_elevation_gain"];
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

