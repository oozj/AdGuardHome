import React from 'react';
import { useTranslation } from 'react-i18next';

import Card from '../../ui/Card';
import { UPSTREAM_RULE_SYNTAX_ROWS } from './ruleSyntax';

const RuleSyntaxHelp = () => {
    const { t } = useTranslation();

    return (
        <Card title={t('upstream_rules_help_title')}>
            <p>{t('upstream_rules_help_intro')}</p>
            <div className="table-responsive">
                <table className="table table-vcenter mb-0">
                    <thead>
                        <tr>
                            <th>{t('upstream_rules_help_syntax')}</th>
                            <th>{t('upstream_rules_help_example')}</th>
                            <th>{t('upstream_rules_help_result')}</th>
                        </tr>
                    </thead>
                    <tbody>
                        {UPSTREAM_RULE_SYNTAX_ROWS.map((row) => (
                            <tr key={row.syntax}>
                                <td>
                                    <code>{row.syntax}</code>
                                </td>
                                <td>
                                    <code>{t(row.exampleKey)}</code>
                                </td>
                                <td>{t(row.descriptionKey)}</td>
                            </tr>
                        ))}
                    </tbody>
                </table>
            </div>
            <div className="alert alert-info mt-4 mb-0" role="note">
                {t('upstream_rules_help_order')}
            </div>
        </Card>
    );
};

export default RuleSyntaxHelp;
