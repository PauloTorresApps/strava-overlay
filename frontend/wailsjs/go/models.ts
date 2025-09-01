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
	export class Activity {
	    id: number;
	    name: string;
	    type: string;
	    // Go type: time
	    start_date: any;
	    distance: number;
	    moving_time: number;
	    max_speed: number;
	    has_heartrate: boolean;
	    start_latlng: number[];
	    end_latlng: number[];
	    map: Map;
	
	    static createFrom(source: any = {}) {
	        return new Activity(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.type = source["type"];
	        this.start_date = this.convertValues(source["start_date"], null);
	        this.distance = source["distance"];
	        this.moving_time = source["moving_time"];
	        this.max_speed = source["max_speed"];
	        this.has_heartrate = source["has_heartrate"];
	        this.start_latlng = source["start_latlng"];
	        this.end_latlng = source["end_latlng"];
	        this.map = this.convertValues(source["map"], Map);
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
	export class ActivityDetail {
	    id: number;
	    name: string;
	    type: string;
	    // Go type: time
	    start_date: any;
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

