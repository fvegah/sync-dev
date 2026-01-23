<script>
    import { onMount, onDestroy } from 'svelte';
    import {
        syncStatus,
        progressData,
        formattedSpeed,
        formattedETA,
        fileCountProgress,
        overallPercentage,
        activeFiles,
        recentEvents,
        folderPairs,
        peers
    } from '../stores/app.js';
    import {
        GetSyncStatus,
        GetSyncProgress,
        GetRecentEvents,
        GetFolderPairs,
        GetPeers,
        SyncNow,
        SyncFolderPair,
        AnalyzeFolderPair,
        FormatBytes
    } from '../../wailsjs/go/main/App.js';
    import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime.js';

    let isRefreshing = false;
    let isAnalyzing = false;
    let showPreview = false;
    let previewData = null;
    let selectedPairId = '';

    onMount(async () => {
        await loadData();

        // Set up event listeners
        EventsOn('sync:status', (data) => {
            syncStatus.set(data);
        });

        EventsOn('sync:progress', (data) => {
            progressData.set(data);
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
        progressData.set(progress);
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

    async function formatBytesAsync(bytes) {
        if (!bytes) return '0 B';
        const result = await FormatBytes(bytes);
        return result;
    }

    function formatBytesSync(bytes) {
        if (!bytes) return '0 B';
        const units = ['B', 'KB', 'MB', 'GB'];
        let size = bytes;
        let unitIndex = 0;
        while (size >= 1024 && unitIndex < units.length - 1) {
            size /= 1024;
            unitIndex++;
        }
        return `${size.toFixed(1)} ${units[unitIndex]}`;
    }

    function getFileName(path) {
        if (!path) return '';
        const parts = path.split('/');
        return parts[parts.length - 1] || path;
    }

    async function analyzeAllPairs() {
        if ($folderPairs.length === 0) {
            alert('No folder pairs configured');
            return;
        }

        isAnalyzing = true;
        showPreview = true;
        previewData = null;

        try {
            // Analyze first enabled pair for now
            const enabledPair = $folderPairs.find(p => p.enabled);
            if (enabledPair) {
                selectedPairId = enabledPair.id;
                previewData = await AnalyzeFolderPair(enabledPair.id);
            }
        } catch (err) {
            console.error('Analysis failed:', err);
            previewData = { error: err.toString() };
        }
        isAnalyzing = false;
    }

    async function analyzePair(pairId) {
        selectedPairId = pairId;
        isAnalyzing = true;
        showPreview = true;
        previewData = null;

        try {
            previewData = await AnalyzeFolderPair(pairId);
        } catch (err) {
            console.error('Analysis failed:', err);
            previewData = { error: err.toString() };
        }
        isAnalyzing = false;
    }

    async function syncFromPreview() {
        if (!selectedPairId) return;
        showPreview = false;
        isRefreshing = true;
        try {
            await SyncFolderPair(selectedPairId);
        } catch (err) {
            console.error('Sync failed:', err);
            alert('Sync failed: ' + err);
        }
        isRefreshing = false;
    }

    function closePreview() {
        showPreview = false;
        previewData = null;
    }

    $: enabledPairs = $folderPairs.filter(p => p.enabled);
</script>

<div class="sync-status">
    <div class="header">
        <h2>Sync Status</h2>
        <div class="header-actions">
            <button
                class="btn-secondary"
                on:click={analyzeAllPairs}
                disabled={isAnalyzing || enabledPairs.length === 0}
            >
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <circle cx="11" cy="11" r="8"/>
                    <path d="m21 21-4.3-4.3"/>
                </svg>
                {isAnalyzing ? 'Analyzing...' : 'Preview Changes'}
            </button>
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

        {#if $progressData && ($syncStatus.status === 'syncing' || $overallPercentage > 0)}
            <div class="progress-section">
                <!-- Overall progress header -->
                <div class="progress-header">
                    <span class="progress-title">Syncing Files</span>
                    <span class="progress-meta">
                        {#if $fileCountProgress}
                            <span class="file-count">{$fileCountProgress}</span>
                        {/if}
                    </span>
                </div>

                <!-- Global progress bar -->
                <div class="progress-info">
                    <span class="progress-percentage">{Math.round($overallPercentage)}%</span>
                    <span class="progress-stats">
                        {#if $formattedSpeed}
                            <span class="speed">{$formattedSpeed}</span>
                        {/if}
                        {#if $formattedETA}
                            <span class="eta">{$formattedETA}</span>
                        {/if}
                    </span>
                </div>
                <div class="progress-bar global">
                    <div class="progress-fill" style="width: {$overallPercentage}%"></div>
                </div>

                <!-- Active files list -->
                {#if $activeFiles.length > 0}
                    <div class="active-files-section">
                        <div class="active-files-header">
                            <span class="active-files-title">Active Transfers</span>
                            <span class="active-files-count">{$activeFiles.length}</span>
                        </div>
                        <div class="active-files-list">
                            {#each $activeFiles.slice(0, 10) as file}
                                <div class="active-file-item">
                                    <div class="active-file-info">
                                        <span class="active-file-name" title={file.path}>{getFileName(file.path)}</span>
                                        <span class="active-file-bytes">
                                            {formatBytesSync(file.bytesTransferred)} / {formatBytesSync(file.totalBytes)}
                                        </span>
                                    </div>
                                    <div class="active-file-progress-row">
                                        <div class="progress-bar file">
                                            <div class="progress-fill" style="width: {file.percentage}%"></div>
                                        </div>
                                        <span class="file-percentage">{Math.round(file.percentage)}%</span>
                                    </div>
                                </div>
                            {/each}
                        </div>
                    </div>
                {/if}
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

    {#if showPreview}
        <div class="preview-overlay" on:click={closePreview}>
            <div class="preview-modal" on:click|stopPropagation>
                <div class="preview-header">
                    <h3>Sync Preview</h3>
                    <button class="btn-close" on:click={closePreview}>×</button>
                </div>

                {#if isAnalyzing}
                    <div class="preview-loading">
                        <div class="spinner"></div>
                        <p>Analyzing changes...</p>
                    </div>
                {:else if previewData?.error}
                    <div class="preview-error">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <circle cx="12" cy="12" r="10"/>
                            <line x1="12" y1="8" x2="12" y2="12"/>
                            <line x1="12" y1="16" x2="12.01" y2="16"/>
                        </svg>
                        <p>{previewData.error}</p>
                    </div>
                {:else if previewData}
                    <div class="preview-content">
                        <div class="preview-info">
                            <div class="preview-peer">
                                <strong>{previewData.peerName}</strong>
                            </div>
                            <div class="preview-paths">
                                <span class="path">{previewData.localPath}</span>
                                <span class="arrow">⟷</span>
                                <span class="path">{previewData.remotePath}</span>
                            </div>
                        </div>

                        <div class="preview-summary">
                            <div class="summary-item push">
                                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                    <path d="M12 19V5M5 12l7-7 7 7"/>
                                </svg>
                                <div class="summary-text">
                                    <span class="count">{previewData.pushCount}</span>
                                    <span class="label">Files to Upload</span>
                                    <span class="size">{formatBytesSync(previewData.pushSize)}</span>
                                </div>
                            </div>
                            <div class="summary-item pull">
                                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                    <path d="M12 5v14M5 12l7 7 7-7"/>
                                </svg>
                                <div class="summary-text">
                                    <span class="count">{previewData.pullCount}</span>
                                    <span class="label">Files to Download</span>
                                    <span class="size">{formatBytesSync(previewData.pullSize)}</span>
                                </div>
                            </div>
                            {#if previewData.deleteCount > 0}
                                <div class="summary-item delete">
                                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                        <path d="M3 6h18M19 6v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6m3 0V4a2 2 0 012-2h4a2 2 0 012 2v2"/>
                                    </svg>
                                    <div class="summary-text">
                                        <span class="count">{previewData.deleteCount}</span>
                                        <span class="label">Files to Delete</span>
                                    </div>
                                </div>
                            {/if}
                        </div>

                        {#if previewData.toPush?.length > 0}
                            <div class="file-list">
                                <h4>Files to Upload</h4>
                                <ul>
                                    {#each previewData.toPush.slice(0, 10) as file}
                                        <li class="push">{file.path} <span class="file-size">{formatBytesSync(file.size)}</span></li>
                                    {/each}
                                    {#if previewData.toPush.length > 10}
                                        <li class="more">...and {previewData.toPush.length - 10} more</li>
                                    {/if}
                                </ul>
                            </div>
                        {/if}

                        {#if previewData.toPull?.length > 0}
                            <div class="file-list">
                                <h4>Files to Download</h4>
                                <ul>
                                    {#each previewData.toPull.slice(0, 10) as file}
                                        <li class="pull">{file.path} <span class="file-size">{formatBytesSync(file.size)}</span></li>
                                    {/each}
                                    {#if previewData.toPull.length > 10}
                                        <li class="more">...and {previewData.toPull.length - 10} more</li>
                                    {/if}
                                </ul>
                            </div>
                        {/if}

                        {#if previewData.pushCount === 0 && previewData.pullCount === 0 && previewData.deleteCount === 0}
                            <div class="preview-empty">
                                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                    <path d="M22 11.08V12a10 10 0 11-5.93-9.14"/>
                                    <polyline points="22 4 12 14.01 9 11.01"/>
                                </svg>
                                <p>Everything is in sync!</p>
                            </div>
                        {/if}
                    </div>
                {/if}

                <div class="preview-actions">
                    <button class="btn-secondary" on:click={closePreview}>Cancel</button>
                    {#if previewData && !previewData.error && (previewData.pushCount > 0 || previewData.pullCount > 0)}
                        <button class="btn-primary" on:click={syncFromPreview}>
                            Start Sync
                        </button>
                    {/if}
                </div>
            </div>
        </div>
    {/if}
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

    .progress-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 8px;
    }

    .progress-title {
        font-size: 0.875rem;
        font-weight: 500;
        color: #f1f5f9;
    }

    .progress-meta {
        font-size: 0.75rem;
        color: #94a3b8;
    }

    .file-count {
        background: rgba(59, 130, 246, 0.2);
        padding: 2px 8px;
        border-radius: 4px;
        color: #60a5fa;
    }

    .progress-percentage {
        font-size: 1.25rem;
        font-weight: 600;
        color: #f1f5f9;
    }

    .progress-stats {
        display: flex;
        gap: 12px;
        font-size: 0.8rem;
    }

    .speed {
        color: #4ade80;
        font-weight: 500;
    }

    .eta {
        color: #94a3b8;
    }

    .progress-bar.global {
        height: 8px;
        margin-bottom: 12px;
    }

    .progress-bar.file {
        height: 4px;
    }

    .active-files-section {
        margin-top: 12px;
        border-top: 1px solid rgba(255, 255, 255, 0.1);
        padding-top: 12px;
    }

    .active-files-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 8px;
    }

    .active-files-title {
        font-size: 0.75rem;
        font-weight: 500;
        color: #94a3b8;
        text-transform: uppercase;
        letter-spacing: 0.05em;
    }

    .active-files-count {
        font-size: 0.7rem;
        background: rgba(59, 130, 246, 0.2);
        color: #60a5fa;
        padding: 2px 6px;
        border-radius: 4px;
    }

    .active-files-list {
        max-height: 200px;
        overflow-y: auto;
        display: flex;
        flex-direction: column;
        gap: 8px;
    }

    /* Custom scrollbar styling */
    .active-files-list::-webkit-scrollbar {
        width: 6px;
    }

    .active-files-list::-webkit-scrollbar-track {
        background: rgba(0, 0, 0, 0.2);
        border-radius: 3px;
    }

    .active-files-list::-webkit-scrollbar-thumb {
        background: rgba(255, 255, 255, 0.2);
        border-radius: 3px;
    }

    .active-files-list::-webkit-scrollbar-thumb:hover {
        background: rgba(255, 255, 255, 0.3);
    }

    .active-file-item {
        background: rgba(0, 0, 0, 0.15);
        border-radius: 6px;
        padding: 8px 10px;
    }

    .active-file-info {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 6px;
    }

    .active-file-name {
        color: #e2e8f0;
        font-size: 0.8rem;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
        max-width: 60%;
    }

    .active-file-bytes {
        font-size: 0.7rem;
        color: #64748b;
        font-family: monospace;
    }

    .active-file-progress-row {
        display: flex;
        align-items: center;
        gap: 8px;
    }

    .active-file-progress-row .progress-bar {
        flex: 1;
    }

    .file-percentage {
        font-size: 0.7rem;
        color: #60a5fa;
        font-weight: 500;
        min-width: 32px;
        text-align: right;
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

    .header-actions {
        display: flex;
        gap: 8px;
    }

    .btn-secondary {
        display: flex;
        align-items: center;
        gap: 8px;
        background: rgba(255, 255, 255, 0.1);
        color: #f1f5f9;
        border: 1px solid rgba(255, 255, 255, 0.2);
        padding: 8px 16px;
        border-radius: 6px;
        cursor: pointer;
        font-size: 0.875rem;
        transition: all 0.2s;
    }

    .btn-secondary:hover:not(:disabled) {
        background: rgba(255, 255, 255, 0.15);
    }

    .btn-secondary:disabled {
        opacity: 0.5;
        cursor: not-allowed;
    }

    .btn-secondary svg {
        width: 16px;
        height: 16px;
    }

    .preview-overlay {
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

    .preview-modal {
        background: #1e293b;
        border-radius: 12px;
        padding: 24px;
        min-width: 500px;
        max-width: 600px;
        max-height: 80vh;
        overflow-y: auto;
        border: 1px solid rgba(255, 255, 255, 0.1);
    }

    .preview-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 20px;
    }

    .preview-header h3 {
        margin: 0;
        font-size: 1.25rem;
        color: #f1f5f9;
    }

    .btn-close {
        background: none;
        border: none;
        color: #64748b;
        font-size: 1.5rem;
        cursor: pointer;
        padding: 0;
        line-height: 1;
    }

    .btn-close:hover {
        color: #94a3b8;
    }

    .preview-loading {
        text-align: center;
        padding: 40px;
    }

    .spinner {
        width: 40px;
        height: 40px;
        border: 3px solid rgba(255, 255, 255, 0.1);
        border-top-color: #3b82f6;
        border-radius: 50%;
        animation: spin 1s linear infinite;
        margin: 0 auto 16px;
    }

    .preview-error {
        text-align: center;
        padding: 40px;
        color: #ef4444;
    }

    .preview-error svg {
        width: 48px;
        height: 48px;
        margin-bottom: 16px;
    }

    .preview-content {
        margin-bottom: 20px;
    }

    .preview-info {
        background: rgba(0, 0, 0, 0.2);
        border-radius: 8px;
        padding: 16px;
        margin-bottom: 16px;
    }

    .preview-peer {
        margin-bottom: 8px;
        color: #f1f5f9;
    }

    .preview-paths {
        display: flex;
        align-items: center;
        gap: 8px;
        font-size: 0.875rem;
    }

    .preview-paths .path {
        color: #94a3b8;
        font-family: monospace;
        font-size: 0.75rem;
        max-width: 180px;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }

    .preview-paths .arrow {
        color: #3b82f6;
    }

    .preview-summary {
        display: flex;
        gap: 12px;
        margin-bottom: 16px;
    }

    .summary-item {
        flex: 1;
        display: flex;
        align-items: center;
        gap: 12px;
        padding: 12px;
        background: rgba(0, 0, 0, 0.2);
        border-radius: 8px;
    }

    .summary-item svg {
        width: 24px;
        height: 24px;
    }

    .summary-item.push svg {
        color: #4ade80;
    }

    .summary-item.pull svg {
        color: #60a5fa;
    }

    .summary-item.delete svg {
        color: #f59e0b;
    }

    .summary-text {
        display: flex;
        flex-direction: column;
    }

    .summary-text .count {
        font-size: 1.25rem;
        font-weight: 600;
        color: #f1f5f9;
    }

    .summary-text .label {
        font-size: 0.75rem;
        color: #64748b;
    }

    .summary-text .size {
        font-size: 0.75rem;
        color: #94a3b8;
    }

    .file-list {
        margin-bottom: 16px;
    }

    .file-list h4 {
        font-size: 0.875rem;
        color: #94a3b8;
        margin: 0 0 8px 0;
    }

    .file-list ul {
        list-style: none;
        padding: 0;
        margin: 0;
        background: rgba(0, 0, 0, 0.2);
        border-radius: 8px;
        max-height: 150px;
        overflow-y: auto;
    }

    .file-list li {
        padding: 8px 12px;
        font-size: 0.8rem;
        font-family: monospace;
        border-bottom: 1px solid rgba(255, 255, 255, 0.05);
        display: flex;
        justify-content: space-between;
    }

    .file-list li:last-child {
        border-bottom: none;
    }

    .file-list li.push {
        color: #4ade80;
    }

    .file-list li.pull {
        color: #60a5fa;
    }

    .file-list li.more {
        color: #64748b;
        font-style: italic;
    }

    .file-size {
        color: #64748b;
        font-size: 0.75rem;
    }

    .preview-empty {
        text-align: center;
        padding: 40px;
        color: #4ade80;
    }

    .preview-empty svg {
        width: 48px;
        height: 48px;
        margin-bottom: 16px;
    }

    .preview-actions {
        display: flex;
        justify-content: flex-end;
        gap: 12px;
        padding-top: 16px;
        border-top: 1px solid rgba(255, 255, 255, 0.1);
    }
</style>
