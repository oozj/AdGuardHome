import { ConfigSyncPeer, ConfigSyncRole, ConfigSyncStatus } from './types';

export const normalizeConfigSyncStatus = (status: Partial<ConfigSyncStatus>): ConfigSyncStatus => ({
    role: status.role || '',
    link: status.link || '',
    peers: status.peers || [],
});

export const addPeerLink = (peers: ConfigSyncPeer[], value: string): ConfigSyncPeer[] => {
    const link = value.trim();
    if (!link || peers.some((peer) => peer.link === link)) {
        return peers;
    }

    const endpoint = link.split('#')[0];
    const existingIndex = peers.findIndex((peer) => peer.link.split('#')[0] === endpoint);
    if (existingIndex >= 0) {
        return peers.map((peer, index) => (index === existingIndex ? { link } : peer));
    }

    return [...peers, { link }];
};

export const removePeerLink = (peers: ConfigSyncPeer[], link: string): ConfigSyncPeer[] =>
    peers.filter((peer) => peer.link !== link);

export const isConfigSyncPanelVisible = (selectedRole: ConfigSyncRole, panelRole: ConfigSyncRole): boolean =>
    selectedRole === panelRole;
