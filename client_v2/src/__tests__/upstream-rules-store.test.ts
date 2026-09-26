import { beforeEach, describe, expect, it, vi } from 'vitest';

import {
    getUpstreamRuleGroups,
    saveUpstreamRuleGroup,
    testUpstreamRule,
    upstreamRulesState,
} from 'panel/stores/upstreamRules';

const mocks = vi.hoisted(() => ({
    getStatus: vi.fn(),
    saveGroup: vi.fn(),
    testRule: vi.fn(),
    addErrorToast: vi.fn(),
    addSuccessToast: vi.fn(),
}));

vi.mock('panel/api/generated', () => ({
    upstreamRuleGroupsStatus: mocks.getStatus,
    upstreamRuleGroupSave: mocks.saveGroup,
    upstreamRuleTest: mocks.testRule,
}));

vi.mock('panel/stores/toasts', () => ({
    addErrorToast: mocks.addErrorToast,
    addSuccessToast: mocks.addSuccessToast,
}));

describe('upstream rule groups store', () => {
    beforeEach(() => vi.clearAllMocks());

    it('loads groups, default upstreams, and compile issues', async () => {
        mocks.getStatus.mockResolvedValue({
            default_upstreams: ['https://default.example/dns-query'],
            groups: [
                {
                    id: 1,
                    name: 'Remote rules',
                    enabled: true,
                    priority: 10,
                    kind: 'subscription',
                    upstreams: ['https://foreign.example/dns-query'],
                },
            ],
            issues: [{ group_id: 1, group_name: 'Remote rules', line: 8, message: 'bad rule' }],
        });

        await getUpstreamRuleGroups();

        expect(upstreamRulesState.defaultUpstreams).toEqual([
            'https://default.example/dns-query',
        ]);
        expect(upstreamRulesState.groups[0]?.name).toBe('Remote rules');
        expect(upstreamRulesState.issues[0]?.line).toBe(8);
        expect(upstreamRulesState.processing).toBe(false);
    });

    it('saves a group and refreshes the list', async () => {
        const group = {
            name: 'Custom',
            enabled: true,
            priority: 20,
            kind: 'custom' as const,
            upstreams: ['192.0.2.53'],
            rules: '||example.com',
        };
        mocks.saveGroup.mockResolvedValue({ ...group, id: 2 });
        mocks.getStatus.mockResolvedValue({ default_upstreams: [], groups: [], issues: [] });

        const ok = await saveUpstreamRuleGroup(group);

        expect(ok).toBe(true);
        expect(mocks.saveGroup).toHaveBeenCalledWith(group);
        expect(mocks.getStatus).toHaveBeenCalled();
    });

    it('stores the latest domain match result', async () => {
        mocks.testRule.mockResolvedValue({
            matched: true,
            group_id: 1,
            group_name: 'Remote rules',
            upstreams: ['https://foreign.example/dns-query'],
        });

        await testUpstreamRule('www.example.com');

        expect(mocks.testRule).toHaveBeenCalledWith({ domain: 'www.example.com' });
        expect(upstreamRulesState.testResult?.group_name).toBe('Remote rules');
    });
});
