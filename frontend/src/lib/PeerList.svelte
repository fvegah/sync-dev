<script>
    import { onMount, onDestroy } from 'svelte';
    import { peers, pairingState, showModal } from '../stores/app.js';
    import { GetPeers, GeneratePairingCode, RequestPairing, UnpairPeer } from '../../wailsjs/go/main/App.js';
    import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime.js';

    let myPairingCode = '';
    let pairingInput = '';
    let selectedPeer = null;

    onMount(async () => {
        await loadPeers();
        EventsOn('peers:changed', loadPeers);
    });

    onDestroy(() => {
        EventsOff('peers:changed');
    });

    async function loadPeers() {
        const result = await GetPeers();
        peers.set(result || []);
    }

    async function generateCode() {
        myPairingCode = await GeneratePairingCode();
    }

    async function startPairing(peer) {
        selectedPeer = peer;
        showModal('pairing', peer);
    }

    async function submitPairing(peer, code) {
        try {
            await RequestPairing(peer.id, code);
            pairingState.set({ code: '', isPairing: true, targetPeer: peer });
        } catch (err) {
            alert('Pairing failed: ' + err);
        }
    }

    async function unpair(peer) {
        if (confirm(`Unpair from ${peer.name}? This will remove all folder pairs with this device.`)) {
            await UnpairPeer(peer.id);
            await loadPeers();
        }
    }

    function getStatusColor(status) {
        switch (status) {
            case 'online': return '#4ade80';
            case 'syncing': return '#60a5fa';
            case 'pairing': return '#fbbf24';
            default: return '#6b7280';
        }
    }

    function getStatusText(status) {
        switch (status) {
            case 'online': return 'Online';
            case 'syncing': return 'Syncing';
            case 'pairing': return 'Pairing';
            default: return 'Offline';
        }
    }
</script>

