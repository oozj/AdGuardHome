import type { UpstreamRuleGroupKind } from './upstreamRuleGroupKind';

export interface UpstreamRuleGroup {
    id?: number;
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
}
