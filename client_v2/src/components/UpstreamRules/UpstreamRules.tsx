import { createMemo, createSignal, onMount, Show } from 'solid-js';
import { useNavigate } from '@solidjs/router';
import cn from 'clsx';

import intl from 'panel/common/intl';
import { Switch } from 'panel/common/controls/Switch';
import { ConfirmDialog } from 'panel/common/ui/ConfirmDialog';
import { Icon } from 'panel/common/ui/Icon';
import { PageLoader } from 'panel/common/ui/Loader';
import { PlusButton } from 'panel/common/ui/PlusButton';
import { Table, type TableColumn } from 'panel/common/ui/Table';
import { Paths } from 'panel/components/Routes/Paths';
import { Upstream } from 'panel/components/DnsSettings/Upstream';
import { getDnsConfig } from 'panel/stores/dnsConfig';
import { initSettings } from 'panel/stores/settings';
import { formatShortDateTime } from 'panel/helpers/helpers';
import {
    deleteUpstreamRuleGroup,
    getUpstreamRuleGroups,
    refreshAllUpstreamRuleGroups,
    refreshUpstreamRuleGroup,
    saveUpstreamRuleGroup,
    upstreamRulesState,
} from 'panel/stores/upstreamRules';
import theme from 'panel/lib/theme';

import type { UpstreamRuleGroup } from 'panel/api/model/upstreamRuleGroup';
import type { UpstreamRuleGroupKind } from 'panel/api/model/upstreamRuleGroupKind';
import { UpstreamRuleDialog } from './UpstreamRuleDialog';
import s from './UpstreamRules.module.pcss';

const countRules = (rules?: string) =>
    (rules || '')
        .split('\n')
        .filter((line) => line.trim() && !line.trim().startsWith('!')).length;

