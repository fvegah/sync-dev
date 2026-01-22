<script>
    import { currentTab, modalState, closeModal, pairingState } from './stores/app.js';
    import PeerList from './lib/PeerList.svelte';
    import FolderPairs from './lib/FolderPairs.svelte';
    import SyncStatus from './lib/SyncStatus.svelte';
    import Settings from './lib/Settings.svelte';
    import { RequestPairing } from '../wailsjs/go/main/App.js';

    let pairingInputCode = '';

    const tabs = [
        { id: 'peers', label: 'Devices', icon: 'devices' },
        { id: 'folders', label: 'Folders', icon: 'folder' },
        { id: 'sync', label: 'Sync', icon: 'sync' },
        { id: 'settings', label: 'Settings', icon: 'settings' }
    ];

    function setTab(tabId) {
        currentTab.set(tabId);
    }

    async function submitPairing() {
        if (!$modalState.data || !pairingInputCode) return;

        try {
            await RequestPairing($modalState.data.id, pairingInputCode);
            pairingState.set({ code: pairingInputCode, isPairing: true, targetPeer: $modalState.data });
            closeModal();
            pairingInputCode = '';
        } catch (err) {
            alert('Pairing failed: ' + err);
        }
    }
</script>

<main>
    <div class="app-container">
        <nav class="sidebar">
            <div class="app-title">
                <div class="logo">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <path d="M21 16V8a2 2 0 00-1-1.73l-7-4a2 2 0 00-2 0l-7 4A2 2 0 003 8v8a2 2 0 001 1.73l7 4a2 2 0 002 0l7-4A2 2 0 0021 16z"/>
                        <polyline points="3.27 6.96 12 12.01 20.73 6.96"/>
                        <line x1="12" y1="22.08" x2="12" y2="12"/>
                    </svg>
                </div>
                <span>SyncDev</span>
            </div>

            <div class="nav-items">
                {#each tabs as tab}
                    <button
                        class="nav-item"
                        class:active={$currentTab === tab.id}
                        on:click={() => setTab(tab.id)}
                    >
                        {#if tab.icon === 'devices'}
                            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                <rect x="2" y="3" width="20" height="14" rx="2"/>
                                <path d="M8 21h8M12 17v4"/>
                            </svg>
                        {:else if tab.icon === 'folder'}
                            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                <path d="M22 19a2 2 0 01-2 2H4a2 2 0 01-2-2V5a2 2 0 012-2h5l2 3h9a2 2 0 012 2z"/>
                            </svg>
                        {:else if tab.icon === 'sync'}
                            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                <path d="M21 12a9 9 0 11-6.219-8.56"/>
                                <polyline points="21 3 21 9 15 9"/>
                            </svg>
                        {:else if tab.icon === 'settings'}
                            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                <circle cx="12" cy="12" r="3"/>
                                <path d="M19.4 15a1.65 1.65 0 00.33 1.82l.06.06a2 2 0 010 2.83 2 2 0 01-2.83 0l-.06-.06a1.65 1.65 0 00-1.82-.33 1.65 1.65 0 00-1 1.51V21a2 2 0 01-2 2 2 2 0 01-2-2v-.09A1.65 1.65 0 009 19.4a1.65 1.65 0 00-1.82.33l-.06.06a2 2 0 01-2.83 0 2 2 0 010-2.83l.06-.06a1.65 1.65 0 00.33-1.82 1.65 1.65 0 00-1.51-1H3a2 2 0 01-2-2 2 2 0 012-2h.09A1.65 1.65 0 004.6 9a1.65 1.65 0 00-.33-1.82l-.06-.06a2 2 0 010-2.83 2 2 0 012.83 0l.06.06a1.65 1.65 0 001.82.33H9a1.65 1.65 0 001-1.51V3a2 2 0 012-2 2 2 0 012 2v.09a1.65 1.65 0 001 1.51 1.65 1.65 0 001.82-.33l.06-.06a2 2 0 012.83 0 2 2 0 010 2.83l-.06.06a1.65 1.65 0 00-.33 1.82V9a1.65 1.65 0 001.51 1H21a2 2 0 012 2 2 2 0 01-2 2h-.09a1.65 1.65 0 00-1.51 1z"/>
                            </svg>
                        {/if}
                        <span>{tab.label}</span>
                    </button>
                {/each}
            </div>
        </nav>

        <div class="content">
            {#if $currentTab === 'peers'}
                <PeerList />
            {:else if $currentTab === 'folders'}
                <FolderPairs />
            {:else if $currentTab === 'sync'}
                <SyncStatus />
            {:else if $currentTab === 'settings'}
                <Settings />
            {/if}
        </div>
    </div>

    {#if $modalState.show && $modalState.type === 'pairing'}
        <div class="modal-overlay" on:click={closeModal}>
            <div class="modal" on:click|stopPropagation>
                <h3>Pair with {$modalState.data?.name}</h3>
                <p>Enter the 6-digit code shown on the other device:</p>
                <input
                    type="text"
                    maxlength="6"
                    bind:value={pairingInputCode}
                    placeholder="000000"
                    class="code-input"
                />
                <div class="modal-actions">
                    <button class="btn-secondary" on:click={closeModal}>Cancel</button>
                    <button class="btn-primary" on:click={submitPairing} disabled={pairingInputCode.length !== 6}>
                        Pair
                    </button>
                </div>
            </div>
        </div>
    {/if}
</main>

<style>
    :global(*) {
        box-sizing: border-box;
    }

    :global(body) {
        margin: 0;
        padding: 0;
        font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif;
        background: #0f172a;
        color: #f1f5f9;
        overflow: hidden;
    }

    main {
        width: 100vw;
        height: 100vh;
    }

    .app-container {
        display: flex;
        height: 100%;
    }

    .sidebar {
        width: 200px;
        background: rgba(15, 23, 42, 0.95);
        border-right: 1px solid rgba(255, 255, 255, 0.1);
        display: flex;
        flex-direction: column;
        padding-top: 30px; /* Space for macOS traffic lights */
        -webkit-app-region: drag;
    }

    .app-title {
        display: flex;
        align-items: center;
        gap: 12px;
        padding: 16px 20px;
        margin-bottom: 20px;
        -webkit-app-region: no-drag;
    }

    .logo {
        width: 32px;
        height: 32px;
        display: flex;
        align-items: center;
        justify-content: center;
    }

    .logo svg {
        width: 28px;
        height: 28px;
        color: #3b82f6;
    }

    .app-title span {
        font-size: 1.25rem;
        font-weight: 600;
    }

    .nav-items {
        flex: 1;
        padding: 0 12px;
        -webkit-app-region: no-drag;
    }

    .nav-item {
        display: flex;
        align-items: center;
        gap: 12px;
        width: 100%;
        padding: 12px 16px;
        background: none;
        border: none;
        border-radius: 8px;
        color: #94a3b8;
        font-size: 0.875rem;
        cursor: pointer;
        transition: all 0.2s;
        margin-bottom: 4px;
    }

    .nav-item:hover {
        background: rgba(255, 255, 255, 0.05);
        color: #f1f5f9;
    }

    .nav-item.active {
        background: rgba(59, 130, 246, 0.2);
        color: #60a5fa;
    }

    .nav-item svg {
        width: 20px;
        height: 20px;
    }

    .content {
        flex: 1;
        overflow: hidden;
        background: linear-gradient(135deg, #1e293b 0%, #0f172a 100%);
    }

    .modal-overlay {
        position: fixed;
        top: 0;
        left: 0;
        right: 0;
        bottom: 0;
        background: rgba(0, 0, 0, 0.7);
        display: flex;
        align-items: center;
        justify-content: center;
        z-index: 1000;
    }

    .modal {
        background: #1e293b;
        border-radius: 12px;
        padding: 24px;
        width: 100%;
        max-width: 400px;
        border: 1px solid rgba(255, 255, 255, 0.1);
    }

    .modal h3 {
        margin: 0 0 12px 0;
        font-size: 1.25rem;
    }

    .modal p {
        margin: 0 0 16px 0;
        color: #94a3b8;
        font-size: 0.875rem;
    }

    .code-input {
        width: 100%;
        padding: 16px;
        background: rgba(0, 0, 0, 0.3);
        border: 1px solid rgba(255, 255, 255, 0.1);
        border-radius: 8px;
        color: #f1f5f9;
        font-size: 1.5rem;
        text-align: center;
        letter-spacing: 8px;
        font-family: monospace;
    }

    .code-input:focus {
        outline: none;
        border-color: #3b82f6;
    }

    .modal-actions {
        display: flex;
        gap: 12px;
        margin-top: 20px;
        justify-content: flex-end;
    }

    .btn-primary {
        background: #3b82f6;
        color: white;
        border: none;
        padding: 10px 20px;
        border-radius: 6px;
        cursor: pointer;
        font-size: 0.875rem;
        font-weight: 500;
        transition: background 0.2s;
    }

    .btn-primary:hover:not(:disabled) {
        background: #2563eb;
    }

    .btn-primary:disabled {
        opacity: 0.5;
        cursor: not-allowed;
    }

    .btn-secondary {
        background: rgba(255, 255, 255, 0.1);
        color: #f1f5f9;
        border: 1px solid rgba(255, 255, 255, 0.2);
        padding: 10px 20px;
        border-radius: 6px;
        cursor: pointer;
        font-size: 0.875rem;
        transition: all 0.2s;
    }

    .btn-secondary:hover {
        background: rgba(255, 255, 255, 0.15);
    }
</style>
