import React, { useCallback, useEffect, useMemo, useState } from 'react';
// @ts-expect-error FIXME: update react-table
import ReactTable from 'react-table';
import { useDispatch } from 'react-redux';
import { useHistory } from 'react-router-dom';
import { useTranslation } from 'react-i18next';

import apiClient from '../../../api/Api';
import { addErrorToast, addSuccessToast } from '../../../actions/toasts';
import Card from '../../ui/Card';
import PageTitle from '../../ui/PageTitle';
import GroupModal from './GroupModal';
import {
    buildUpstreamRuleGroupRows,
    formatRulesCount,
    getUpstreamRuleGroupErrors,
    summarizeUpstreams,
} from './helpers';
import { UpstreamRuleGroup, UpstreamRuleGroupKind, UpstreamRuleGroupsStatus } from './types';

const UpstreamRules = () => {
    const { t } = useTranslation();
    const dispatch = useDispatch();
    const history = useHistory();
    const [status, setStatus] = useState<UpstreamRuleGroupsStatus>({
        default_upstreams: [],
        groups: [],
        issues: [],
    });
    const [loading, setLoading] = useState(true);
    const [processing, setProcessing] = useState(false);
    const [modalKind, setModalKind] = useState<UpstreamRuleGroupKind>();
    const [editing, setEditing] = useState<UpstreamRuleGroup>();

    const load = useCallback(async () => {
        setLoading(true);
        try {
            const next: UpstreamRuleGroupsStatus = await apiClient.getUpstreamRuleGroups();
            setStatus({
                ...next,
                groups: next.groups || [],
                issues: next.issues || [],
            });
        } catch (error) {
            dispatch(addErrorToast({ error }));
        } finally {
            setLoading(false);
        }
    }, []);

    useEffect(() => {
        load();
    }, []);

    const closeModal = () => {
        setModalKind(undefined);
        setEditing(undefined);
    };

    const save = async (group: UpstreamRuleGroup) => {
        setProcessing(true);
        try {
            const saved: UpstreamRuleGroup = await apiClient.saveUpstreamRuleGroup(group);
            closeModal();
            await load();
            dispatch(addSuccessToast('upstream_rules_saved'));
            if (saved.kind === 'custom' && group.id === 0) {
                history.push(`/upstream_rules/${saved.id}`);
            }
        } catch (error) {
            dispatch(addErrorToast({ error }));
        } finally {
            setProcessing(false);
        }
    };

    const remove = async (group: UpstreamRuleGroup) => {
        if (!window.confirm(t('upstream_rules_confirm_delete'))) {
            return;
        }
        setProcessing(true);
        try {
            await apiClient.deleteUpstreamRuleGroup(group.id);
            await load();
            dispatch(addSuccessToast('upstream_rules_deleted'));
        } catch (error) {
            dispatch(addErrorToast({ error }));
        } finally {
            setProcessing(false);
        }
    };

    const toggle = async (group: UpstreamRuleGroup) => {
        if (group.kind === 'default') {
            return;
        }
        await save({ ...group, enabled: !group.enabled });
    };

    const refresh = async () => {
        setProcessing(true);
        try {
            const subscriptions = status.groups.filter((group) => group.kind === 'subscription');
            await Promise.all(subscriptions.map((group) => apiClient.refreshUpstreamRuleGroup(group.id)));
            await load();
            dispatch(addSuccessToast('upstream_rules_refreshed'));
        } catch (error) {
            dispatch(addErrorToast({ error }));
        } finally {
            setProcessing(false);
        }
    };

    const edit = (group: UpstreamRuleGroup) => {
        if (group.kind === 'default') {
            history.push('/upstream_rules/default');
            return;
        }
        if (group.kind === 'custom') {
            history.push(`/upstream_rules/${group.id}`);
            return;
        }
        setEditing(group);
        setModalKind(group.kind);
    };

    const ruleKindLabel = (kind: UpstreamRuleGroupKind) => {
        if (kind === 'subscription') {
            return t('upstream_rules_subscription');
        }
        if (kind === 'default') {
            return t('upstream_rules_default');
        }

        return t('upstream_rules_custom');
    };

    const columns = useMemo(
        () => [
            {
                Header: t('enabled_table_header'),
                accessor: 'enabled',
                width: 90,
                className: 'text-center',
                sortable: false,
                Cell: ({ original }: { original: UpstreamRuleGroup }) => (
                    <label className="checkbox">
                        <input
                            type="checkbox"
                            className="checkbox__input"
                            checked={original.enabled}
                            disabled={processing || original.kind === 'default'}
                            onChange={() => toggle(original)}
                        />
                        <span className="checkbox__label" />
                    </label>
                ),
            },
            {
                Header: t('name_table_header'),
                accessor: 'name',
                minWidth: 160,
                Cell: ({ original }: { original: UpstreamRuleGroup }) => {
                    const errors = getUpstreamRuleGroupErrors(original, status.issues);

                    return (
                        <div>
                            <div>{original.name}</div>
                            {errors.length > 0 && (
                                <div className="small text-danger text-truncate" title={errors.join('\n')}>
                                    {t('upstream_rules_issue_count', { count: errors.length })}
                                </div>
                            )}
                        </div>
                    );
                },
            },
            {
                Header: t('upstream_rules_type'),
                accessor: 'kind',
                width: 110,
                Cell: ({ value }: { value: UpstreamRuleGroupKind }) => ruleKindLabel(value),
            },
            {
                Header: t('upstream_rules_dns_servers'),
                accessor: 'upstreams',
                minWidth: 200,
                Cell: ({ value }: { value: string[] }) => (
                    <div className="text-truncate" title={summarizeUpstreams(value)}>
                        {summarizeUpstreams(value)}
                    </div>
                ),
            },
            {
                Header: t('rules_count_table_header'),
                accessor: 'rules_count',
                width: 90,
                className: 'text-center',
                Cell: ({ original, value }: { original: UpstreamRuleGroup; value: number }) =>
                    formatRulesCount(original.kind, value),
            },
            {
                Header: t('upstream_rules_priority'),
                accessor: 'priority',
                width: 80,
                className: 'text-center',
                Cell: ({ original, value }: { original: UpstreamRuleGroup; value: number }) =>
                    original.kind === 'default' ? t('upstream_rules_last') : value,
            },
            {
                Header: t('actions_table_header'),
                width: 100,
                className: 'text-center',
                sortable: false,
                Cell: ({ original }: { original: UpstreamRuleGroup }) => (
                    <div className="logs__row logs__row--center">
                        <button
                            type="button"
                            className="btn btn-icon btn-outline-primary btn-sm mr-2"
                            title={t('edit_table_action')}
                            onClick={() => edit(original)}>
                            <svg className="icons icon12">
                                <use xlinkHref="#edit" />
                            </svg>
                        </button>
                        {original.kind !== 'default' && (
                            <button
                                type="button"
                                className="btn btn-icon btn-outline-secondary btn-sm"
                                title={t('delete_table_action')}
                                onClick={() => remove(original)}>
                                <svg className="icons icon12">
                                    <use xlinkHref="#delete" />
                                </svg>
                            </button>
                        )}
                    </div>
                ),
            },
        ],
        [processing, status.groups],
    );

    return (
        <>
            <PageTitle title={t('upstream_rules_title')} subtitle={t('upstream_rules_desc')} />
            <div className="content">
                <div className="row">
                    <div className="col-md-12">
                        <Card subtitle={t('upstream_rules_list_hint')}>
                            <ReactTable
                                data={buildUpstreamRuleGroupRows(
                                    status.groups,
                                    status.default_upstreams,
                                    t('upstream_rules_default_group'),
                                )}
                                columns={columns}
                                showPagination
                                defaultPageSize={10}
                                loading={loading || processing}
                                minRows={6}
                                ofText="/"
                                previousText={t('previous_btn')}
                                nextText={t('next_btn')}
                                pageText={t('page_table_footer_text')}
                                rowsText={t('rows_table_footer_text')}
                                loadingText={t('loading_table_status')}
                                noDataText={t('upstream_rules_empty')}
                            />
                            <div className="card-actions">
                                <button
                                    className="btn btn-success btn-standard mr-2 btn-large mb-2"
                                    type="button"
                                    onClick={() => setModalKind('subscription')}>
                                    {t('upstream_rules_add_subscription')}
                                </button>
                                <button
                                    className="btn btn-success btn-standard mr-2 btn-large mb-2"
                                    type="button"
                                    onClick={() => setModalKind('custom')}>
                                    {t('upstream_rules_add_custom')}
                                </button>
                                <button
                                    className="btn btn-primary btn-standard mb-2"
                                    type="button"
                                    disabled={processing}
                                    onClick={refresh}>
                                    {t('check_updates_btn')}
                                </button>
                            </div>
                        </Card>
                    </div>
                </div>
            </div>
            {modalKind && (
                <GroupModal
                    isOpen
                    kind={modalKind}
                    group={editing}
                    processing={processing}
                    onClose={closeModal}
                    onSave={save}
                />
            )}
        </>
    );
};

export default UpstreamRules;
