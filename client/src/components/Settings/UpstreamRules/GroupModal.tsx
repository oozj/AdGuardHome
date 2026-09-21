import React, { useEffect, useState } from 'react';
import ReactModal from 'react-modal';
import { useTranslation } from 'react-i18next';

import { normalizeUpstreams } from './helpers';
import { UpstreamRuleGroup, UpstreamRuleGroupKind } from './types';
import '../../ui/Modal.css';

ReactModal.setAppElement('#root');

type Props = {
    isOpen: boolean;
    kind: UpstreamRuleGroupKind;
    group?: UpstreamRuleGroup;
    processing: boolean;
    onClose: () => void;
    onSave: (group: UpstreamRuleGroup) => void;
};

const emptyGroup = (kind: UpstreamRuleGroupKind): UpstreamRuleGroup => ({
    id: 0,
    name: '',
    enabled: true,
    priority: 100,
    kind,
    url: '',
    upstreams: [],
    rules: '',
});

const GroupModal = ({ isOpen, kind, group, processing, onClose, onSave }: Props) => {
    const { t } = useTranslation();
    const [draft, setDraft] = useState<UpstreamRuleGroup>(emptyGroup(kind));
    const [upstreams, setUpstreams] = useState('');

    useEffect(() => {
        const next = group || emptyGroup(kind);
        setDraft({ ...next, kind });
        setUpstreams((next.upstreams || []).join('\n'));
    }, [group, isOpen, kind]);

    const submit = (event: React.FormEvent) => {
        event.preventDefault();
        onSave({
            ...draft,
            name: draft.name.trim(),
            url: kind === 'subscription' ? (draft.url || '').trim() : '',
            upstreams: normalizeUpstreams(upstreams),
        });
    };

    return (
        <ReactModal
            className="Modal__Bootstrap modal-dialog modal-dialog-centered"
            closeTimeoutMS={0}
            isOpen={isOpen}
            onRequestClose={onClose}>
            <div className="modal-content">
                <div className="modal-header">
                    <h4 className="modal-title">
                        {group ? t('upstream_rules_edit_group') : t('upstream_rules_new_group')}
                    </h4>
                    <button type="button" className="close" onClick={onClose}>
                        <span className="sr-only">Close</span>
                    </button>
                </div>
                <form onSubmit={submit}>
                    <div className="modal-body">
                        <div className="form__group">
                            <label className="form__label" htmlFor="upstream-rule-name">
                                {t('name_table_header')}
                            </label>
                            <input
                                id="upstream-rule-name"
                                className="form-control"
                                required
                                value={draft.name}
                                onChange={(event) => setDraft({ ...draft, name: event.target.value })}
                            />
                        </div>
                        {kind === 'subscription' && (
                            <div className="form__group">
                                <label className="form__label" htmlFor="upstream-rule-url">
                                    {t('upstream_rules_subscription_url')}
                                </label>
                                <input
                                    id="upstream-rule-url"
                                    type="url"
                                    className="form-control"
                                    required
                                    value={draft.url || ''}
                                    onChange={(event) => setDraft({ ...draft, url: event.target.value })}
                                />
                            </div>
                        )}
                        <div className="form__group">
                            <label className="form__label" htmlFor="upstream-rule-servers">
                                {t('upstream_rules_dns_servers')}
                            </label>
                            <textarea
                                id="upstream-rule-servers"
                                className="form-control font-monospace"
                                rows={5}
                                required
                                value={upstreams}
                                onChange={(event) => setUpstreams(event.target.value)}
                            />
                            <div className="form__description">{t('upstream_rules_dns_servers_hint')}</div>
                        </div>
                        <div className="form__group">
                            <label className="form__label" htmlFor="upstream-rule-priority">
                                {t('upstream_rules_priority')}
                            </label>
                            <input
                                id="upstream-rule-priority"
                                type="number"
                                className="form-control"
                                value={draft.priority}
                                onChange={(event) => setDraft({ ...draft, priority: Number(event.target.value) })}
                            />
                        </div>
                        <label className="checkbox">
                            <input
                                type="checkbox"
                                className="checkbox__input"
                                checked={draft.enabled}
                                onChange={(event) => setDraft({ ...draft, enabled: event.target.checked })}
                            />
                            <span className="checkbox__label">{t('enabled_table_header')}</span>
                        </label>
                    </div>
                    <div className="modal-footer">
                        <button type="button" className="btn btn-secondary" onClick={onClose}>
                            {t('cancel_btn')}
                        </button>
                        <button type="submit" className="btn btn-success" disabled={processing}>
                            {t('save_btn')}
                        </button>
                    </div>
                </form>
            </div>
        </ReactModal>
    );
};

export default GroupModal;

