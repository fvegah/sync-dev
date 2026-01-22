<script>
    import { onMount, onDestroy } from 'svelte';
    import { syncStatus, transferProgress, recentEvents, folderPairs, peers } from '../stores/app.js';
    import {
        GetSyncStatus,
        GetSyncProgress,
        GetRecentEvents,
        GetFolderPairs,
        GetPeers,
        SyncNow,
        FormatBytes
    } from '../../wailsjs/go/main/App.js';
    import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime.js';

    let isRefreshing = false;

    onMount(async () => {
        await loadData();

        // Set up event listeners
        EventsOn('sync:status', (data) => {
            syncStatus.set(data);
        });

        EventsOn('sync:progress', (data) => {
            transferProgress.set(data);
        });

        EventsOn('sync:event', async (data) => {
            recentEvents.update(events => [data, ...events].slice(0, 50));
        });
    });

    onDestroy(() => {
        EventsOff('sync:status');
        EventsOff('sync:progress');
        EventsOff('sync:event');
    });

    async function loadData() {
        const [status, progress, events, pairs, peerList] = await Promise.all([
            GetSyncStatus(),
            GetSyncProgress(),
            GetRecentEvents(),
            GetFolderPairs(),
            GetPeers()
        ]);

        syncStatus.set(status);
        transferProgress.set(progress);
        recentEvents.set(events || []);
        folderPairs.set(pairs || []);
        peers.set(peerList || []);
    }

    async function triggerSync() {
        isRefreshing = true;
        try {
            await SyncNow();
        } catch (err) {
            console.error('Sync failed:', err);
        }
        isRefreshing = false;
    }

    function getStatusIcon(status) {
        switch (status) {
            case 'syncing': return 'syncing';
            case 'scanning': return 'scanning';
            case 'error': return 'error';
            default: return 'idle';
        }
    }

    function getStatusText(status) {
        switch (status) {
            case 'syncing': return 'Syncing';
            case 'scanning': return 'Scanning';
            case 'error': return 'Error';
            default: return 'Ready';
        }
    }

    function getStatusColor(status) {
        switch (status) {
            case 'syncing': return '#3b82f6';
            case 'scanning': return '#f59e0b';
            case 'error': return '#ef4444';
            default: return '#4ade80';
        }
    }

    function getEventIcon(type) {
        switch (type) {
            case 'push': return '↑';
            case 'pull': return '↓';
            case 'delete': return '×';
            case 'error': return '!';
            default: return '•';
        }
    }

    function getEventColor(type) {
        switch (type) {
            case 'push': return '#4ade80';
            case 'pull': return '#60a5fa';
            case 'delete': return '#f59e0b';
            case 'error': return '#ef4444';
            default: return '#94a3b8';
        }
    }

    function formatTime(dateStr) {
        if (!dateStr) return '';
        const date = new Date(dateStr);
        return date.toLocaleTimeString();
    }

    function getPeerName(peerId) {
        const peer = $peers.find(p => p.id === peerId);
        return peer ? peer.name : 'Unknown';
    }

    async function formatBytes(bytes) {
        if (!bytes) return '0 B';
        const result = await FormatBytes(bytes);
        return result;
    }

    $: progressPercentage = $transferProgress ? $transferProgress.percentage : 0;
</script>

