export type UpstreamRuleSyntaxRow = {
    syntax: string;
    exampleKey: string;
    descriptionKey: string;
};

export const UPSTREAM_RULE_SYNTAX_ROWS: UpstreamRuleSyntaxRow[] = [
    {
        syntax: 'example.com',
        exampleKey: 'upstream_rules_help_example_plain_domain',
        descriptionKey: 'upstream_rules_help_plain_domain',
    },
    {
        syntax: 'google',
        exampleKey: 'upstream_rules_help_example_keyword',
        descriptionKey: 'upstream_rules_help_keyword',
    },
    {
        syntax: '||example.com^',
        exampleKey: 'upstream_rules_help_example_domain_anchor',
        descriptionKey: 'upstream_rules_help_domain_anchor',
    },
    {
        syntax: '||google',
        exampleKey: 'upstream_rules_help_example_anchor_prefix',
        descriptionKey: 'upstream_rules_help_anchor_prefix',
    },
    {
        syntax: '||google^',
        exampleKey: 'upstream_rules_help_example_anchor_label',
        descriptionKey: 'upstream_rules_help_anchor_label',
    },
    {
        syntax: 'cdn*.example.com',
        exampleKey: 'upstream_rules_help_example_glob',
        descriptionKey: 'upstream_rules_help_glob',
    },
    {
        syntax: '|https://www.example.com/path',
        exampleKey: 'upstream_rules_help_example_url',
        descriptionKey: 'upstream_rules_help_url',
    },
    {
        syntax: '/^api\\d+\\.example\\.com$/',
        exampleKey: 'upstream_rules_help_example_regexp',
        descriptionKey: 'upstream_rules_help_regexp',
    },
    {
        syntax: '@@||safe.example.com^',
        exampleKey: 'upstream_rules_help_example_exception',
        descriptionKey: 'upstream_rules_help_exception',
    },
    {
        syntax: '*',
        exampleKey: 'upstream_rules_help_example_all',
        descriptionKey: 'upstream_rules_help_all',
    },
    {
        syntax: '! note / # note / [Header]',
        exampleKey: 'upstream_rules_help_example_comment',
        descriptionKey: 'upstream_rules_help_comment',
    },
];
