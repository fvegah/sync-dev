<script>
    import { onMount } from 'svelte';
    import { Monitor, Folder, RefreshCw, Trash2, Search, ArrowLeftRight, Check, AlertCircle, ChevronUp, ChevronDown, X } from 'lucide-svelte';
    import { folderPairs, peers, showModal } from '../stores/app.js';
    import {
        GetFolderPairs,
        GetPeers,
        AddFolderPair,
        RemoveFolderPair,
        UpdateFolderPair,
        SelectFolder,
        SyncFolderPair,
        AnalyzeFolderPair
    } from '../../bindings/SyncDev/app.js';

    let showAddForm = $state(false);
    let selectedPeerId = $state('');
    let localPath = $state('');
    let remotePath = $state('');
    let analyzingPairId = $state(null);
    let showPreview = $state(false);
    let previewData = $state(null);

    const pairedPeers = $derived(($peers || []).filter(p => p.paired));

    onMount(async () => {
        await loadData();
    });

    async function loadData() {
        try {
            const [pairs, peerList] = await Promise.all([
                GetFolderPairs(),
                GetPeers()
            ]);
            console.log('FolderPairs loaded:', { pairs, peerList });
            folderPairs.set(pairs || []);
            peers.set(peerList || []);
        } catch (err) {
            console.error('Error loading folder pairs data:', err);
        }
    }

    function getPeerName(peerId) {
        const peerList = $peers || [];
        const peer = peerList.find(p => p.id === peerId);
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
        try {
            console.log('Removing folder pair:', id);
            await RemoveFolderPair(id);
            console.log('Folder pair removed, reloading...');
            await loadData();
        } catch (err) {
            console.error('Error removing folder pair:', err);
            alert('Failed to remove: ' + err);
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

    async function analyzePair(pair) {
        analyzingPairId = pair.id;
        showPreview = true;
        previewData = null;

        try {
            previewData = await AnalyzeFolderPair(pair.id);
        } catch (err) {
            console.error('Analysis failed:', err);
            previewData = { error: err.toString() };
        }
        analyzingPairId = null;
    }

    async function syncFromPreview() {
        if (!previewData?.folderPairId) return;
        const pairId = previewData.folderPairId;
        closePreview();
        try {
            await SyncFolderPair(pairId);
        } catch (err) {
            alert('Sync failed: ' + err);
        }
    }

    function closePreview() {
        showPreview = false;
        previewData = null;
    }

    function toggleAddForm() {
        showAddForm = !showAddForm;
    }

    function handleOverlayClick() {
        closePreview();
    }

    function handleModalClick(event) {
        event.stopPropagation();
    }
</script>

<div class="p-5 h-full flex flex-col">
    <!-- Header -->
    <div class="flex justify-between items-center mb-5">
        <h2 class="text-2xl font-semibold">Folder Pairs</h2>
        {#if pairedPeers.length > 0}
            <button
                class="px-4 py-2 bg-macos-blue text-white font-medium text-sm rounded-md hover:bg-blue-600 transition-colors"
                onclick={toggleAddForm}
            >
                {showAddForm ? 'Cancel' : 'Add Folder Pair'}
            </button>
        {/if}
    </div>

    <!-- Add Form -->
    {#if showAddForm}
        <div class="bg-white/5 rounded-lg p-5 mb-5">
            <div class="mb-4">
                <label class="block mb-2 text-sm text-slate-400" for="peer-select">Paired Device</label>
                <select
                    id="peer-select"
                    bind:value={selectedPeerId}
                    class="w-full p-2.5 bg-black/30 border border-white/10 rounded-md text-slate-100 text-sm cursor-pointer"
                >
                    <option value="">Select a device...</option>
                    {#each pairedPeers as peer}
                        <option value={peer.id}>{peer.name}</option>
                    {/each}
                </select>
            </div>

            <div class="mb-4">
                <label class="block mb-2 text-sm text-slate-400" for="local-path">Local Folder</label>
                <div class="flex gap-2">
                    <input
                        id="local-path"
                        type="text"
                        bind:value={localPath}
                        placeholder="/Users/you/Documents/Projects"
                        class="flex-1 p-2.5 bg-black/30 border border-white/10 rounded-md text-slate-100 text-sm"
                    />
                    <button
                        class="px-4 py-2 bg-white/10 border border-white/20 rounded-md text-slate-100 text-sm hover:bg-white/15 transition-colors"
                        onclick={browseLocalFolder}
                    >
                        Browse
                    </button>
                </div>
            </div>

            <div class="mb-4">
                <label class="block mb-2 text-sm text-slate-400" for="remote-path">Remote Folder (on paired device)</label>
                <input
                    id="remote-path"
                    type="text"
                    bind:value={remotePath}
                    placeholder="/Users/other/Documents/Projects"
                    class="w-full p-2.5 bg-black/30 border border-white/10 rounded-md text-slate-100 text-sm"
                />
            </div>

            <div class="mt-5">
                <button
                    class="px-4 py-2 bg-macos-blue text-white font-medium text-sm rounded-md hover:bg-blue-600 transition-colors"
                    onclick={addPair}
                >
                    Add Folder Pair
                </button>
            </div>
        </div>
    {/if}

    <!-- Pairs List -->
    <div class="flex-1 overflow-y-auto">
        {#if ($folderPairs || []).length === 0}
            <div class="text-center py-10 text-slate-500">
                {#if pairedPeers.length === 0}
                    <p>No paired devices yet.</p>
                    <p class="text-sm mt-2">Go to the Devices tab to pair with another Mac first.</p>
                {:else}
                    <p>No folder pairs configured.</p>
                    <p class="text-sm mt-2">Click "Add Folder Pair" to start syncing folders.</p>
                {/if}
            </div>
        {:else}
            {#each $folderPairs as pair}
                <div class="bg-white/5 rounded-lg p-4 mb-3 border border-white/10 transition-all {!pair.enabled ? 'opacity-50' : ''}">
                    <!-- Pair Header -->
                    <div class="flex items-center gap-3 mb-3">
                        <div class="pair-toggle">
                            <input
                                type="checkbox"
                                checked={pair.enabled}
                                onchange={() => toggleEnabled(pair)}
                                class="w-4.5 h-4.5 cursor-pointer accent-macos-blue"
                            />
                        </div>
                        <div class="flex-1">
                            <div class="flex items-center gap-2 font-medium">
                                <Monitor size={18} class="text-slate-500" />
                                {getPeerName(pair.peerId)}
                            </div>
                        </div>
                        <div class="flex gap-1">
                            <button
                                class="p-1.5 rounded hover:bg-white/10 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                                onclick={() => analyzePair(pair)}
                                title="Preview changes"
                                disabled={!pair.enabled || analyzingPairId === pair.id}
                            >
                                <Search size={18} class="text-slate-500 hover:text-slate-300" />
                            </button>
                            <button
                                class="p-1.5 rounded hover:bg-white/10 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                                onclick={() => syncNow(pair)}
                                title="Sync now"
                                disabled={!pair.enabled}
                            >
                                <RefreshCw size={18} class="text-slate-500 hover:text-slate-300" />
                            </button>
                            <button
                                class="p-1.5 rounded hover:bg-white/10 transition-colors group"
                                onclick={() => removePair(pair.id)}
                                title="Remove"
                            >
                                <Trash2 size={18} class="text-slate-500 group-hover:text-red-500" />
                            </button>
                        </div>
                    </div>

                    <!-- Paths -->
                    <div class="flex items-center gap-3 p-3 bg-black/20 rounded-md mb-3">
                        <div class="flex-1 min-w-0">
                            <span class="block text-xs text-slate-500 mb-1">Local:</span>
                            <span class="block text-sm text-slate-400 font-mono break-all">{pair.localPath}</span>
                        </div>
                        <div class="px-2 flex-shrink-0">
                            <ArrowLeftRight size={20} class="text-macos-blue" />
                        </div>
                        <div class="flex-1 min-w-0">
                            <span class="block text-xs text-slate-500 mb-1">Remote:</span>
                            <span class="block text-sm text-slate-400 font-mono break-all">{pair.remotePath}</span>
                        </div>
                    </div>

                    <!-- Footer -->
                    <div class="text-xs text-slate-500">
                        Last sync: {formatDate(pair.lastSyncTime)}
                    </div>
                </div>
            {/each}
        {/if}
    </div>

    <!-- Preview Modal -->
    {#if showPreview}
        <!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions a11y_interactive_supports_focus -->
        <div
            class="fixed inset-0 bg-black/70 backdrop-blur-sm flex items-center justify-center z-50"
            onclick={handleOverlayClick}
            role="dialog"
            aria-modal="true"
        >
            <!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions a11y_no_noninteractive_element_interactions -->
            <div
                class="bg-slate-800 rounded-xl p-6 min-w-[450px] max-w-[550px] max-h-[80vh] overflow-y-auto border border-white/10"
                onclick={handleModalClick}
                role="document"
            >
                <!-- Modal Header -->
                <div class="flex justify-between items-center mb-5">
                    <h3 class="text-xl font-semibold text-slate-100">Sync Preview</h3>
                    <button
                        class="text-slate-500 hover:text-slate-300 text-2xl leading-none bg-transparent border-none cursor-pointer"
                        onclick={closePreview}
                    >
                        <X size={24} />
                    </button>
                </div>

                {#if analyzingPairId}
                    <div class="text-center py-10">
                        <div class="w-10 h-10 border-3 border-white/10 border-t-macos-blue rounded-full animate-spin mx-auto mb-4"></div>
                        <p class="text-slate-400">Analyzing changes...</p>
                    </div>
                {:else if previewData?.error}
                    <div class="text-center py-10 text-red-500">
                        <AlertCircle size={48} class="mx-auto mb-4" />
                        <p>{previewData.error}</p>
                    </div>
                {:else if previewData}
                    <div class="mb-5">
                        <!-- Preview Info -->
                        <div class="bg-black/20 rounded-lg p-4 mb-4">
                            <div class="mb-2 text-slate-100 font-medium">{previewData.peerName}</div>
                            <div class="flex items-center gap-2 text-sm">
                                <span class="text-slate-400 font-mono truncate max-w-[180px]">{previewData.localPath}</span>
                                <span class="text-macos-blue">-</span>
                                <span class="text-slate-400 font-mono truncate max-w-[180px]">{previewData.remotePath}</span>
                            </div>
                        </div>

                        <!-- Summary -->
                        <div class="flex gap-3 mb-4">
                            <div class="flex-1 flex items-center gap-3 p-3 bg-black/20 rounded-lg">
                                <ChevronUp size={24} class="text-macos-green" />
                                <div class="flex flex-col">
                                    <span class="text-xl font-semibold text-slate-100">{previewData.pushCount}</span>
                                    <span class="text-xs text-slate-500">Upload</span>
                                    <span class="text-xs text-slate-400">{formatBytes(previewData.pushSize)}</span>
                                </div>
                            </div>
                            <div class="flex-1 flex items-center gap-3 p-3 bg-black/20 rounded-lg">
                                <ChevronDown size={24} class="text-macos-blue" />
                                <div class="flex flex-col">
                                    <span class="text-xl font-semibold text-slate-100">{previewData.pullCount}</span>
                                    <span class="text-xs text-slate-500">Download</span>
                                    <span class="text-xs text-slate-400">{formatBytes(previewData.pullSize)}</span>
                                </div>
                            </div>
                        </div>

                        <!-- Files to Push -->
                        {#if previewData.toPush?.length > 0}
                            <div class="mb-3">
                                <h4 class="text-sm text-slate-400 mb-2">Files to Upload ({previewData.pushCount})</h4>
                                <ul class="list-none p-0 m-0 bg-black/20 rounded-lg max-h-[120px] overflow-y-auto">
                                    {#each previewData.toPush.slice(0, 8) as file}
                                        <li class="px-3 py-1.5 text-xs font-mono text-slate-400 border-b border-white/5 last:border-b-0">{file.path}</li>
                                    {/each}
                                    {#if previewData.toPush.length > 8}
                                        <li class="px-3 py-1.5 text-xs text-slate-500 italic">...and {previewData.toPush.length - 8} more</li>
                                    {/if}
                                </ul>
                            </div>
                        {/if}

                        <!-- Files to Pull -->
                        {#if previewData.toPull?.length > 0}
                            <div class="mb-3">
                                <h4 class="text-sm text-slate-400 mb-2">Files to Download ({previewData.pullCount})</h4>
                                <ul class="list-none p-0 m-0 bg-black/20 rounded-lg max-h-[120px] overflow-y-auto">
                                    {#each previewData.toPull.slice(0, 8) as file}
                                        <li class="px-3 py-1.5 text-xs font-mono text-slate-400 border-b border-white/5 last:border-b-0">{file.path}</li>
                                    {/each}
                                    {#if previewData.toPull.length > 8}
                                        <li class="px-3 py-1.5 text-xs text-slate-500 italic">...and {previewData.toPull.length - 8} more</li>
                                    {/if}
                                </ul>
                            </div>
                        {/if}

                        <!-- Everything in sync -->
                        {#if previewData.pushCount === 0 && previewData.pullCount === 0}
                            <div class="text-center py-8 text-macos-green">
                                <Check size={40} class="mx-auto mb-3" />
                                <p>Everything is in sync!</p>
                            </div>
                        {/if}
                    </div>
                {/if}

                <!-- Modal Actions -->
                <div class="flex justify-end gap-3 pt-4 border-t border-white/10">
                    <button
                        class="px-4 py-2 bg-white/10 border border-white/20 rounded-md text-slate-100 text-sm hover:bg-white/15 transition-colors"
                        onclick={closePreview}
                    >
                        Cancel
                    </button>
                    {#if previewData && !previewData.error && (previewData.pushCount > 0 || previewData.pullCount > 0)}
                        <button
                            class="px-4 py-2 bg-macos-blue text-white font-medium text-sm rounded-md hover:bg-blue-600 transition-colors"
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
