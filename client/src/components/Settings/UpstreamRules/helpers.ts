import { UpstreamRuleGroup, UpstreamRuleGroupsStatus } from './types';

export const normalizeUpstreams = (value: string): string[] =>
    value
        .split('\n')
        .map((line) => line.trim())
        .filter(Boolean);

export const sortUpstreamRuleGroups = <T extends { priority: number }>(groups?: T[] | null): T[] =>
    [...(groups || [])].sort((left, right) => left.priority - right.priority);

export const buildUpstreamRuleGroupRows = (
    groups: UpstreamRuleGroup[] | null | undefined,
    defaultUpstreams: string[] | null | undefined,
    defaultName: string,
): UpstreamRuleGroup[] => [
    ...sortUpstreamRuleGroups(groups),
    {
        id: 0,
        name: defaultName,
        enabled: true,
        priority: 0,
        kind: 'default',
        upstreams: defaultUpstreams || [],
        rules_count: 0,
    },
];

export const summarizeUpstreams = (upstreams: string[] | null | undefined): string => {
    const addresses = upstreams || [];
    const summary = addresses.slice(0, 2).join(', ');

    return addresses.length > 2 ? `${summary} (+${addresses.length - 2})` : summary;
};

export const getUpstreamRuleGroupErrors = (
    group: UpstreamRuleGroup,
    issues: UpstreamRuleGroupsStatus['issues'] | null | undefined,
): string[] => [
    ...(group.last_error ? [group.last_error] : []),
    ...(issues || [])
        .filter((issue) => issue.group_id === group.id)
        .map((issue) => `Line ${issue.line}: ${issue.message}`),
];

export const formatRulesCount = (
    kind: UpstreamRuleGroup['kind'],
    count: number | null | undefined,
): number | string => (kind === 'default' ? '—' : (count ?? 0));
