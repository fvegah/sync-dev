<script>
    import { Monitor, Folder, RefreshCw, Settings } from 'lucide-svelte';
    import { currentTab, modalState, closeModal, pairingState } from './stores/app.js';
    import PeerList from './lib/PeerList.svelte';
    import FolderPairs from './lib/FolderPairs.svelte';
    import SyncStatus from './lib/SyncStatus.svelte';
    import SettingsComponent from './lib/Settings.svelte';
    import { RequestPairing } from '../wailsjs/go/main/App.js';

    let pairingInputCode = $state('');

    const tabs = [
        { id: 'peers', label: 'Devices', icon: Monitor },
        { id: 'folders', label: 'Folders', icon: Folder },
        { id: 'sync', label: 'Sync', icon: RefreshCw },
        { id: 'settings', label: 'Settings', icon: Settings }
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

    function handleModalOverlayClick() {
        closeModal();
    }

    function handleModalContentClick(event) {
        event.stopPropagation();
    }
</script>

<main class="h-screen w-screen overflow-hidden bg-slate-900 dark:bg-slate-950 text-slate-100">
    <div class="flex h-full">
        <!-- Sidebar with macOS vibrancy -->
        <aside class="
            w-52 h-full flex flex-col
            bg-white/5 dark:bg-slate-900/30
            backdrop-blur-md backdrop-saturate-150
            border-r border-white/10 dark:border-slate-700/50
            pt-8
        " style="-webkit-app-region: drag;">
            <!-- App title -->
            <div class="flex items-center gap-3 px-5 mb-6" style="-webkit-app-region: no-drag;">
                <div class="w-8 h-8 flex items-center justify-center">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="w-7 h-7 text-macos-blue">
                        <path d="M21 16V8a2 2 0 00-1-1.73l-7-4a2 2 0 00-2 0l-7 4A2 2 0 003 8v8a2 2 0 001 1.73l7 4a2 2 0 002 0l7-4A2 2 0 0021 16z"/>
                        <polyline points="3.27 6.96 12 12.01 20.73 6.96"/>
                        <line x1="12" y1="22.08" x2="12" y2="12"/>
                    </svg>
                </div>
                <span class="text-lg font-semibold">SyncDev</span>
            </div>

            <!-- Navigation -->
            <nav class="flex-1 px-3" style="-webkit-app-region: no-drag;">
                {#each tabs as tab}
                    {@const Icon = tab.icon}
                    <button
                        class="
                            w-full flex items-center gap-3 px-3 py-2.5 mb-1 rounded-lg
                            text-sm font-medium transition-colors
                            {$currentTab === tab.id
                                ? 'bg-macos-blue/20 text-macos-blue'
                                : 'text-slate-400 hover:text-slate-100 hover:bg-white/5'
                            }
                        "
                        onclick={() => setTab(tab.id)}
                    >
                        <Icon size={18} strokeWidth={2} />
                        <span>{tab.label}</span>
                    </button>
                {/each}
            </nav>
        </aside>

        <!-- Main content -->
        <div class="flex-1 overflow-hidden bg-gradient-to-br from-slate-800 to-slate-900 dark:from-slate-900 dark:to-slate-950">
            {#if $currentTab === 'peers'}
                <PeerList />
            {:else if $currentTab === 'folders'}
                <FolderPairs />
            {:else if $currentTab === 'sync'}
                <SyncStatus />
            {:else if $currentTab === 'settings'}
                <SettingsComponent />
            {/if}
        </div>
    </div>

    <!-- Pairing Modal -->
    {#if $modalState.show && $modalState.type === 'pairing'}
        <div
            class="fixed inset-0 bg-black/70 backdrop-blur-sm flex items-center justify-center z-50"
            onclick={handleModalOverlayClick}
            role="dialog"
            aria-modal="true"
        >
            <div
                class="bg-slate-800 border border-white/10 rounded-xl p-6 w-full max-w-md shadow-2xl"
                onclick={handleModalContentClick}
            >
                <h3 class="text-xl font-semibold mb-3">Pair with {$modalState.data?.name}</h3>
                <p class="text-slate-400 text-sm mb-4">Enter the 6-digit code shown on the other device:</p>
                <input
                    type="text"
                    maxlength="6"
                    bind:value={pairingInputCode}
                    placeholder="000000"
                    class="
                        w-full p-4 bg-black/30 border border-white/10 rounded-lg
                        text-2xl text-center font-mono tracking-[0.5em]
                        text-slate-100 placeholder-slate-600
                        focus:outline-none focus:border-macos-blue
                    "
                />
                <div class="flex justify-end gap-3 mt-6">
                    <button
                        class="px-4 py-2 text-sm font-medium text-slate-300 bg-white/10 border border-white/20 rounded-lg hover:bg-white/15 transition-colors"
                        onclick={closeModal}
                    >
                        Cancel
                    </button>
                    <button
                        class="px-4 py-2 text-sm font-medium text-white bg-macos-blue rounded-lg hover:bg-blue-600 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                        onclick={submitPairing}
                        disabled={pairingInputCode.length !== 6}
                    >
                        Pair
                    </button>
                </div>
            </div>
        </div>
    {/if}
</main>
