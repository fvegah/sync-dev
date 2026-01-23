<script>
    import { onMount, onDestroy } from 'svelte';
    import { Monitor, RefreshCw, X, Search } from 'lucide-svelte';
    import { peers, pairingState, showModal } from '../stores/app.js';
    import { GetPeers, GeneratePairingCode, RequestPairing, UnpairPeer, RefreshPeers } from '../../wailsjs/go/main/App.js';
    import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime.js';

    let myPairingCode = $state('');
    let searchFilter = $state('');
    let isScanning = $state(false);
    let scanTimeout = $state(null);

    // Filter and sort peers: paired first, then by name
    const filteredPeers = $derived.by(() => {
        const peerList = $peers || [];
        return peerList
            .filter(p => {
                if (!searchFilter) return true;
                const search = searchFilter.toLowerCase();
                return p.name.toLowerCase().includes(search) ||
                       p.host.toLowerCase().includes(search);
            })
            .sort((a, b) => {
                // Paired devices first
                if (a.paired && !b.paired) return -1;
                if (!a.paired && b.paired) return 1;
                // Then by online status
                if (a.status === 'online' && b.status !== 'online') return -1;
                if (a.status !== 'online' && b.status === 'online') return 1;
                // Then by name
                return a.name.localeCompare(b.name);
            });
    });

    const pairedPeers = $derived(filteredPeers.filter(p => p.paired));
    const otherPeers = $derived(filteredPeers.filter(p => !p.paired));

    onMount(async () => {
        await loadPeers();
        EventsOn('peers:changed', loadPeers);
        // Start initial scan
        startScan();
    });

    onDestroy(() => {
        EventsOff('peers:changed');
        if (scanTimeout) clearTimeout(scanTimeout);
    });

    async function loadPeers() {
        const result = await GetPeers();
        peers.set(result || []);
    }

    async function startScan() {
        isScanning = true;
        await loadPeers();
        // Scan for 15 seconds
        scanTimeout = setTimeout(() => {
            isScanning = false;
        }, 15000);
    }

    async function refreshDevices() {
        if (scanTimeout) clearTimeout(scanTimeout);
        isScanning = true;
        try {
            await RefreshPeers();
        } catch (e) {
            // RefreshPeers may not exist, just reload
        }
        await loadPeers();
        scanTimeout = setTimeout(() => {
            isScanning = false;
        }, 15000);
    }

    async function generateCode() {
        myPairingCode = await GeneratePairingCode();
    }

    async function startPairing(peer) {
        showModal('pairing', peer);
    }

    async function unpair(peer) {
        if (confirm(`Unpair from ${peer.name}? This will remove all folder pairs with this device.`)) {
            await UnpairPeer(peer.id);
            await loadPeers();
        }
    }

    function getStatusColor(status) {
        switch (status) {
            case 'online': return 'bg-macos-green';
            case 'syncing': return 'bg-macos-blue';
            case 'pairing': return 'bg-macos-orange';
            default: return 'bg-slate-500';
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

    function clearSearch() {
        searchFilter = '';
    }

    function dismissCode() {
        myPairingCode = '';
    }
</script>

<div class="p-5 h-full flex flex-col">
    <!-- Header -->
    <div class="flex justify-between items-center mb-4">
        <h2 class="text-2xl font-semibold">Devices</h2>
        <div class="flex gap-2">
            <button
                class="flex items-center gap-1.5 px-4 py-2 bg-white/10 border border-white/20 rounded-lg text-sm text-slate-100 hover:bg-white/15 disabled:opacity-70 disabled:cursor-default transition-colors"
                onclick={refreshDevices}
                disabled={isScanning}
            >
                <RefreshCw size={16} class={isScanning ? 'animate-spin' : ''} />
                {isScanning ? 'Scanning...' : 'Refresh'}
            </button>
            <button
                class="flex items-center gap-1.5 px-4 py-2 bg-white/10 border border-white/20 rounded-lg text-sm text-slate-100 hover:bg-white/15 transition-colors"
                onclick={generateCode}
            >
                Generate Pairing Code
            </button>
        </div>
    </div>

    <!-- Search Bar -->
    <div class="flex items-center gap-2.5 bg-white/5 border border-white/10 rounded-lg px-3.5 py-2.5 mb-4">
        <Search size={18} class="text-slate-500 flex-shrink-0" />
        <input
            type="text"
            placeholder="Filter devices by name or IP..."
            bind:value={searchFilter}
            class="flex-1 bg-transparent border-none text-slate-100 text-sm outline-none placeholder-slate-500"
        />
        {#if searchFilter}
            <button
                class="text-slate-500 hover:text-slate-400 p-0 border-none bg-transparent cursor-pointer"
                onclick={clearSearch}
            >
                <X size={18} />
            </button>
        {/if}
    </div>

    <!-- Pairing Code Display -->
    {#if myPairingCode}
        <div class="bg-macos-blue/10 border border-macos-blue/30 rounded-lg p-4 mb-4 text-center">
            <p class="text-slate-400 mb-3">Share this code with another device to pair:</p>
            <div class="text-3xl font-mono font-bold tracking-[8px] text-macos-blue p-3 bg-black/20 rounded">
                {myPairingCode}
            </div>
            <button
                class="mt-3 text-sm text-slate-500 hover:text-slate-400 bg-transparent border-none cursor-pointer"
                onclick={dismissCode}
            >
                Dismiss
            </button>
        </div>
    {/if}

    <!-- Peers List -->
    <div class="flex-1 overflow-y-auto">
        {#if filteredPeers.length === 0}
            <div class="text-center py-10 text-slate-500">
                {#if searchFilter}
                    <p>No devices match your search.</p>
                {:else if isScanning}
                    <p>Scanning for devices...</p>
                    <p class="text-sm mt-2">Looking for other Macs running SyncDev on your network.</p>
                {:else}
                    <p>No devices found on your network.</p>
                    <p class="text-sm mt-2">Make sure SyncDev is running on other Macs in your local network.</p>
                {/if}
            </div>
        {:else}
            <!-- Paired Devices Section -->
            {#if pairedPeers.length > 0}
                <div class="flex items-center gap-2 mb-3 mt-2">
                    <span class="text-xs font-semibold uppercase text-slate-500 tracking-wide">Paired Devices</span>
                    <span class="text-[0.7rem] bg-white/10 text-slate-400 px-1.5 py-0.5 rounded-full">{pairedPeers.length}</span>
                </div>
                {#each pairedPeers as peer}
                    <div class="flex items-center gap-4 p-3.5 bg-macos-green/5 rounded-lg mb-2 border border-macos-green/30 hover:bg-macos-green/10 transition-colors">
                        <div class="flex items-center gap-3 flex-1">
                            <div class="w-10 h-10 flex items-center justify-center bg-macos-green/20 rounded-lg">
                                <Monitor size={24} class="text-macos-green" />
                            </div>
                            <div class="flex flex-col">
                                <span class="font-medium text-slate-100">{peer.name}</span>
                                <span class="text-xs text-slate-500 font-mono">{peer.host}:{peer.port}</span>
                            </div>
                        </div>
                        <div class="flex items-center gap-2">
                            <span class={`w-2 h-2 rounded-full ${getStatusColor(peer.status)}`}></span>
                            <span class="text-sm text-slate-400">{getStatusText(peer.status)}</span>
                        </div>
                        <div class="flex items-center gap-2">
                            <span class="text-xs px-2 py-1 bg-macos-green/20 text-macos-green rounded">Paired</span>
                            <button
                                class="p-1.5 rounded hover:bg-white/10 transition-colors group"
                                onclick={() => unpair(peer)}
                                title="Unpair"
                            >
                                <X size={16} class="text-slate-500 group-hover:text-red-500" />
                            </button>
                        </div>
                    </div>
                {/each}
            {/if}

            <!-- Available Devices Section -->
            {#if otherPeers.length > 0}
                <div class="flex items-center gap-2 mb-3 mt-2">
                    <span class="text-xs font-semibold uppercase text-slate-500 tracking-wide">Available Devices</span>
                    <span class="text-[0.7rem] bg-white/10 text-slate-400 px-1.5 py-0.5 rounded-full">{otherPeers.length}</span>
                </div>
                {#each otherPeers as peer}
                    <div class="flex items-center gap-4 p-3.5 bg-white/5 rounded-lg mb-2 border border-transparent hover:bg-white/8 transition-colors">
                        <div class="flex items-center gap-3 flex-1">
                            <div class="w-10 h-10 flex items-center justify-center bg-white/10 rounded-lg">
                                <Monitor size={24} class="text-slate-400" />
                            </div>
                            <div class="flex flex-col">
                                <span class="font-medium text-slate-100">{peer.name}</span>
                                <span class="text-xs text-slate-500 font-mono">{peer.host}:{peer.port}</span>
                            </div>
                        </div>
                        <div class="flex items-center gap-2">
                            <span class={`w-2 h-2 rounded-full ${getStatusColor(peer.status)}`}></span>
                            <span class="text-sm text-slate-400">{getStatusText(peer.status)}</span>
                        </div>
                        <div class="flex items-center gap-2">
                            {#if peer.status === 'online'}
                                <button
                                    class="px-3 py-1.5 bg-macos-blue text-white text-xs font-medium rounded-md hover:bg-blue-600 transition-colors"
                                    onclick={() => startPairing(peer)}
                                >
                                    Pair
                                </button>
                            {/if}
                        </div>
                    </div>
                {/each}
            {/if}
        {/if}
    </div>
</div>
