import { createEffect, createMemo, createSignal, onMount, Show } from 'solid-js';
import { useNavigate, useParams } from '@solidjs/router';
import cn from 'clsx';

import intl from 'panel/common/intl';
import { Input } from 'panel/common/controls/Input';
import { Textarea } from 'panel/common/controls/Textarea';
import { Button } from 'panel/common/ui/Button';
import { PageLoader } from 'panel/common/ui/Loader';
import { Paths } from 'panel/components/Routes/Paths';
import {
    getUpstreamRuleGroups,
    saveUpstreamRuleGroup,
    testUpstreamRule,
    upstreamRulesState,
} from 'panel/stores/upstreamRules';
import theme from 'panel/lib/theme';
import s from './UpstreamRules.module.pcss';

export const UpstreamRuleEditor = () => {
    const params = useParams<{ groupId: string }>();
    const navigate = useNavigate();
    const [rules, setRules] = createSignal('');
    const [domain, setDomain] = createSignal('');
    const [initialized, setInitialized] = createSignal(false);
    const group = createMemo(() =>
        upstreamRulesState.groups.find((item) => item.id === Number(params.groupId)),
    );

    onMount(() => void getUpstreamRuleGroups());
    createEffect(() => {
        const current = group();
        if (current && !initialized()) {
            setRules(current.rules ?? '');
            setInitialized(true);
        }
    });

    const saveRules = async () => {
        const current = group();
        if (!current) return;
        await saveUpstreamRuleGroup({ ...current, rules: rules() });
    };

    return (
        <Show when={!upstreamRulesState.processing && group()} fallback={<PageLoader />}>
            {(current) => (
                <div class={theme.layout.container}>
                    <div class={s.editorGrid}>
                        <div class={s.editorMain}>
                            <button
                                type="button"
                                class={cn(theme.link.link, s.back)}
                                onClick={() => navigate(Paths.UpstreamRules)}
                            >
                                ← {intl.getMessage('upstream_rule_groups_title')}
                            </button>
                            <h1 class={cn(theme.layout.title, theme.title.h4, theme.title.h3_tablet)}>
                                {current().name}
                            </h1>
                            <p class={cn(theme.text.t2, s.description)}>
                                {intl.getMessage('upstream_rule_editor_desc')}
                            </p>
                            <Textarea
                                id="upstream_rule_editor"
                                rows={22}
                                size="large"
                                value={rules()}
                                onChange={(event) => setRules(event.currentTarget.value)}
                                highlightComments
                                class={s.ruleEditor}
                            />
                            <div class={s.editorActions}>
                                <Button
                                    variant="primary"
                                    onClick={saveRules}
                                    disabled={upstreamRulesState.processingAction}
                                >
                                    {intl.getMessage('save')}
                                </Button>
                            </div>
                        </div>

                        <aside class={s.testCard}>
                            <h2 class={theme.title.h6}>
                                {intl.getMessage('upstream_rule_test_title')}
                            </h2>
                            <Input
                                id="upstream_rule_test_domain"
                                value={domain()}
                                placeholder="www.example.com"
                                onChange={(event) => setDomain(event.currentTarget.value)}
                            />
                            <Button
                                variant="secondary"
                                compact
                                disabled={!domain().trim() || upstreamRulesState.processingTest}
                                onClick={() => void testUpstreamRule(domain().trim())}
                            >
                                {intl.getMessage('user_rules_check_button')}
                            </Button>
                            <Show when={upstreamRulesState.testResult}>
                                {(result) => (
                                    <div class={s.testResult}>
                                        <strong>
                                            {result().matched
                                                ? result().group_name
                                                : intl.getMessage('upstream_rule_no_match')}
                                        </strong>
                                        <Show when={result().upstreams?.length}>
                                            <div class={s.servers}>{result().upstreams?.join('\n')}</div>
                                        </Show>
                                    </div>
                                )}
                            </Show>
                        </aside>
                    </div>
                </div>
            )}
        </Show>
    );
};
