export interface UpstreamRuleTestResponse {
    matched: boolean;
    group_id?: number;
    group_name?: string;
    upstreams?: string[];
}