<div class="sync-status">
    <div class="header">
        <h2>Sync Status</h2>
        <button
            class="btn-primary"
            class:spinning={isRefreshing || $syncStatus.status === 'syncing'}
            on:click={triggerSync}
            disabled={$syncStatus.status === 'syncing' || $syncStatus.status === 'scanning'}
        >
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M21 12a9 9 0 11-6.219-8.56"/>
                <polyline points="21 3 21 9 15 9"/>
            </svg>
            Sync Now
        </button>
    </div>

    <div class="status-card">
        <div class="status-main">
            <div class="status-indicator" style="background-color: {getStatusColor($syncStatus.status)}">
                {#if $syncStatus.status === 'syncing' || $syncStatus.status === 'scanning'}
                    <div class="pulse"></div>
                {/if}
            </div>
            <div class="status-info">
                <span class="status-label">{getStatusText($syncStatus.status)}</span>
                {#if $syncStatus.action}
                    <span class="status-action">{$syncStatus.action}</span>
                {/if}
            </div>
        </div>

        {#if $transferProgress && ($syncStatus.status === 'syncing' || progressPercentage > 0)}
            <div class="progress-section">
                <div class="progress-info">
                    <span class="file-name">{$transferProgress.fileName}</span>
                    <span class="progress-stats">
                        {Math.round(progressPercentage)}%
                    </span>
                </div>
                <div class="progress-bar">
                    <div class="progress-fill" style="width: {progressPercentage}%"></div>
                </div>
            </div>
        {/if}

        <div class="stats-row">
            <div class="stat">
                <span class="stat-value">{$folderPairs.filter(p => p.enabled).length}</span>
                <span class="stat-label">Active Pairs</span>
            </div>
            <div class="stat">
                <span class="stat-value">{$peers.filter(p => p.paired && p.status === 'online').length}</span>
                <span class="stat-label">Peers Online</span>
            </div>
            <div class="stat">
                <span class="stat-value">{$recentEvents.filter(e => e.type !== 'error').length}</span>
                <span class="stat-label">Files Synced</span>
            </div>
        </div>
    </div>

    <div class="activity-section">
        <h3>Recent Activity</h3>
        <div class="activity-list">
            {#if $recentEvents.length === 0}
                <div class="empty-state">
                    <p>No sync activity yet.</p>
                </div>
            {:else}
                {#each $recentEvents as event}
                    <div class="activity-item">
                        <div class="event-icon" style="color: {getEventColor(event.type)}">
                            {getEventIcon(event.type)}
                        </div>
                        <div class="event-info">
                            <span class="event-file">{event.filePath || event.description}</span>
                            <span class="event-meta">
                                {event.peerName || ''} • {formatTime(event.time)}
                            </span>
                        </div>
                        <div class="event-type" style="color: {getEventColor(event.type)}">
                            {event.type}
                        </div>
                    </div>
                {/each}
            {/if}
        </div>
    </div>
</div>

<style>
    .sync-status {
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

    h3 {
        margin: 0 0 12px 0;
        font-size: 1rem;
        font-weight: 500;
        color: #94a3b8;
    }

    .btn-primary {
        display: flex;
        align-items: center;
        gap: 8px;
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

    .btn-primary:hover:not(:disabled) {
        background: #2563eb;
    }

    .btn-primary:disabled {
        opacity: 0.7;
        cursor: not-allowed;
    }

    .btn-primary svg {
        width: 18px;
        height: 18px;
    }

    .btn-primary.spinning svg {
        animation: spin 1s linear infinite;
    }

    @keyframes spin {
        from { transform: rotate(0deg); }
        to { transform: rotate(360deg); }
    }

    .status-card {
        background: rgba(255, 255, 255, 0.05);
        border-radius: 12px;
        padding: 20px;
        margin-bottom: 20px;
    }

    .status-main {
        display: flex;
        align-items: center;
        gap: 16px;
        margin-bottom: 16px;
    }

    .status-indicator {
        width: 48px;
        height: 48px;
        border-radius: 12px;
        position: relative;
        display: flex;
        align-items: center;
        justify-content: center;
    }

    .pulse {
        position: absolute;
        width: 100%;
        height: 100%;
        border-radius: 12px;
        background: inherit;
        animation: pulse 1.5s ease-out infinite;
    }

    @keyframes pulse {
        0% {
            transform: scale(1);
            opacity: 0.5;
        }
        100% {
            transform: scale(1.5);
            opacity: 0;
        }
    }

    .status-info {
        display: flex;
        flex-direction: column;
        gap: 4px;
    }

    .status-label {
        font-size: 1.25rem;
        font-weight: 600;
    }

    .status-action {
        font-size: 0.875rem;
        color: #94a3b8;
    }

    .progress-section {
        margin-bottom: 16px;
        padding: 12px;
        background: rgba(0, 0, 0, 0.2);
        border-radius: 8px;
    }

    .progress-info {
        display: flex;
        justify-content: space-between;
        margin-bottom: 8px;
        font-size: 0.875rem;
    }

    .file-name {
        color: #94a3b8;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
        max-width: 70%;
    }

    .progress-stats {
        color: #60a5fa;
        font-weight: 500;
    }

    .progress-bar {
        height: 6px;
        background: rgba(255, 255, 255, 0.1);
        border-radius: 3px;
        overflow: hidden;
    }

    .progress-fill {
        height: 100%;
        background: linear-gradient(90deg, #3b82f6, #60a5fa);
        border-radius: 3px;
        transition: width 0.3s ease;
    }

    .stats-row {
        display: flex;
        gap: 16px;
    }

    .stat {
        flex: 1;
        text-align: center;
        padding: 12px;
        background: rgba(0, 0, 0, 0.2);
        border-radius: 8px;
    }

    .stat-value {
        display: block;
        font-size: 1.5rem;
        font-weight: 600;
        color: #f1f5f9;
    }

    .stat-label {
        font-size: 0.75rem;
        color: #64748b;
    }

    .activity-section {
        flex: 1;
        overflow: hidden;
        display: flex;
        flex-direction: column;
    }

    .activity-list {
        flex: 1;
        overflow-y: auto;
    }

    .empty-state {
        text-align: center;
        padding: 40px;
        color: #64748b;
    }

    .activity-item {
        display: flex;
        align-items: center;
        gap: 12px;
        padding: 12px;
        background: rgba(255, 255, 255, 0.03);
        border-radius: 8px;
        margin-bottom: 8px;
    }

    .event-icon {
        width: 24px;
        height: 24px;
        display: flex;
        align-items: center;
        justify-content: center;
        font-size: 1rem;
        font-weight: bold;
    }

    .event-info {
        flex: 1;
        overflow: hidden;
    }

    .event-file {
        display: block;
        font-size: 0.875rem;
        color: #f1f5f9;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }

    .event-meta {
        font-size: 0.75rem;
        color: #64748b;
    }

    .event-type {
        font-size: 0.75rem;
        font-weight: 500;
        text-transform: uppercase;
    }
</style>
