import { describe, expect, test } from 'vitest';

import {
    addPeerLink,
    isConfigSyncPanelVisible,
    normalizeConfigSyncStatus,
    removePeerLink,
} from '../components/Settings/ConfigSync/helpers';

describe('configuration synchronization helpers', () => {
    test('normalizes missing peer data', () => {
        expect(normalizeConfigSyncStatus({ role: 'primary' } as any)).toEqual({
            role: 'primary',
            link: '',
            peers: [],
        });
    });

    test('adds trimmed unique pairing links', () => {
        const existing = [{ link: 'http://192.0.2.2:3000/control/config_sync/receive#token=one' }];

        expect(
            addPeerLink(existing, '  http://192.0.2.3:3000/control/config_sync/receive#token=two  '),
        ).toEqual([
            existing[0],
            { link: 'http://192.0.2.3:3000/control/config_sync/receive#token=two' },
        ]);
        expect(addPeerLink(existing, existing[0].link)).toEqual(existing);
    });

    test('replaces an old token for the same secondary endpoint', () => {
        const oldLink = 'http://192.0.2.2:3000/control/config_sync/receive#token=old';
        const newLink = 'http://192.0.2.2:3000/control/config_sync/receive#token=new';

        expect(addPeerLink([{ link: oldLink }], newLink)).toEqual([{ link: newLink }]);
    });

    test('removes the selected pairing link', () => {
        const peers = [{ link: 'one' }, { link: 'two' }];

        expect(removePeerLink(peers, 'one')).toEqual([{ link: 'two' }]);
    });

    test('shows a role panel immediately after selecting that role', () => {
        expect(isConfigSyncPanelVisible('primary', 'primary')).toBe(true);
        expect(isConfigSyncPanelVisible('primary', 'secondary')).toBe(false);
        expect(isConfigSyncPanelVisible('secondary', 'secondary')).toBe(true);
    });
});
