import React, { useEffect, useState } from 'react';
import { useDispatch } from 'react-redux';
import { useHistory, useParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';

import apiClient from '../../../api/Api';
import { addErrorToast, addSuccessToast } from '../../../actions/toasts';
import Card from '../../ui/Card';
import Loading from '../../ui/Loading';
import PageTitle from '../../ui/PageTitle';
import { normalizeUpstreams } from './helpers';
import RuleSyntaxHelp from './RuleSyntaxHelp';
import { UpstreamRuleGroup, UpstreamRuleGroupsStatus, UpstreamRuleTestResponse } from './types';

const Editor = () => {
    const { t } = useTranslation();
    const dispatch = useDispatch();
    const history = useHistory();
    const { groupId } = useParams<{ groupId: string }>();
    const [group, setGroup] = useState<UpstreamRuleGroup>();
    const [upstreams, setUpstreams] = useState('');
    const [loading, setLoading] = useState(true);
    const [saving, setSaving] = useState(false);
    const [domain, setDomain] = useState('');
    const [testResult, setTestResult] = useState<UpstreamRuleTestResponse>();

    useEffect(() => {
        apiClient
            .getUpstreamRuleGroups()
            .then((status: UpstreamRuleGroupsStatus) => {
                const found = (status.groups || []).find(
                    (item) => item.id === Number(groupId) && item.kind === 'custom',
                );
                if (!found) {
                    history.replace('/upstream_rules');
                    return;
                }
                setGroup(found);
                setUpstreams(found.upstreams.join('\n'));
            })
            .catch((error) => dispatch(addErrorToast({ error })))
            .finally(() => setLoading(false));
    }, [groupId]);

    const save = async (event: React.FormEvent) => {
        event.preventDefault();
        if (!group) {
            return;
        }
        setSaving(true);
        try {
            const saved = await apiClient.saveUpstreamRuleGroup({
                ...group,
                name: group.name.trim(),
                upstreams: normalizeUpstreams(upstreams),
                rules: group.rules || '',
            });
            setGroup(saved);
            setUpstreams(saved.upstreams.join('\n'));
            dispatch(addSuccessToast('upstream_rules_saved'));
        } catch (error) {
            dispatch(addErrorToast({ error }));
        } finally {
            setSaving(false);
        }
    };

    const testDomain = async (event: React.FormEvent) => {
        event.preventDefault();
        try {
            setTestResult(await apiClient.testUpstreamRuleDomain(domain.trim()));
        } catch (error) {
            dispatch(addErrorToast({ error }));
        }
    };

    if (loading || !group) {
        return <Loading />;
    }

    return (
        <>
            <PageTitle title={t('upstream_rules_custom_editor')} subtitle={t('upstream_rules_custom_editor_desc')} />
            <Card>
                <form onSubmit={save}>
                    <div className="row">
                        <div className="col-md-6 form__group">
                            <label className="form__label" htmlFor="custom-group-name">
                                {t('name_table_header')}
                            </label>
                            <input
                                id="custom-group-name"
                                className="form-control"
                                required
                                value={group.name}
                                onChange={(event) => setGroup({ ...group, name: event.target.value })}
                            />
                        </div>
                        <div className="col-md-3 form__group">
                            <label className="form__label" htmlFor="custom-group-priority">
                                {t('upstream_rules_priority')}
                            </label>
                            <input
                                id="custom-group-priority"
                                type="number"
                                className="form-control"
                                value={group.priority}
                                onChange={(event) => setGroup({ ...group, priority: Number(event.target.value) })}
                            />
                        </div>
                        <div className="col-md-3 form__group d-flex align-items-end">
                            <label className="checkbox mb-2">
                                <input
                                    type="checkbox"
                                    className="checkbox__input"
                                    checked={group.enabled}
                                    onChange={(event) => setGroup({ ...group, enabled: event.target.checked })}
                                />
                                <span className="checkbox__label">{t('enabled_table_header')}</span>
                            </label>
                        </div>
                    </div>
                    <div className="form__group">
                        <label className="form__label" htmlFor="custom-group-upstreams">
                            {t('upstream_rules_dns_servers')}
                        </label>
                        <textarea
                            id="custom-group-upstreams"
                            className="form-control font-monospace"
                            rows={4}
                            required
                            value={upstreams}
                            onChange={(event) => setUpstreams(event.target.value)}
                        />
                    </div>
                    <div className="form__group">
                        <label className="form__label" htmlFor="custom-group-rules">
                            {t('upstream_rules_rules')}
                        </label>
                        <textarea
                            id="custom-group-rules"
                            className="form-control font-monospace text-input"
                            rows={18}
                            value={group.rules || ''}
                            onChange={(event) => setGroup({ ...group, rules: event.target.value })}
                        />
                        <div className="form__description">{t('upstream_rules_rules_hint')}</div>
                    </div>
                    <div className="card-actions">
                        <button className="btn btn-success btn-standard btn-large mr-2" type="submit" disabled={saving}>
                            {t('apply_btn')}
                        </button>
                        <button className="btn btn-secondary" type="button" onClick={() => history.push('/upstream_rules')}>
                            {t('cancel_btn')}
                        </button>
                    </div>
                </form>
            </Card>
            <Card title={t('upstream_rules_test_domain')}>
                <form onSubmit={testDomain} className="d-flex align-items-start">
                    <input
                        className="form-control mr-2"
                        required
                        value={domain}
                        onChange={(event) => setDomain(event.target.value)}
                        placeholder="www.example.com"
                    />
                    <button className="btn btn-primary" type="submit">
                        {t('test_upstream_btn')}
                    </button>
                </form>
                {testResult && (
                    <div className="mt-3">
                        {testResult.matched
                            ? t('upstream_rules_test_matched', {
                                  name: testResult.group_name,
                                  upstreams: (testResult.upstreams || []).join(', '),
                              })
                            : t('upstream_rules_test_not_matched')}
                    </div>
                )}
            </Card>
            <RuleSyntaxHelp />
        </>
    );
};

export default Editor;
