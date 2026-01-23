import { vitePreprocess } from '@sveltejs/vite-plugin-svelte'

export default {
  preprocess: vitePreprocess()
  // Note: runes mode is left at default (undefined) for compatibility mode.
  // This allows Svelte 5 to accept both legacy ($:) and runes ($derived) syntax.
  // Will be set to 'true' after component migration in later plans.
}
