export type UpstreamRuleGroupKind = 'custom' | 'subscription' | 'default';

export type UpstreamRuleGroup = {
    id: number;
    name: string;
    enabled: boolean;
    priority: number;
    kind: UpstreamRuleGroupKind;
    url?: string;
    upstreams: string[];
    rules?: string;
    rules_count?: number;
    last_updated?: string;
    last_error?: string;
};

export type UpstreamRuleGroupsStatus = {
    default_upstreams: string[];
    groups: UpstreamRuleGroup[];
    issues: Array<{
        group_id: number;
        group_name: string;
        line: number;
        message: string;
    }>;
};

export type UpstreamRuleTestResponse = {
    matched: boolean;
    group_id?: number;
    group_name?: string;
    upstreams?: string[];
};
