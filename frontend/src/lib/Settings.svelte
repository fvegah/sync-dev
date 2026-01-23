<script>
    import { onMount } from 'svelte';
    import { ExternalLink } from 'lucide-svelte';
    import { config } from '../stores/app.js';
    import {
        GetConfig,
        UpdateDeviceName,
        UpdateSyncInterval,
        UpdateAutoSync,
        UpdateGlobalExclusions,
        GetAppVersion,
        GetDataDirectory,
        OpenDataDirectory
    } from '../../bindings/SyncDev/app.js';

    let deviceName = $state('');
    let syncInterval = $state(5);
    let autoSync = $state(true);
    let exclusionsText = $state('');
    let appVersion = $state('');
    let dataDir = $state('');
    let saved = $state(false);
    let saving = $state(false);

    onMount(async () => {
        const [cfg, version, dir] = await Promise.all([
            GetConfig(),
            GetAppVersion(),
            GetDataDirectory()
        ]);

        config.set(cfg);
        deviceName = cfg.deviceName;
        syncInterval = cfg.syncIntervalMins;
        autoSync = cfg.autoSync;
        exclusionsText = (cfg.globalExclusions || []).join('\n');
        appVersion = version;
        dataDir = dir;
    });

    async function saveSettings() {
        saving = true;
        try {
            await UpdateDeviceName(deviceName);
            await UpdateSyncInterval(syncInterval);
            await UpdateAutoSync(autoSync);

            const exclusions = exclusionsText
                .split('\n')
                .map(s => s.trim())
                .filter(s => s.length > 0);
            await UpdateGlobalExclusions(exclusions);

            saved = true;
            setTimeout(() => saved = false, 2000);
        } catch (err) {
            alert('Failed to save settings: ' + err);
        }
        saving = false;
    }

    async function openDataDir() {
        await OpenDataDirectory();
    }
</script>

<div class="h-full flex flex-col p-5">
    <!-- Header -->
    <div class="flex justify-between items-center mb-4">
        <h2 class="text-2xl font-semibold">Settings</h2>
        <button
            class="px-4 py-2 text-sm font-medium text-white bg-macos-blue rounded-lg hover:bg-blue-600 transition-colors disabled:opacity-50 min-w-[120px]"
            onclick={saveSettings}
            disabled={saving}
        >
            {#if saving}
                Saving...
            {:else if saved}
                Saved!
            {:else}
                Save Changes
            {/if}
        </button>
    </div>

    <!-- Settings content -->
    <div class="flex-1 overflow-y-auto space-y-4">
        <!-- Device section -->
        <section class="bg-white/5 border border-white/10 rounded-xl p-5">
            <h3 class="text-sm font-semibold text-slate-400 uppercase tracking-wider mb-4">Device</h3>
            <div>
                <label for="device-name" class="block text-sm font-medium text-slate-200 mb-2">Device Name</label>
                <input
                    id="device-name"
                    type="text"
                    bind:value={deviceName}
                    placeholder="My Mac"
                    class="w-full px-3 py-2.5 bg-black/30 border border-white/10 rounded-lg text-slate-100 text-sm focus:outline-none focus:border-macos-blue"
                />
                <p class="text-xs text-slate-500 mt-2">This name will be shown to other devices on your network.</p>
            </div>
        </section>

        <!-- Synchronization section -->
        <section class="bg-white/5 border border-white/10 rounded-xl p-5">
            <h3 class="text-sm font-semibold text-slate-400 uppercase tracking-wider mb-4">Synchronization</h3>

            <div class="mb-5">
                <label for="sync-interval" class="block text-sm font-medium text-slate-200 mb-2">Sync Interval</label>
                <div class="flex items-center gap-4">
                    <input
                        id="sync-interval"
                        type="range"
                        min="1"
                        max="60"
                        bind:value={syncInterval}
                        class="flex-1 h-1.5 bg-white/10 rounded-full appearance-none cursor-pointer [&::-webkit-slider-thumb]:appearance-none [&::-webkit-slider-thumb]:w-4 [&::-webkit-slider-thumb]:h-4 [&::-webkit-slider-thumb]:bg-macos-blue [&::-webkit-slider-thumb]:rounded-full [&::-webkit-slider-thumb]:cursor-pointer"
                    />
                    <span class="min-w-[4rem] text-right font-medium text-macos-blue">{syncInterval} min</span>
                </div>
                <p class="text-xs text-slate-500 mt-2">How often to automatically check for changes.</p>
            </div>

            <div>
                <label class="flex items-center gap-3 cursor-pointer">
                    <input
                        type="checkbox"
                        bind:checked={autoSync}
                        class="w-5 h-5 rounded border-white/20 bg-black/30 text-macos-blue focus:ring-macos-blue cursor-pointer"
                    />
                    <span class="text-sm font-medium text-slate-200">Enable automatic sync</span>
                </label>
                <p class="text-xs text-slate-500 mt-2 ml-8">When disabled, you'll need to manually trigger syncs.</p>
            </div>
        </section>

        <!-- Exclusions section -->
        <section class="bg-white/5 border border-white/10 rounded-xl p-5">
            <h3 class="text-sm font-semibold text-slate-400 uppercase tracking-wider mb-4">Exclusions</h3>
            <div>
                <label for="exclusions" class="block text-sm font-medium text-slate-200 mb-2">Global Exclusion Patterns</label>
                <textarea
                    id="exclusions"
                    bind:value={exclusionsText}
                    placeholder=".DS_Store
.git
node_modules
*.tmp"
                    rows="8"
                    class="w-full px-3 py-2.5 bg-black/30 border border-white/10 rounded-lg text-slate-100 text-sm font-mono resize-y focus:outline-none focus:border-macos-blue"
                ></textarea>
                <p class="text-xs text-slate-500 mt-2">
                    Files and folders matching these patterns will be ignored during sync. Use glob patterns, one per line.
                </p>
            </div>
        </section>

        <!-- About section -->
        <section class="bg-white/5 border border-white/10 rounded-xl p-5">
            <h3 class="text-sm font-semibold text-slate-400 uppercase tracking-wider mb-4">About</h3>
            <div class="space-y-3">
                <div class="flex justify-between items-center py-2 border-b border-white/5">
                    <span class="text-sm text-slate-400">Version</span>
                    <span class="text-sm text-slate-100">{appVersion}</span>
                </div>
                <div class="flex justify-between items-center py-2 border-b border-white/5">
                    <span class="text-sm text-slate-400">Data Directory</span>
                    <button
                        class="flex items-center gap-2 text-sm text-macos-blue hover:text-blue-400 transition-colors"
                        onclick={openDataDir}
                    >
                        {dataDir}
                        <ExternalLink size={14} />
                    </button>
                </div>
                <div class="flex justify-between items-center py-2">
                    <span class="text-sm text-slate-400">Device ID</span>
                    <span class="text-xs text-slate-500 font-mono truncate max-w-[200px]">{$config.deviceId}</span>
                </div>
            </div>
        </section>
    </div>
</div>
