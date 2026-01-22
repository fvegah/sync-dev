import { writable } from 'svelte/store';

// Current tab
export const currentTab = writable('peers');

// Peers list
export const peers = writable([]);

// Folder pairs
export const folderPairs = writable([]);

// Sync status
export const syncStatus = writable({
    status: 'idle',
    action: ''
});

// Transfer progress
export const transferProgress = writable(null);

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
