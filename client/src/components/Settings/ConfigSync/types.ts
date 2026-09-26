export type ConfigSyncRole = '' | 'primary' | 'secondary';

export interface ConfigSyncPeer {
    link: string;
    last_sync?: string;
    error?: string;
}

export interface ConfigSyncStatus {
    role: ConfigSyncRole;
    link: string;
    peers: ConfigSyncPeer[];
}
