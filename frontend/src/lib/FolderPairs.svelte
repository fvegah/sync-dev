<script>
    import { onMount } from 'svelte';
    import { folderPairs, peers, showModal } from '../stores/app.js';
    import {
        GetFolderPairs,
        GetPeers,
        AddFolderPair,
        RemoveFolderPair,
        UpdateFolderPair,
        SelectFolder,
        SyncFolderPair
    } from '../../wailsjs/go/main/App.js';

    let showAddForm = false;
    let selectedPeerId = '';
    let localPath = '';
    let remotePath = '';

    onMount(async () => {
        await loadData();
    });

    async function loadData() {
        const [pairs, peerList] = await Promise.all([
            GetFolderPairs(),
            GetPeers()
        ]);
        folderPairs.set(pairs || []);
        peers.set(peerList || []);
    }

    function getPairedPeers() {
        return $peers.filter(p => p.paired);
    }

    function getPeerName(peerId) {
        const peer = $peers.find(p => p.id === peerId);
        return peer ? peer.name : 'Unknown';
    }

    async function browseLocalFolder() {
        const path = await SelectFolder();
        if (path) {
            localPath = path;
        }
    }

    async function addPair() {
        if (!selectedPeerId || !localPath || !remotePath) {
            alert('Please fill in all fields');
            return;
        }

        try {
            await AddFolderPair(selectedPeerId, localPath, remotePath);
            showAddForm = false;
            selectedPeerId = '';
            localPath = '';
            remotePath = '';
            await loadData();
        } catch (err) {
            alert('Failed to add folder pair: ' + err);
        }
    }

    async function removePair(id) {
        if (confirm('Remove this folder pair? Files will not be deleted.')) {
            await RemoveFolderPair(id);
            await loadData();
        }
    }

    async function toggleEnabled(pair) {
        await UpdateFolderPair(pair.id, !pair.enabled, pair.exclusions || []);
        await loadData();
    }

    async function syncNow(pair) {
        try {
            await SyncFolderPair(pair.id);
        } catch (err) {
            alert('Sync failed: ' + err);
        }
    }

    function formatDate(dateStr) {
        if (!dateStr) return 'Never';
        const date = new Date(dateStr);
        return date.toLocaleString();
    }
</script>

