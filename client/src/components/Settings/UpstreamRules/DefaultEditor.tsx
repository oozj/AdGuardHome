import React, { useEffect, useState } from 'react';
import { useDispatch } from 'react-redux';
import { useHistory } from 'react-router-dom';
import { useTranslation } from 'react-i18next';

import apiClient from '../../../api/Api';
import { addErrorToast, addSuccessToast } from '../../../actions/toasts';
import Card from '../../ui/Card';
import Loading from '../../ui/Loading';
import PageTitle from '../../ui/PageTitle';
import { normalizeUpstreams } from './helpers';
import { UpstreamRuleGroupsStatus } from './types';

const DefaultEditor = () => {
    const { t } = useTranslation();
    const dispatch = useDispatch();
    const history = useHistory();
    const [upstreams, setUpstreams] = useState('');
    const [loading, setLoading] = useState(true);
    const [saving, setSaving] = useState(false);

    useEffect(() => {
        apiClient
            .getUpstreamRuleGroups()
            .then((status: UpstreamRuleGroupsStatus) => setUpstreams((status.default_upstreams || []).join('\n')))
            .catch((error) => dispatch(addErrorToast({ error })))
            .finally(() => setLoading(false));
    }, []);

    const save = async (event: React.FormEvent) => {
        event.preventDefault();
        setSaving(true);
        try {
            await apiClient.setDnsConfig({
                upstream_dns: normalizeUpstreams(upstreams),
                upstream_dns_file: '',
            });
            dispatch(addSuccessToast('upstream_rules_default_saved'));
            history.push('/upstream_rules');
        } catch (error) {
            dispatch(addErrorToast({ error }));
        } finally {
            setSaving(false);
        }
    };

    if (loading) {
        return <Loading />;
    }

    return (
        <>
            <PageTitle title={t('upstream_rules_default_group')} subtitle={t('upstream_rules_default_desc')} />
            <Card>
                <form onSubmit={save}>
                    <div className="form__group">
                        <label className="form__label" htmlFor="default-group-upstreams">
                            {t('upstream_rules_dns_servers')}
                        </label>
                        <textarea
                            id="default-group-upstreams"
                            className="form-control font-monospace text-input"
                            rows={18}
                            required
                            value={upstreams}
                            onChange={(event) => setUpstreams(event.target.value)}
                        />
                        <div className="form__description">{t('upstream_rules_default_hint')}</div>
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
        </>
    );
};

export default DefaultEditor;

