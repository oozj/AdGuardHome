import React, { useCallback, useEffect, useState } from 'react';
import { useDispatch } from 'react-redux';
import { useTranslation } from 'react-i18next';

import apiClient from '../../../api/Api';
import { addErrorToast, addSuccessToast } from '../../../actions/toasts';
import Card from '../../ui/Card';
import PageTitle from '../../ui/PageTitle';
import { addPeerLink, isConfigSyncPanelVisible, normalizeConfigSyncStatus, removePeerLink } from './helpers';
import { ConfigSyncRole, ConfigSyncStatus } from './types';

const emptyStatus: ConfigSyncStatus = { role: '', link: '', peers: [] };

const ConfigSync = () => {
    const { t } = useTranslation();
    const dispatch = useDispatch();
    const [status, setStatus] = useState<ConfigSyncStatus>(emptyStatus);
    const [role, setRole] = useState<ConfigSyncRole>('');
    const [peerLink, setPeerLink] = useState('');
    const [loading, setLoading] = useState(true);
    const [processing, setProcessing] = useState(false);

    const load = useCallback(async () => {
        setLoading(true);
        try {
            const next = normalizeConfigSyncStatus(await apiClient.getConfigSync());
            setStatus(next);
            setRole(next.role);
        } catch (error) {
            dispatch(addErrorToast({ error }));
        } finally {
            setLoading(false);
        }
    }, []);

    useEffect(() => {
        load();
    }, []);

    const save = async (nextRole: ConfigSyncRole, links = status.peers.map((peer) => peer.link)) => {
        setProcessing(true);
        try {
            const next = normalizeConfigSyncStatus(await apiClient.saveConfigSync({ role: nextRole, links }));
            setStatus(next);
            setRole(next.role);
            dispatch(addSuccessToast('config_sync_saved'));
        } catch (error) {
            dispatch(addErrorToast({ error }));
        } finally {
            setProcessing(false);
        }
    };

    const addPeer = async () => {
        if (role !== status.role) {
            return;
        }
        const peers = addPeerLink(status.peers, peerLink);
        if (peers === status.peers) {
            return;
        }
        setPeerLink('');
        await save('primary', peers.map((peer) => peer.link));
    };

    const removePeer = async (link: string) => {
        const peers = removePeerLink(status.peers, link);
        await save('primary', peers.map((peer) => peer.link));
    };

    const syncNow = async () => {
        setProcessing(true);
        try {
            await apiClient.syncConfigNow();
            await load();
            dispatch(addSuccessToast('config_sync_completed'));
        } catch (error) {
            dispatch(addErrorToast({ error }));
        } finally {
            setProcessing(false);
        }
    };

    const regenerate = async () => {
        setProcessing(true);
        try {
            const next = normalizeConfigSyncStatus(await apiClient.regenerateConfigSyncToken());
            setStatus(next);
            dispatch(addSuccessToast('config_sync_link_regenerated'));
        } catch (error) {
            dispatch(addErrorToast({ error }));
        } finally {
            setProcessing(false);
        }
    };

    const copyLink = async () => {
        try {
            if (navigator.clipboard && window.isSecureContext) {
                await navigator.clipboard.writeText(status.link);
            } else {
                const copyArea = document.createElement('textarea');
                copyArea.value = status.link;
                copyArea.style.position = 'fixed';
                copyArea.style.opacity = '0';
                document.body.appendChild(copyArea);
                copyArea.select();
                const copied = document.execCommand('copy');
                copyArea.remove();
                if (!copied) {
                    throw new Error(t('config_sync_copy_failed'));
                }
            }
            dispatch(addSuccessToast('config_sync_link_copied'));
        } catch (error) {
            dispatch(addErrorToast({ error }));
        }
    };

    return (
        <>
            <PageTitle title={t('config_sync_title')} subtitle={t('config_sync_desc')} />
            <div className="content">
                <div className="row">
                    <div className="col-md-12">
                        <Card title={t('config_sync_role_title')} bodyType="card-body box-body--settings">
                            <div className="form-group">
                                <label className="custom-control custom-radio d-block mb-3">
                                    <input
                                        type="radio"
                                        className="custom-control-input"
                                        name="config-sync-role"
                                        checked={role === 'primary'}
                                        disabled={loading || processing}
                                        onChange={() => setRole('primary')}
                                    />
                                    <span className="custom-control-label">
                                        <strong>{t('config_sync_primary')}</strong>
                                        <span className="d-block text-muted">{t('config_sync_primary_desc')}</span>
                                    </span>
                                </label>
                                <label className="custom-control custom-radio d-block">
                                    <input
                                        type="radio"
                                        className="custom-control-input"
                                        name="config-sync-role"
                                        checked={role === 'secondary'}
                                        disabled={loading || processing}
                                        onChange={() => setRole('secondary')}
                                    />
                                    <span className="custom-control-label">
                                        <strong>{t('config_sync_secondary')}</strong>
                                        <span className="d-block text-muted">{t('config_sync_secondary_desc')}</span>
                                    </span>
                                </label>
                                <label className="custom-control custom-radio d-block mt-3">
                                    <input
                                        type="radio"
                                        className="custom-control-input"
                                        name="config-sync-role"
                                        checked={role === ''}
                                        disabled={loading || processing}
                                        onChange={() => setRole('')}
                                    />
                                    <span className="custom-control-label">
                                        <strong>{t('config_sync_disabled')}</strong>
                                        <span className="d-block text-muted">{t('config_sync_disabled_desc')}</span>
                                    </span>
                                </label>
                            </div>
                            <button
                                type="button"
                                className="btn btn-success btn-standard"
                                disabled={role === status.role || processing}
                                onClick={() => save(role)}>
                                {t('save_btn')}
                            </button>
                        </Card>
                    </div>
                </div>

                {isConfigSyncPanelVisible(role, 'primary') && (
                    <div className="row">
                        <div className="col-md-12">
                            <Card title={t('config_sync_secondary_servers')} bodyType="card-body box-body--settings">
                                <div className="form-group">
                                    <label className="form-label" htmlFor="config-sync-peer-link">
                                        {t('config_sync_pairing_link')}
                                    </label>
                                    <div className="input-group">
                                        <input
                                            id="config-sync-peer-link"
                                            className="form-control"
                                            value={peerLink}
                                            disabled={processing}
                                            placeholder={t('config_sync_pairing_link_placeholder')}
                                            onChange={(event) => setPeerLink(event.target.value)}
                                        />
                                        <span className="input-group-append">
                                            <button
                                                type="button"
                                                className="btn btn-success"
                                                disabled={!peerLink.trim() || role !== status.role || processing}
                                                onClick={addPeer}>
                                                {t('config_sync_add_secondary')}
                                            </button>
                                        </span>
                                    </div>
                                    {role !== status.role && (
                                        <small className="form-text text-muted">{t('config_sync_save_role_first')}</small>
                                    )}
                                </div>

                                {status.peers.length === 0 ? (
                                    <p className="text-muted">{t('config_sync_no_secondaries')}</p>
                                ) : (
                                    <div className="table-responsive mb-3">
                                        <table className="table table-vcenter">
                                            <thead>
                                                <tr>
                                                    <th>{t('config_sync_secondary_server')}</th>
                                                    <th>{t('config_sync_last_result')}</th>
                                                    <th className="text-right">{t('actions_table_header')}</th>
                                                </tr>
                                            </thead>
                                            <tbody>
                                                {status.peers.map((peer) => (
                                                    <tr key={peer.link}>
                                                        <td className="text-break">{peer.link.split('#')[0]}</td>
                                                        <td className={peer.error ? 'text-danger' : 'text-muted'}>
                                                            {peer.error || peer.last_sync || t('config_sync_not_yet_synced')}
                                                        </td>
                                                        <td className="text-right">
                                                            <button
                                                                type="button"
                                                                className="btn btn-outline-secondary btn-sm"
                                                                disabled={processing}
                                                                onClick={() => removePeer(peer.link)}>
                                                                {t('delete_table_action')}
                                                            </button>
                                                        </td>
                                                    </tr>
                                                ))}
                                            </tbody>
                                        </table>
                                    </div>
                                )}

                                <button
                                    type="button"
                                    className="btn btn-primary btn-standard"
                                    disabled={status.peers.length === 0 || processing}
                                    onClick={syncNow}>
                                    {t('config_sync_now')}
                                </button>
                            </Card>
                        </div>
                    </div>
                )}

                {isConfigSyncPanelVisible(role, 'secondary') && (
                    <div className="row">
                        <div className="col-md-12">
                            <Card title={t('config_sync_pairing_link')} bodyType="card-body box-body--settings">
                                <p className="text-muted">{t('config_sync_pairing_link_desc')}</p>
                                <textarea
                                    className="form-control mb-3"
                                    rows={3}
                                    readOnly
                                    value={status.link}
                                    placeholder={t('config_sync_save_to_generate')}
                                />
                                <button
                                    type="button"
                                    className="btn btn-primary btn-standard mr-2"
                                    disabled={!status.link || processing}
                                    onClick={copyLink}>
                                    {t('config_sync_copy_link')}
                                </button>
                                <button
                                    type="button"
                                    className="btn btn-outline-secondary btn-standard"
                                    disabled={processing}
                                    onClick={regenerate}>
                                    {t('config_sync_regenerate_link')}
                                </button>
                            </Card>
                        </div>
                    </div>
                )}
            </div>
        </>
    );
};

export default ConfigSync;