<div class="folder-pairs">
    <div class="header">
        <h2>Folder Pairs</h2>
        {#if getPairedPeers().length > 0}
            <button class="btn-primary" on:click={() => showAddForm = !showAddForm}>
                {showAddForm ? 'Cancel' : 'Add Folder Pair'}
            </button>
        {/if}
    </div>

    {#if showAddForm}
        <div class="add-form">
            <div class="form-group">
                <label>Paired Device</label>
                <select bind:value={selectedPeerId}>
                    <option value="">Select a device...</option>
                    {#each getPairedPeers() as peer}
                        <option value={peer.id}>{peer.name}</option>
                    {/each}
                </select>
            </div>

            <div class="form-group">
                <label>Local Folder</label>
                <div class="path-input">
                    <input type="text" bind:value={localPath} placeholder="/Users/you/Documents/Projects" />
                    <button class="btn-secondary" on:click={browseLocalFolder}>Browse</button>
                </div>
            </div>

            <div class="form-group">
                <label>Remote Folder (on paired device)</label>
                <input type="text" bind:value={remotePath} placeholder="/Users/other/Documents/Projects" />
            </div>

            <div class="form-actions">
                <button class="btn-primary" on:click={addPair}>Add Folder Pair</button>
            </div>
        </div>
    {/if}

    <div class="pairs-list">
        {#if $folderPairs.length === 0}
            <div class="empty-state">
                {#if getPairedPeers().length === 0}
                    <p>No paired devices yet.</p>
                    <p class="hint">Go to the Devices tab to pair with another Mac first.</p>
                {:else}
                    <p>No folder pairs configured.</p>
                    <p class="hint">Click "Add Folder Pair" to start syncing folders.</p>
                {/if}
            </div>
        {:else}
            {#each $folderPairs as pair}
                <div class="pair-card" class:disabled={!pair.enabled}>
                    <div class="pair-header">
                        <div class="pair-toggle">
                            <input
                                type="checkbox"
                                checked={pair.enabled}
                                on:change={() => toggleEnabled(pair)}
                            />
                        </div>
                        <div class="pair-info">
                            <div class="peer-name">
                                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                    <rect x="2" y="3" width="20" height="14" rx="2"/>
                                    <path d="M8 21h8M12 17v4"/>
                                </svg>
                                {getPeerName(pair.peerId)}
                            </div>
                        </div>
                        <div class="pair-actions">
                            <button
                                class="btn-icon"
                                on:click={() => syncNow(pair)}
                                title="Sync now"
                                disabled={!pair.enabled}
                            >
                                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                    <path d="M21 12a9 9 0 11-6.219-8.56"/>
                                    <polyline points="21 3 21 9 15 9"/>
                                </svg>
                            </button>
                            <button
                                class="btn-icon danger"
                                on:click={() => removePair(pair.id)}
                                title="Remove"
                            >
                                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                    <path d="M3 6h18M19 6v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6m3 0V4a2 2 0 012-2h4a2 2 0 012 2v2"/>
                                </svg>
                            </button>
                        </div>
                    </div>
                    <div class="pair-paths">
                        <div class="path-row">
                            <span class="path-label">Local:</span>
                            <span class="path-value">{pair.localPath}</span>
                        </div>
                        <div class="sync-arrow">
                            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                <path d="M7 16l-4-4 4-4M17 8l4 4-4 4M3 12h18"/>
                            </svg>
                        </div>
                        <div class="path-row">
                            <span class="path-label">Remote:</span>
                            <span class="path-value">{pair.remotePath}</span>
                        </div>
                    </div>
                    <div class="pair-footer">
                        <span class="last-sync">Last sync: {formatDate(pair.lastSyncTime)}</span>
                    </div>
                </div>
            {/each}
        {/if}
    </div>
</div>

<style>
    .folder-pairs {
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

    .add-form {
        background: rgba(255, 255, 255, 0.05);
        border-radius: 8px;
        padding: 20px;
        margin-bottom: 20px;
    }

    .form-group {
        margin-bottom: 16px;
    }

    .form-group label {
        display: block;
        margin-bottom: 8px;
        font-size: 0.875rem;
        color: #94a3b8;
    }

    .form-group input,
    .form-group select {
        width: 100%;
        padding: 10px 12px;
        background: rgba(0, 0, 0, 0.3);
        border: 1px solid rgba(255, 255, 255, 0.1);
        border-radius: 6px;
        color: #f1f5f9;
        font-size: 0.875rem;
    }

    .form-group select {
        cursor: pointer;
    }

    .path-input {
        display: flex;
        gap: 8px;
    }

    .path-input input {
        flex: 1;
    }

    .form-actions {
        margin-top: 20px;
    }

    .pairs-list {
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

    .pair-card {
        background: rgba(255, 255, 255, 0.05);
        border-radius: 8px;
        padding: 16px;
        margin-bottom: 12px;
        border: 1px solid rgba(255, 255, 255, 0.1);
        transition: all 0.2s;
    }

    .pair-card.disabled {
        opacity: 0.5;
    }

    .pair-header {
        display: flex;
        align-items: center;
        gap: 12px;
        margin-bottom: 12px;
    }

    .pair-toggle input {
        width: 18px;
        height: 18px;
        cursor: pointer;
    }

    .pair-info {
        flex: 1;
    }

    .peer-name {
        display: flex;
        align-items: center;
        gap: 8px;
        font-weight: 500;
    }

    .peer-name svg {
        width: 18px;
        height: 18px;
        color: #64748b;
    }

    .pair-actions {
        display: flex;
        gap: 4px;
    }

    .pair-paths {
        background: rgba(0, 0, 0, 0.2);
        border-radius: 6px;
        padding: 12px;
        margin-bottom: 12px;
    }

    .path-row {
        display: flex;
        align-items: center;
        gap: 8px;
        font-size: 0.875rem;
    }

    .path-label {
        color: #64748b;
        min-width: 60px;
    }

    .path-value {
        font-family: monospace;
        color: #94a3b8;
        word-break: break-all;
    }

    .sync-arrow {
        display: flex;
        justify-content: center;
        padding: 8px 0;
    }

    .sync-arrow svg {
        width: 20px;
        height: 20px;
        color: #3b82f6;
    }

    .pair-footer {
        font-size: 0.75rem;
        color: #64748b;
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
        width: 18px;
        height: 18px;
        color: #64748b;
    }

    .btn-icon:hover {
        background: rgba(255, 255, 255, 0.1);
    }

    .btn-icon:hover svg {
        color: #94a3b8;
    }

    .btn-icon.danger:hover svg {
        color: #ef4444;
    }

    .btn-icon:disabled {
        opacity: 0.5;
        cursor: not-allowed;
    }
</style>
