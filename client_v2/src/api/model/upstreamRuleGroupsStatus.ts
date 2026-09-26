import type { UpstreamRuleGroup } from './upstreamRuleGroup';
import type { UpstreamRuleIssue } from './upstreamRuleIssue';

export interface UpstreamRuleGroupsStatus {
    default_upstreams: string[];
    groups: UpstreamRuleGroup[];
    issues: UpstreamRuleIssue[];
}
