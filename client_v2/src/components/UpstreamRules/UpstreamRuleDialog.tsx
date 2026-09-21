import { createEffect, createSignal } from 'solid-js';

import intl from 'panel/common/intl';
import { Button } from 'panel/common/ui/Button';
import { Dialog } from 'panel/common/ui/Dialog';
import { Input } from 'panel/common/controls/Input';
import { Switch } from 'panel/common/controls/Switch';
import { Textarea } from 'panel/common/controls/Textarea';
import { InlineLoader } from 'panel/common/ui/Loader/InlineLoader';
import theme from 'panel/lib/theme';
import { saveUpstreamRuleGroup, upstreamRulesState } from 'panel/stores/upstreamRules';

import type { UpstreamRuleGroup } from 'panel/api/model/upstreamRuleGroup';
import type { UpstreamRuleGroupKind } from 'panel/api/model/upstreamRuleGroupKind';

type Props = {
    visible: boolean;
    group?: UpstreamRuleGroup;
    initialKind: UpstreamRuleGroupKind;
    onClose: () => void;
};

export const UpstreamRuleDialog = (props: Props) => {
    const [name, setName] = createSignal('');
    const [url, setURL] = createSignal('');
    const [upstreams, setUpstreams] = createSignal('');
    const [priority, setPriority] = createSignal(10);
    const [enabled, setEnabled] = createSignal(true);

    const kind = () => props.group?.kind ?? props.initialKind;

    createEffect(() => {
        setName(props.group?.name ?? '');
        setURL(props.group?.url ?? '');
        setUpstreams((props.group?.upstreams ?? []).join('\n'));
        setPriority(props.group?.priority ?? 10);
        setEnabled(props.group?.enabled ?? true);
    });

    const handleSubmit = async (event: Event) => {
        event.preventDefault();
        const servers = upstreams()
            .split('\n')
            .map((server) => server.trim())
            .filter(Boolean);

        if (!name().trim() || servers.length === 0 || (kind() === 'subscription' && !url())) {
            return;
        }

        const saved = await saveUpstreamRuleGroup({
            ...props.group,
            name: name().trim(),
            url: kind() === 'subscription' ? url().trim() : undefined,
            upstreams: servers,
            priority: priority(),
            enabled: enabled(),
            kind: kind(),
            rules: props.group?.rules ?? '',
        });
        if (saved) props.onClose();
    };

    return (
        <Dialog
            visible={props.visible}
            onClose={props.onClose}
            title={
                props.group
                    ? intl.getMessage('upstream_rule_group_edit')
                    : intl.getMessage('upstream_rule_group_add')
            }
        >
            <form onSubmit={handleSubmit}>
                <div class={theme.form.group}>
                    <div class={theme.form.input}>
                        <Input
                            id="upstream_rule_name"
                            label={intl.getMessage('name_label')}
                            value={name()}
                            onChange={(event) => setName(event.currentTarget.value)}
                        />
                    </div>

                    {kind() === 'subscription' ? (
                        <div class={theme.form.input}>
                            <Input
                                id="upstream_rule_url"
                                label={intl.getMessage('url_label')}
                                value={url()}
                                onChange={(event) => setURL(event.currentTarget.value)}
                            />
                        </div>
                    ) : null}

                    <div class={theme.form.input}>
                        <Textarea
                            id="upstream_rule_servers"
                            label={intl.getMessage('upstream_rule_dns_servers')}
                            rows={4}
                            value={upstreams()}
                            placeholder="https://dns.example/dns-query"
                            onChange={(event) => setUpstreams(event.currentTarget.value)}
                        />
                    </div>

                    <div class={theme.form.input}>
                        <Input
                            id="upstream_rule_priority"
                            type="number"
                            label={intl.getMessage('upstream_rule_priority')}
                            value={priority()}
                            onChange={(event) =>
                                setPriority(Number.parseInt(event.currentTarget.value, 10) || 0)
                            }
                        />
                    </div>

                    <Switch
                        id="upstream_rule_enabled"
                        checked={enabled()}
                        onChange={(event) =>
                            setEnabled((event.currentTarget as HTMLInputElement).checked)
                        }
                    >
                        {intl.getMessage('enable')}
                    </Switch>
                </div>

                <div class={theme.dialog.footer}>
                    <Button
                        type="submit"
                        variant="primary"
                        size="small"
                        disabled={upstreamRulesState.processingAction}
                        leftAddon={
                            upstreamRulesState.processingAction ? <InlineLoader /> : undefined
                        }
                        class={theme.dialog.button}
                    >
                        {intl.getMessage('save')}
                    </Button>
                    <Button
                        type="button"
                        variant="secondary"
                        size="small"
                        onClick={props.onClose}
                        class={theme.dialog.button}
                    >
                        {intl.getMessage('cancel')}
                    </Button>
                </div>
            </form>
        </Dialog>
    );
};