<div class="peer-list">
    <div class="header">
        <h2>Devices</h2>
        <button class="btn-secondary" on:click={generateCode}>
            Generate Pairing Code
        </button>
    </div>

    {#if myPairingCode}
        <div class="pairing-code-display">
            <p>Share this code with another device to pair:</p>
            <div class="code">{myPairingCode}</div>
            <button class="btn-text" on:click={() => myPairingCode = ''}>Dismiss</button>
        </div>
    {/if}

    <div class="peers">
        {#if $peers.length === 0}
            <div class="empty-state">
                <p>No devices found on your network.</p>
                <p class="hint">Make sure SyncDev is running on other Macs in your local network.</p>
            </div>
        {:else}
            {#each $peers as peer}
                <div class="peer-card" class:paired={peer.paired}>
                    <div class="peer-info">
                        <div class="peer-icon">
                            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                <rect x="2" y="3" width="20" height="14" rx="2"/>
                                <path d="M8 21h8M12 17v4"/>
                            </svg>
                        </div>
                        <div class="peer-details">
                            <span class="peer-name">{peer.name}</span>
                            <span class="peer-host">{peer.host}:{peer.port}</span>
                        </div>
                    </div>
                    <div class="peer-status">
                        <span class="status-indicator" style="background-color: {getStatusColor(peer.status)}"></span>
                        <span class="status-text">{getStatusText(peer.status)}</span>
                    </div>
                    <div class="peer-actions">
                        {#if peer.paired}
                            <span class="paired-badge">Paired</span>
                            <button class="btn-icon danger" on:click={() => unpair(peer)} title="Unpair">
                                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                    <path d="M18 6L6 18M6 6l12 12"/>
                                </svg>
                            </button>
                        {:else if peer.status === 'online'}
                            <button class="btn-primary small" on:click={() => startPairing(peer)}>
                                Pair
                            </button>
                        {/if}
                    </div>
                </div>
            {/each}
        {/if}
    </div>
</div>

<style>
    .peer-list {
        padding: 20px;
        height: 100%;
        display: flex;
        flex-direction: column;
    }

    .header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 20px;
    }

    h2 {
        margin: 0;
        font-size: 1.5rem;
        font-weight: 600;
    }

    .pairing-code-display {
        background: rgba(59, 130, 246, 0.1);
        border: 1px solid rgba(59, 130, 246, 0.3);
        border-radius: 8px;
        padding: 16px;
        margin-bottom: 20px;
        text-align: center;
    }

    .pairing-code-display p {
        margin: 0 0 12px 0;
        color: #94a3b8;
    }

    .code {
        font-size: 2rem;
        font-family: monospace;
        font-weight: bold;
        letter-spacing: 8px;
        color: #60a5fa;
        padding: 12px;
        background: rgba(0, 0, 0, 0.2);
        border-radius: 4px;
    }

    .peers {
        flex: 1;
        overflow-y: auto;
    }

    .empty-state {
        text-align: center;
        padding: 40px;
        color: #64748b;
    }

    .empty-state .hint {
        font-size: 0.875rem;
        margin-top: 8px;
    }

    .peer-card {
        display: flex;
        align-items: center;
        gap: 16px;
        padding: 16px;
        background: rgba(255, 255, 255, 0.05);
        border-radius: 8px;
        margin-bottom: 12px;
        border: 1px solid transparent;
        transition: all 0.2s;
    }

    .peer-card:hover {
        background: rgba(255, 255, 255, 0.08);
    }

    .peer-card.paired {
        border-color: rgba(74, 222, 128, 0.3);
    }

    .peer-info {
        display: flex;
        align-items: center;
        gap: 12px;
        flex: 1;
    }

    .peer-icon {
        width: 40px;
        height: 40px;
        display: flex;
        align-items: center;
        justify-content: center;
        background: rgba(255, 255, 255, 0.1);
        border-radius: 8px;
    }

    .peer-icon svg {
        width: 24px;
        height: 24px;
        color: #94a3b8;
    }

    .peer-details {
        display: flex;
        flex-direction: column;
    }

    .peer-name {
        font-weight: 500;
        color: #f1f5f9;
    }

    .peer-host {
        font-size: 0.75rem;
        color: #64748b;
        font-family: monospace;
    }

    .peer-status {
        display: flex;
        align-items: center;
        gap: 8px;
    }

    .status-indicator {
        width: 8px;
        height: 8px;
        border-radius: 50%;
    }

    .status-text {
        font-size: 0.875rem;
        color: #94a3b8;
    }

    .peer-actions {
        display: flex;
        align-items: center;
        gap: 8px;
    }

    .paired-badge {
        font-size: 0.75rem;
        padding: 4px 8px;
        background: rgba(74, 222, 128, 0.2);
        color: #4ade80;
        border-radius: 4px;
    }

    .btn-primary {
        background: #3b82f6;
        color: white;
        border: none;
        padding: 8px 16px;
        border-radius: 6px;
        cursor: pointer;
        font-size: 0.875rem;
        font-weight: 500;
        transition: background 0.2s;
    }

    .btn-primary:hover {
        background: #2563eb;
    }

    .btn-primary.small {
        padding: 6px 12px;
        font-size: 0.75rem;
    }

    .btn-secondary {
        background: rgba(255, 255, 255, 0.1);
        color: #f1f5f9;
        border: 1px solid rgba(255, 255, 255, 0.2);
        padding: 8px 16px;
        border-radius: 6px;
        cursor: pointer;
        font-size: 0.875rem;
        transition: all 0.2s;
    }

    .btn-secondary:hover {
        background: rgba(255, 255, 255, 0.15);
    }

    .btn-text {
        background: none;
        border: none;
        color: #64748b;
        cursor: pointer;
        padding: 4px 8px;
        font-size: 0.875rem;
    }

    .btn-text:hover {
        color: #94a3b8;
    }

    .btn-icon {
        background: none;
        border: none;
        padding: 6px;
        cursor: pointer;
        border-radius: 4px;
        display: flex;
        align-items: center;
        justify-content: center;
        transition: background 0.2s;
    }

    .btn-icon svg {
        width: 16px;
        height: 16px;
        color: #64748b;
    }

    .btn-icon:hover {
        background: rgba(255, 255, 255, 0.1);
    }

    .btn-icon.danger:hover svg {
        color: #ef4444;
    }
</style>
