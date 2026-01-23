<script>
    import { onMount, onDestroy } from 'svelte';
    import { RefreshCw, Search, AlertCircle, Check, ArrowUp, ArrowDown, Trash2, ArrowLeftRight, X } from 'lucide-svelte';
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
        AnalyzeFolderPair
    } from '../../bindings/SyncDev/app.js';
    import { Events } from '@wailsio/runtime';

    let isRefreshing = $state(false);
    let isAnalyzing = $state(false);
    let showPreview = $state(false);
    let previewData = $state(null);
    let selectedPairId = $state('');

    onMount(async () => {
        await loadData();

        Events.On('sync:status', (data) => {
            syncStatus.set(data);
        });

        Events.On('sync:progress', (data) => {
            progressData.set(data);
        });

        Events.On('sync:event', async (data) => {
            recentEvents.update(events => [data, ...events].slice(0, 50));
        });
    });

    onDestroy(() => {
        Events.Off('sync:status');
        Events.Off('sync:progress');
        Events.Off('sync:event');
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

    function getStatusColor(status) {
        switch (status) {
            case 'syncing': return 'bg-macos-blue';
            case 'scanning': return 'bg-macos-orange';
            case 'error': return 'bg-macos-red';
            default: return 'bg-macos-green';
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

    function getEventIcon(type) {
        switch (type) {
            case 'push': return ArrowUp;
            case 'pull': return ArrowDown;
            case 'delete': return Trash2;
            case 'error': return AlertCircle;
            default: return RefreshCw;
        }
    }

    function getEventColor(type) {
        switch (type) {
            case 'push': return 'text-macos-green';
            case 'pull': return 'text-macos-blue';
            case 'delete': return 'text-macos-orange';
            case 'error': return 'text-macos-red';
            default: return 'text-slate-400';
        }
    }

    function formatTime(dateStr) {
        if (!dateStr) return '';
        const date = new Date(dateStr);
        return date.toLocaleTimeString();
    }

    function formatBytes(bytes) {
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

    const enabledPairs = $derived($folderPairs.filter(p => p.enabled));
    const isSyncing = $derived($syncStatus.status === 'syncing' || $syncStatus.status === 'scanning');
</script>

<div class="h-full flex flex-col p-5">
    <!-- Header -->
    <div class="flex justify-between items-center mb-4">
        <h2 class="text-2xl font-semibold">Sync Status</h2>
        <div class="flex gap-2">
            <button
                class="flex items-center gap-2 px-3 py-2 text-sm font-medium text-slate-100 bg-white/10 border border-white/20 rounded-lg hover:bg-white/15 transition-colors disabled:opacity-50"
                onclick={analyzeAllPairs}
                disabled={isAnalyzing || enabledPairs.length === 0}
            >
                <Search size={16} />
                {isAnalyzing ? 'Analyzing...' : 'Preview Changes'}
            </button>
            <button
                class="flex items-center gap-2 px-4 py-2 text-sm font-medium text-white bg-macos-blue rounded-lg hover:bg-blue-600 transition-colors disabled:opacity-50"
                onclick={triggerSync}
                disabled={isSyncing}
            >
                <RefreshCw size={16} class={isSyncing ? 'animate-spin' : ''} />
                Sync Now
            </button>
        </div>
    </div>

    <!-- Status card -->
    <div class="bg-white/5 border border-white/10 rounded-xl p-5 mb-4">
        <div class="flex items-center gap-4 mb-4">
            <div class="w-12 h-12 rounded-xl {getStatusColor($syncStatus.status)} flex items-center justify-center relative">
                {#if isSyncing}
                    <div class="absolute inset-0 rounded-xl {getStatusColor($syncStatus.status)} animate-ping opacity-50"></div>
                {/if}
            </div>
            <div>
                <span class="block text-xl font-semibold">{getStatusText($syncStatus.status)}</span>
                {#if $syncStatus.action}
                    <span class="text-sm text-slate-400">{$syncStatus.action}</span>
                {/if}
            </div>
        </div>

        {#if $progressData && (isSyncing || $overallPercentage > 0)}
            <div class="bg-black/20 rounded-lg p-4 mb-4">
                <div class="flex justify-between items-center mb-2">
                    <span class="text-sm font-medium">Syncing Files</span>
                    {#if $fileCountProgress}
                        <span class="text-xs px-2 py-0.5 bg-macos-blue/20 text-macos-blue rounded">{$fileCountProgress}</span>
                    {/if}
                </div>

                <div class="flex justify-between items-center mb-2">
                    <span class="text-2xl font-semibold">{Math.round($overallPercentage)}%</span>
                    <div class="flex gap-3 text-sm">
                        {#if $formattedSpeed}
                            <span class="text-macos-green font-medium">{$formattedSpeed}</span>
                        {/if}
                        {#if $formattedETA}
                            <span class="text-slate-400">{$formattedETA}</span>
                        {/if}
                    </div>
                </div>

                <div class="h-2 bg-white/10 rounded-full overflow-hidden">
                    <div class="h-full bg-gradient-to-r from-macos-blue to-blue-400 rounded-full transition-all duration-300" style="width: {$overallPercentage}%"></div>
                </div>

                {#if $activeFiles.length > 0}
                    <div class="mt-4 pt-4 border-t border-white/10">
                        <div class="flex justify-between items-center mb-3">
                            <span class="text-xs font-medium uppercase tracking-wider text-slate-500">Active Transfers</span>
                            <span class="text-xs px-2 py-0.5 bg-macos-blue/20 text-macos-blue rounded">{$activeFiles.length}</span>
                        </div>
                        <div class="space-y-2 max-h-48 overflow-y-auto">
                            {#each $activeFiles.slice(0, 10) as file}
                                <div class="bg-black/15 rounded-lg p-2.5">
                                    <div class="flex justify-between items-center mb-1.5">
                                        <span class="text-sm text-slate-200 truncate max-w-[60%]" title={file.path}>{getFileName(file.path)}</span>
                                        <span class="text-xs text-slate-500 font-mono">{formatBytes(file.bytesTransferred)} / {formatBytes(file.totalBytes)}</span>
                                    </div>
                                    <div class="flex items-center gap-2">
                                        <div class="flex-1 h-1 bg-white/10 rounded-full overflow-hidden">
                                            <div class="h-full bg-macos-blue rounded-full transition-all" style="width: {file.percentage}%"></div>
                                        </div>
                                        <span class="text-xs text-macos-blue font-medium min-w-[2rem] text-right">{Math.round(file.percentage)}%</span>
                                    </div>
                                </div>
                            {/each}
                        </div>
                    </div>
                {/if}
            </div>
        {/if}

        <div class="flex gap-4">
            <div class="flex-1 text-center p-3 bg-black/20 rounded-lg">
                <span class="block text-2xl font-semibold">{$folderPairs.filter(p => p.enabled).length}</span>
                <span class="text-xs text-slate-500">Active Pairs</span>
            </div>
            <div class="flex-1 text-center p-3 bg-black/20 rounded-lg">
                <span class="block text-2xl font-semibold">{$peers.filter(p => p.paired && p.status === 'online').length}</span>
                <span class="text-xs text-slate-500">Peers Online</span>
            </div>
            <div class="flex-1 text-center p-3 bg-black/20 rounded-lg">
                <span class="block text-2xl font-semibold">{$recentEvents.filter(e => e.type !== 'error').length}</span>
                <span class="text-xs text-slate-500">Files Synced</span>
            </div>
        </div>
    </div>

    <!-- Activity section -->
    <div class="flex-1 flex flex-col min-h-0">
        <h3 class="text-sm font-medium text-slate-400 mb-3">Recent Activity</h3>
        <div class="flex-1 overflow-y-auto">
            {#if $recentEvents.length === 0}
                <div class="text-center py-10 text-slate-500">
                    <p>No sync activity yet.</p>
                </div>
            {:else}
                {#each $recentEvents as event}
                    {@const Icon = getEventIcon(event.type)}
                    <div class="flex items-center gap-3 p-3 bg-white/[0.03] rounded-lg mb-2">
                        <div class="w-6 h-6 flex items-center justify-center {getEventColor(event.type)}">
                            <Icon size={16} />
                        </div>
                        <div class="flex-1 min-w-0">
                            <span class="block text-sm text-slate-100 truncate">{event.filePath || event.description}</span>
                            <span class="text-xs text-slate-500">{event.peerName || ''} {event.peerName ? '•' : ''} {formatTime(event.time)}</span>
                        </div>
                        <span class="text-xs font-medium uppercase {getEventColor(event.type)}">{event.type}</span>
                    </div>
                {/each}
            {/if}
        </div>
    </div>

    <!-- Preview modal -->
    {#if showPreview}
        <!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions a11y_interactive_supports_focus -->
        <div class="fixed inset-0 bg-black/70 backdrop-blur-sm flex items-center justify-center z-50" onclick={closePreview} role="dialog" aria-modal="true">
            <!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions a11y_no_noninteractive_element_interactions -->
            <div class="bg-slate-800 border border-white/10 rounded-xl p-6 w-full max-w-lg max-h-[80vh] overflow-y-auto shadow-2xl" onclick={(e) => e.stopPropagation()} role="document">
                <div class="flex justify-between items-center mb-5">
                    <h3 class="text-xl font-semibold">Sync Preview</h3>
                    <button class="text-slate-500 hover:text-slate-300" onclick={closePreview}>
                        <X size={24} />
                    </button>
                </div>

                {#if isAnalyzing}
                    <div class="text-center py-10">
                        <RefreshCw size={40} class="animate-spin text-macos-blue mx-auto mb-4" />
                        <p class="text-slate-400">Analyzing changes...</p>
                    </div>
                {:else if previewData?.error}
                    <div class="text-center py-10 text-macos-red">
                        <AlertCircle size={48} class="mx-auto mb-4" />
                        <p>{previewData.error}</p>
                    </div>
                {:else if previewData}
                    <div class="mb-5 p-4 bg-black/20 rounded-lg">
                        <p class="font-medium mb-2">{previewData.peerName}</p>
                        <div class="flex items-center gap-2 text-xs text-slate-400 font-mono">
                            <span class="truncate max-w-[40%]">{previewData.localPath}</span>
                            <ArrowLeftRight size={14} class="text-macos-blue flex-shrink-0" />
                            <span class="truncate max-w-[40%]">{previewData.remotePath}</span>
                        </div>
                    </div>

                    <div class="flex gap-3 mb-5">
                        <div class="flex-1 p-4 bg-black/20 rounded-lg">
                            <div class="flex items-center gap-3">
                                <ArrowUp size={24} class="text-macos-green" />
                                <div>
                                    <span class="block text-2xl font-semibold">{previewData.pushCount}</span>
                                    <span class="text-xs text-slate-500">Files to Upload</span>
                                    <span class="block text-xs text-slate-400">{formatBytes(previewData.pushSize)}</span>
                                </div>
                            </div>
                        </div>
                        <div class="flex-1 p-4 bg-black/20 rounded-lg">
                            <div class="flex items-center gap-3">
                                <ArrowDown size={24} class="text-macos-blue" />
                                <div>
                                    <span class="block text-2xl font-semibold">{previewData.pullCount}</span>
                                    <span class="text-xs text-slate-500">Files to Download</span>
                                    <span class="block text-xs text-slate-400">{formatBytes(previewData.pullSize)}</span>
                                </div>
                            </div>
                        </div>
                    </div>

                    {#if previewData.pushCount === 0 && previewData.pullCount === 0}
                        <div class="text-center py-10 text-macos-green">
                            <Check size={48} class="mx-auto mb-4" />
                            <p>Everything is in sync!</p>
                        </div>
                    {/if}
                {/if}

                <div class="flex justify-end gap-3 pt-4 border-t border-white/10">
                    <button
                        class="px-4 py-2 text-sm font-medium text-slate-300 bg-white/10 border border-white/20 rounded-lg hover:bg-white/15 transition-colors"
                        onclick={closePreview}
                    >
                        Cancel
                    </button>
                    {#if previewData && !previewData.error && (previewData.pushCount > 0 || previewData.pullCount > 0)}
                        <button
                            class="px-4 py-2 text-sm font-medium text-white bg-macos-blue rounded-lg hover:bg-blue-600 transition-colors"
                            onclick={syncFromPreview}
                        >
                            Start Sync
                        </button>
                    {/if}
                </div>
            </div>
        </div>
    {/if}
</div>
