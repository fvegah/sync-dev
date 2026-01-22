<script>
    import { onMount } from 'svelte';
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
    } from '../../wailsjs/go/main/App.js';

    let deviceName = '';
    let syncInterval = 5;
    let autoSync = true;
    let exclusionsText = '';
    let appVersion = '';
    let dataDir = '';
    let saved = false;
    let saving = false;

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

<div class="settings">
    <div class="header">
        <h2>Settings</h2>
        <button class="btn-primary" on:click={saveSettings} disabled={saving}>
            {#if saving}
                Saving...
            {:else if saved}
                Saved!
            {:else}
                Save Changes
            {/if}
        </button>
    </div>

    <div class="settings-content">
        <div class="settings-section">
            <h3>Device</h3>
            <div class="form-group">
                <label>Device Name</label>
                <input type="text" bind:value={deviceName} placeholder="My Mac" />
                <p class="hint">This name will be shown to other devices on your network.</p>
            </div>
        </div>

        <div class="settings-section">
            <h3>Synchronization</h3>
            <div class="form-group">
                <label>Sync Interval</label>
                <div class="interval-input">
                    <input type="range" min="1" max="60" bind:value={syncInterval} />
                    <span class="interval-value">{syncInterval} min</span>
                </div>
                <p class="hint">How often to automatically check for changes.</p>
            </div>

            <div class="form-group">
                <label class="toggle-label">
                    <input type="checkbox" bind:checked={autoSync} />
                    <span>Enable automatic sync</span>
                </label>
                <p class="hint">When disabled, you'll need to manually trigger syncs.</p>
            </div>
        </div>

        <div class="settings-section">
            <h3>Exclusions</h3>
            <div class="form-group">
                <label>Global Exclusion Patterns</label>
                <textarea
                    bind:value={exclusionsText}
                    placeholder=".DS_Store
.git
node_modules
*.tmp"
                    rows="8"
                ></textarea>
                <p class="hint">
                    Files and folders matching these patterns will be ignored during sync.
                    Use glob patterns, one per line.
                </p>
            </div>
        </div>

        <div class="settings-section">
            <h3>About</h3>
            <div class="about-info">
                <div class="info-row">
                    <span class="info-label">Version</span>
                    <span class="info-value">{appVersion}</span>
                </div>
                <div class="info-row">
                    <span class="info-label">Data Directory</span>
                    <span class="info-value clickable" on:click={openDataDir}>
                        {dataDir}
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <path d="M18 13v6a2 2 0 01-2 2H5a2 2 0 01-2-2V8a2 2 0 012-2h6"/>
                            <polyline points="15 3 21 3 21 9"/>
                            <line x1="10" y1="14" x2="21" y2="3"/>
                        </svg>
                    </span>
                </div>
                <div class="info-row">
                    <span class="info-label">Device ID</span>
                    <span class="info-value device-id">{$config.deviceId}</span>
                </div>
            </div>
        </div>
    </div>
</div>

<style>
    .settings {
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
        margin: 0 0 16px 0;
        font-size: 1rem;
        font-weight: 600;
        color: #94a3b8;
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
        transition: all 0.2s;
        min-width: 120px;
    }

    .btn-primary:hover:not(:disabled) {
        background: #2563eb;
    }

    .btn-primary:disabled {
        opacity: 0.7;
        cursor: not-allowed;
    }

    .settings-content {
        flex: 1;
        overflow-y: auto;
    }

    .settings-section {
        background: rgba(255, 255, 255, 0.05);
        border-radius: 12px;
        padding: 20px;
        margin-bottom: 16px;
    }

    .form-group {
        margin-bottom: 20px;
    }

    .form-group:last-child {
        margin-bottom: 0;
    }

    .form-group label {
        display: block;
        margin-bottom: 8px;
        font-size: 0.875rem;
        color: #f1f5f9;
        font-weight: 500;
    }

    .form-group input[type="text"],
    .form-group textarea {
        width: 100%;
        padding: 10px 12px;
        background: rgba(0, 0, 0, 0.3);
        border: 1px solid rgba(255, 255, 255, 0.1);
        border-radius: 6px;
        color: #f1f5f9;
        font-size: 0.875rem;
        font-family: inherit;
        resize: vertical;
    }

    .form-group input:focus,
    .form-group textarea:focus {
        outline: none;
        border-color: #3b82f6;
    }

    .hint {
        margin: 8px 0 0 0;
        font-size: 0.75rem;
        color: #64748b;
    }

    .interval-input {
        display: flex;
        align-items: center;
        gap: 16px;
    }

    .interval-input input[type="range"] {
        flex: 1;
        height: 6px;
        background: rgba(255, 255, 255, 0.1);
        border-radius: 3px;
        appearance: none;
        cursor: pointer;
    }

    .interval-input input[type="range"]::-webkit-slider-thumb {
        appearance: none;
        width: 18px;
        height: 18px;
        background: #3b82f6;
        border-radius: 50%;
        cursor: pointer;
    }

    .interval-value {
        min-width: 60px;
        text-align: right;
        font-weight: 500;
        color: #60a5fa;
    }

    .toggle-label {
        display: flex !important;
        align-items: center;
        gap: 12px;
        cursor: pointer;
    }

    .toggle-label input[type="checkbox"] {
        width: 20px;
        height: 20px;
        cursor: pointer;
    }

    .toggle-label span {
        font-weight: normal;
    }

    .about-info {
        display: flex;
        flex-direction: column;
        gap: 12px;
    }

    .info-row {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 8px 0;
        border-bottom: 1px solid rgba(255, 255, 255, 0.05);
    }

    .info-row:last-child {
        border-bottom: none;
    }

    .info-label {
        color: #64748b;
        font-size: 0.875rem;
    }

    .info-value {
        color: #f1f5f9;
        font-size: 0.875rem;
    }

    .info-value.clickable {
        display: flex;
        align-items: center;
        gap: 8px;
        color: #60a5fa;
        cursor: pointer;
        transition: color 0.2s;
    }

    .info-value.clickable:hover {
        color: #93c5fd;
    }

    .info-value.clickable svg {
        width: 14px;
        height: 14px;
    }

    .info-value.device-id {
        font-family: monospace;
        font-size: 0.75rem;
        color: #64748b;
        max-width: 200px;
        overflow: hidden;
        text-overflow: ellipsis;
    }
</style>
