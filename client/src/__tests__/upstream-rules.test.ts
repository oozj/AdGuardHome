import { describe, expect, test } from 'vitest';

import {
    buildUpstreamRuleGroupRows,
    formatRulesCount,
    getUpstreamRuleGroupErrors,
    normalizeUpstreams,
    summarizeUpstreams,
    sortUpstreamRuleGroups,
} from '../components/Settings/UpstreamRules/helpers';
import { UPSTREAM_RULE_SYNTAX_ROWS } from '../components/Settings/UpstreamRules/ruleSyntax';

describe('classic upstream rule groups helpers', () => {
    test('normalizes one upstream per line', () => {
        expect(normalizeUpstreams(' 1.1.1.1\n\nhttps://dns.example/dns-query \n')).toEqual([
            '1.1.1.1',
            'https://dns.example/dns-query',
        ]);
    });

    test('sorts rule groups by priority without mutating input', () => {
        const groups = [
            { id: 2, priority: 20 },
            { id: 1, priority: 10 },
        ];

        expect(sortUpstreamRuleGroups(groups).map((group) => group.id)).toEqual([1, 2]);
        expect(groups.map((group) => group.id)).toEqual([2, 1]);
        expect(sortUpstreamRuleGroups(null)).toEqual([]);
    });

    test('keeps the non-removable default group at the end', () => {
        const rows = buildUpstreamRuleGroupRows(
            [
                {
                    id: 1,
                    name: 'Custom',
                    enabled: true,
                    priority: 10,
                    kind: 'custom',
                    upstreams: ['192.0.2.1'],
                },
            ],
            ['1.1.1.1'],
            'Default group',
        );

        expect(rows[0].id).toBe(1);
        expect(rows[1]).toMatchObject({ id: 0, kind: 'default', name: 'Default group' });
        expect(rows[1].upstreams).toEqual(['1.1.1.1']);
    });

    test('summarizes long upstream lists without rendering the whole list', () => {
        expect(summarizeUpstreams(['one', 'two', 'three', 'four'])).toBe('one, two (+2)');
        expect(summarizeUpstreams(['one'])).toBe('one');
    });

    test('combines subscription and compile errors for a group', () => {
        const errors = getUpstreamRuleGroupErrors(
            {
                id: 7,
                name: 'Remote',
                enabled: true,
                priority: 10,
                kind: 'subscription',
                upstreams: ['1.1.1.1'],
                last_error: 'download failed',
            },
            [{ group_id: 7, group_name: 'Remote', line: 3, message: 'invalid rule' }],
        );

        expect(errors).toEqual(['download failed', 'Line 3: invalid rule']);
    });

    test('renders an omitted rule count as zero', () => {
        expect(formatRulesCount('custom', undefined)).toBe(0);
        expect(formatRulesCount('subscription', 12)).toBe(12);
        expect(formatRulesCount('default', undefined)).toBe('—');
    });

    test('documents every supported custom-rule family', () => {
        expect(UPSTREAM_RULE_SYNTAX_ROWS).toHaveLength(11);
        expect(UPSTREAM_RULE_SYNTAX_ROWS.every((row) => row.exampleKey && row.descriptionKey)).toBe(true);
        expect(UPSTREAM_RULE_SYNTAX_ROWS.map((row) => row.syntax)).toEqual(
            expect.arrayContaining(['google', '||google', '||google^', '@@||safe.example.com^', '*']),
        );
    });
});
