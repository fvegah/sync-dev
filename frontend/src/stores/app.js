import { writable, derived } from 'svelte/store';

// Current tab - navigation state
export const currentTab = writable('peers');

// Peers list from backend
export const peers = writable([]);

// Folder pairs configuration
export const folderPairs = writable([]);

// Sync status from backend
export const syncStatus = writable({
    status: 'idle',
    action: ''
});

// Raw progress data from backend (AggregateProgress shape)
export const progressData = writable(null);

// Backward compatibility alias
export const transferProgress = progressData;

// Derived: formatted speed string
export const formattedSpeed = derived(progressData, ($p) => {
    if (!$p || !$p.bytesPerSecond || $p.bytesPerSecond <= 0) return '';
    const speed = $p.bytesPerSecond;
    if (speed >= 1024 * 1024) {
        return `${(speed / (1024 * 1024)).toFixed(1)} MB/s`;
    } else if (speed >= 1024) {
        return `${(speed / 1024).toFixed(1)} KB/s`;
    }
    return `${Math.round(speed)} B/s`;
});

// Derived: formatted ETA string
export const formattedETA = derived(progressData, ($p) => {
    if (!$p || !$p.eta || $p.eta < 0) return '';
    const eta = $p.eta;
    if (eta < 60) {
        return `${eta}s remaining`;
    } else if (eta < 3600) {
        const mins = Math.floor(eta / 60);
        const secs = eta % 60;
        return `${mins}:${secs.toString().padStart(2, '0')} remaining`;
    } else {
        const hours = Math.floor(eta / 3600);
        const mins = Math.floor((eta % 3600) / 60);
        return `${hours}h ${mins}m remaining`;
    }
});

// Derived: file count string
export const fileCountProgress = derived(progressData, ($p) => {
    if (!$p || !$p.totalFiles) return '';
    return `${$p.completedFiles} of ${$p.totalFiles} files`;
});

// Derived: is syncing
export const isSyncing = derived(progressData, ($p) => {
    return $p?.status === 'syncing';
});

// Derived: overall percentage
export const overallPercentage = derived(progressData, ($p) => {
    return $p?.percentage ?? 0;
});

// Derived: active files list
export const activeFiles = derived(progressData, ($p) => {
    return $p?.activeFiles ?? [];
});

// Recent sync events
export const recentEvents = writable([]);

// App config
export const config = writable({
    deviceId: '',
    deviceName: '',
    port: 52525,
    syncIntervalMins: 5,
    globalExclusions: [],
    autoSync: true,
    showNotifications: true
});

// Pairing state
export const pairingState = writable({
    code: '',
    isPairing: false,
    targetPeer: null
});

// Modal state
export const modalState = writable({
    show: false,
    type: null,
    data: null
});

// Helper to show modal
export function showModal(type, data = null) {
    modalState.set({ show: true, type, data });
}

// Helper to close modal
export function closeModal() {
    modalState.set({ show: false, type: null, data: null });
}