export const UpstreamRules = () => {
    const navigate = useNavigate();
    const [dialogVisible, setDialogVisible] = createSignal(false);
    const [dialogKind, setDialogKind] = createSignal<UpstreamRuleGroupKind>('subscription');
    const [selectedGroup, setSelectedGroup] = createSignal<UpstreamRuleGroup>();
    const [deleteTarget, setDeleteTarget] = createSignal<UpstreamRuleGroup>();

    onMount(() => {
        void Promise.all([getDnsConfig(), initSettings(), getUpstreamRuleGroups()]);
    });

    const openAdd = (kind: UpstreamRuleGroupKind) => {
        setSelectedGroup(undefined);
        setDialogKind(kind);
        setDialogVisible(true);
    };

    const openEdit = (group: UpstreamRuleGroup) => {
        setSelectedGroup(group);
        setDialogKind(group.kind);
        setDialogVisible(true);
    };

    const columns = createMemo<TableColumn<UpstreamRuleGroup>[]>(() => [
        {
            key: 'name',
            header: { text: intl.getMessage('name_label') },
            accessor: 'name',
            sortable: true,
            render: (value: string, group) => (
                <div class={s.nameCell}>
                    <span class={theme.common.textOverflow}>{value}</span>
                    <Switch
                        id={`upstream_group_${group.id}`}
                        checked={group.enabled}
                        disabled={upstreamRulesState.processingAction}
                        onChange={() =>
                            void saveUpstreamRuleGroup({ ...group, enabled: !group.enabled })
                        }
                    />
                </div>
            ),
        },
        {
            key: 'kind',
            header: { text: intl.getMessage('upstream_rule_type') },
            accessor: 'kind',
            sortable: true,
            render: (_value, group) => (
                <div>
                    <strong>
                        {intl.getMessage(
                            group.kind === 'subscription'
                                ? 'upstream_rule_subscription'
                                : 'upstream_rule_custom',
                        )}
                    </strong>
                    <Show when={group.url}>
                        <div class={cn(theme.text.t3, s.secondary)}>{group.url}</div>
                    </Show>
                    <Show when={group.last_error}>
                        <div class={cn(theme.text.t3, s.errorText)}>{group.last_error}</div>
                    </Show>
                    <Show
                        when={
                            upstreamRulesState.issues.filter(
                                (issue) => issue.group_id === group.id,
                            ).length
                        }
                    >
                        {(count) => (
                            <div class={cn(theme.text.t3, s.errorText)}>
                                {intl.getMessage('upstream_rule_issue_count', {
                                    count: count(),
                                })}
                            </div>
                        )}
                    </Show>
                </div>
            ),
        },
        {
            key: 'upstreams',
            header: { text: intl.getMessage('upstream_rule_dns_servers') },
            accessor: (group) => group.upstreams.join(', '),
            sortable: false,
            render: (value: string) => <span class={s.servers}>{value}</span>,
        },
        {
            key: 'rules',
            header: { text: intl.getMessage('rules_label') },
            accessor: (group) => group.rules_count ?? countRules(group.rules),
            sortable: true,
        },
        {
            key: 'last_updated',
            header: { text: intl.getMessage('last_updated_label') },
            accessor: 'last_updated',
            sortable: true,
            render: (value: string) => <span>{value ? formatShortDateTime(value) : '—'}</span>,
        },
        {
            key: 'priority',
            header: { text: intl.getMessage('upstream_rule_priority') },
            accessor: 'priority',
            sortable: true,
            width: 90,
        },
        {
            key: 'actions',
            header: { text: '' },
            sortable: false,
            width: 150,
            render: (_value, group) => (
                <div class={theme.table.cellActions}>
                    <Show when={group.kind === 'subscription'}>
                        <button
                            type="button"
                            class={theme.table.action}
                            title={intl.getMessage('check_updates_btn')}
                            onClick={() => group.id && void refreshUpstreamRuleGroup(group.id)}
                        >
                            <Icon icon="refresh" color="gray" />
                        </button>
                    </Show>
                    <Show when={group.kind === 'custom'}>
                        <button
                            type="button"
                            class={theme.table.action}
                            title={intl.getMessage('upstream_rule_edit_rules')}
                            onClick={() =>
                                group.id && navigate(Paths.UpstreamRulesEdit.replace(':groupId', String(group.id)))
                            }
                        >
                            <Icon icon="bullets" color="gray" />
                        </button>
                    </Show>
                    <button
                        type="button"
                        class={theme.table.action}
                        title={intl.getMessage('edit_table_action')}
                        onClick={() => openEdit(group)}
                    >
                        <Icon icon="edit" color="gray" />
                    </button>
                    <button
                        type="button"
                        class={cn(theme.table.action, theme.table.action_danger)}
                        title={intl.getMessage('delete_table_action')}
                        onClick={() => setDeleteTarget(group)}
                    >
                        <Icon icon="delete" color="red" />
                    </button>
                </div>
            ),
        },
    ]);

    return (
        <div class={theme.layout.container}>
            <div class={cn(theme.layout.containerIn, theme.layout.containerIn_one_col)}>
                <h1 class={cn(theme.layout.title, theme.title.h4, theme.title.h3_tablet)}>
                    {intl.getMessage('upstream_rule_groups_title')}
                </h1>

                <Upstream />

                <div class={s.header}>
                    <div>
                        <h2 class={cn(theme.layout.subtitle, theme.title.h5, theme.title.h4_tablet)}>
                            {intl.getMessage('upstream_rule_groups')}
                        </h2>
                        <p class={cn(theme.text.t2, s.description)}>
                            {intl.getMessage('upstream_rule_groups_desc')}
                        </p>
                    </div>
                    <button
                        type="button"
                        class={s.refreshButton}
                        disabled={upstreamRulesState.processingAction}
                        onClick={() => void refreshAllUpstreamRuleGroups()}
                    >
                        <Icon icon="refresh" color="green" />
                        <span>{intl.getMessage('check_updates_btn')}</span>
                    </button>
                </div>

                <div class={s.actions}>
                    <PlusButton onClick={() => openAdd('subscription')}>
                        {intl.getMessage('upstream_rule_add_subscription')}
                    </PlusButton>
                    <PlusButton onClick={() => openAdd('custom')}>
                        {intl.getMessage('upstream_rule_add_custom')}
                    </PlusButton>
                </div>

                <Show when={!upstreamRulesState.processing} fallback={<PageLoader />}>
                    <Table
                        data={upstreamRulesState.groups}
                        columns={columns()}
                        getRowId={(group) => group.id ?? group.name}
                        defaultSort={{ key: 'priority', direction: 'asc' }}
                        emptyTable={<div class={s.empty}>{intl.getMessage('upstream_rule_empty')}</div>}
                    />
                </Show>
            </div>

            <Show when={dialogVisible()}>
                <UpstreamRuleDialog
                    visible
                    group={selectedGroup()}
                    initialKind={dialogKind()}
                    onClose={() => setDialogVisible(false)}
                />
            </Show>

            <Show when={deleteTarget()}>
                {(group) => (
                    <ConfirmDialog
                        title={intl.getMessage('upstream_rule_delete')}
                        text={group().name}
                        buttonText={intl.getMessage('delete_table_action')}
                        cancelText={intl.getMessage('cancel')}
                        buttonVariant="danger"
                        submitDisabled={upstreamRulesState.processingAction}
                        onClose={() => setDeleteTarget(undefined)}
                        onConfirm={() => {
                            const id = group().id;
                            if (!id) return;
                            void deleteUpstreamRuleGroup(id).then((deleted) => {
                                if (deleted) setDeleteTarget(undefined);
                            });
                        }}
                    />
                )}
            </Show>
        </div>
    );
};
