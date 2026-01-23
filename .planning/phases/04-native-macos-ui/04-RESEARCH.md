# Phase 4: Native macOS UI - Research

**Researched:** 2026-01-23
**Domain:** macOS-native web UI with Svelte 5, shadcn-svelte, and Tailwind CSS
**Confidence:** MEDIUM

## Summary

This research investigates building a native-looking macOS UI using modern web technologies: Svelte 5 with runes, shadcn-svelte component library, and Tailwind CSS for styling. The phase involves a significant migration from Svelte 3 to Svelte 5, which requires intermediate migration to Svelte 4 and complete reactivity system rewrite using runes ($state, $derived, $effect).

The standard approach combines shadcn-svelte (v1.0.9+) for accessible UI components with Tailwind CSS v4 for styling, using system font stacks for San Francisco font integration and CSS-based dark mode detection. SF Symbols cannot be used due to licensing restrictions for web apps, so alternatives like Lucide icons (1500+ icons, tree-shakable) or Heroicons (by Tailwind Labs) are recommended.

Key architectural changes include moving from implicit reactivity (let, $:) to explicit runes, replacing slots with snippets, and converting event dispatchers to callback props. The migration script (npx sv migrate svelte-5) automates many transformations but requires manual intervention for event dispatchers, lifecycle methods, and performance optimization.

**Primary recommendation:** Migrate Svelte 3 → 4 → 5 incrementally, use shadcn-svelte for component foundation, implement macOS visual patterns with Tailwind CSS backdrop blur and system colors, avoid SF Symbols in favor of Lucide icons, and minimize $effect() usage (prefer $derived for 90% of reactive code).

## Standard Stack

The established libraries/tools for building native-looking macOS UIs with Svelte:

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Svelte | 5.46.0 | Reactive UI framework | Stable release (Oct 2024), runes system provides better performance and reactivity |
| shadcn-svelte | 1.0.9+ | Component library | Official Svelte port of shadcn/ui, Svelte 5 compatible, accessible components |
| Tailwind CSS | 4.1.18 | Utility-first CSS | v4 offers 5x faster builds, CSS-first configuration, native support for modern features |
| @sveltejs/vite-plugin-svelte | ^5.0.0 | Vite integration | Official Svelte plugin for Vite bundler |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| bits-ui | Latest (Svelte 5) | Headless UI primitives | Automatically installed by shadcn-svelte for component logic |
| lucide-svelte | 0.562.0 | Icon library | 1500+ SVG icons, tree-shakable, replaces SF Symbols |
| @sveltejs/adapter-static | Latest | SvelteKit adapter | Required for Wails v3 - disables SSR, enables prerendering |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| lucide-svelte | Heroicons | Heroicons by Tailwind Labs, fewer icons (~300), good Tailwind integration |
| shadcn-svelte | Custom components | More control but lose accessibility, testing, maintenance from shadcn |
| Tailwind v4 | Tailwind v3 | v3 more stable but 5x slower builds, lacks modern CSS features |

**Installation:**
```bash
# Upgrade to Svelte 5 (requires intermediate migration to Svelte 4 if on Svelte 3)
npm install svelte@latest @sveltejs/vite-plugin-svelte@latest

# Run migration script
npx sv migrate svelte-5

# Initialize shadcn-svelte
npm dlx shadcn-svelte@latest init

# Install Tailwind CSS v4 (if not already installed)
npm install tailwindcss@latest

# Install icon library
npm install lucide-svelte

# Install SvelteKit static adapter for Wails
npm install -D @sveltejs/adapter-static
```

## Architecture Patterns

### Recommended Project Structure
```
frontend/
├── src/
│   ├── lib/
│   │   ├── components/
│   │   │   ├── ui/              # shadcn-svelte components
│   │   │   │   ├── button/
│   │   │   │   ├── card/
│   │   │   │   └── ...
│   │   │   ├── PeerList.svelte  # Feature components
│   │   │   ├── FolderPairs.svelte
│   │   │   └── SyncStatus.svelte
│   │   ├── stores/              # Svelte stores (if needed)
│   │   ├── utils/               # Utility functions
│   │   └── wailsjs/             # Wails Go bindings
│   ├── routes/                  # SvelteKit routes (minimal for desktop)
│   │   ├── +layout.svelte       # Root layout
│   │   └── +page.svelte         # Main page
│   ├── app.css                  # Global styles, Tailwind imports
│   └── app.html                 # HTML template
├── static/                      # Static assets
├── svelte.config.js
├── tailwind.config.js
├── vite.config.js
└── package.json
```

### Pattern 1: Svelte 5 Runes for Reactivity
**What:** Replace Svelte 3/4 reactivity with explicit runes
**When to use:** All reactive state management in Svelte 5

**Example:**
```typescript
// Svelte 3 (OLD)
let count = 0;
$: doubled = count * 2;
$: {
  console.log('Count changed:', count);
}

// Svelte 5 (NEW)
let count = $state(0);
const doubled = $derived(count * 2);
$effect(() => {
  console.log('Count changed:', count);
});
```
**Source:** [Svelte 5 Runes Documentation](https://svelte.dev/docs/svelte/$state)

### Pattern 2: Component Props with $props Rune
**What:** Replace export let with $props rune
**When to use:** All component prop declarations

**Example:**
```typescript
// Svelte 3 (OLD)
export let title;
export let count = 0;

// Svelte 5 (NEW)
let { title, count = 0 } = $props();
```

### Pattern 3: Callback Props Instead of Event Dispatchers
**What:** Replace createEventDispatcher with callback props
**When to use:** All component events

**Example:**
```typescript
// Svelte 3 (OLD)
import { createEventDispatcher } from 'svelte';
const dispatch = createEventDispatcher();
function handleClick() {
  dispatch('save', { data });
}

// Svelte 5 (NEW)
let { onSave } = $props();
function handleClick() {
  onSave?.({ data });
}
```

### Pattern 4: macOS Vibrancy with Tailwind Backdrop Blur
**What:** Create translucent, frosted-glass sidebars like macOS Finder
**When to use:** Sidebar, modal overlays, floating panels

**Example:**
```svelte
<!-- Translucent sidebar with macOS-style blur -->
<aside class="
  fixed left-0 top-0 h-full w-64
  bg-white/30 dark:bg-gray-900/30
  backdrop-blur-md
  border-r border-gray-200/50 dark:border-gray-700/50
">
  <nav><!-- Navigation items --></nav>
</aside>
```
**Source:** [Tailwind CSS Backdrop Blur](https://tailwindcss.com/docs/backdrop-blur)

### Pattern 5: System Font Stack for San Francisco
**What:** Use CSS font-family stack to access San Francisco on macOS
**When to use:** All text elements (global style)

**Example:**
```css
/* app.css */
body {
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto,
               "Helvetica Neue", Arial, sans-serif;
}
```
**Source:** [Using the System Font in Web Content](https://webkit.org/blog/3709/using-the-system-font-in-web-content/)

### Pattern 6: Dark Mode with prefers-color-scheme
**What:** Detect and respond to macOS system theme
**When to use:** All color styling

**Example:**
```css
/* Tailwind config approach */
/* tailwind.config.js */
export default {
  darkMode: 'media', // Uses prefers-color-scheme
  theme: {
    extend: {
      colors: {
        'macos-blue': '#007AFF',
        'macos-gray': '#8E8E93',
      }
    }
  }
}
```

```svelte
<!-- Component usage -->
<button class="
  bg-blue-500 dark:bg-macos-blue
  text-white
  hover:bg-blue-600 dark:hover:bg-blue-600
">
  Action
</button>
```
**Source:** [Dark Mode Support in WebKit](https://webkit.org/blog/8840/dark-mode-support-in-webkit/)

### Pattern 7: Wails v3 Static Adapter Configuration
**What:** Configure SvelteKit for desktop app (no SSR)
**When to use:** All Wails v3 projects with SvelteKit

**Example:**
```javascript
// svelte.config.js
import adapter from '@sveltejs/adapter-static';

/** @type {import('@sveltejs/kit').Config} */
const config = {
  kit: {
    adapter: adapter({
      pages: 'build',
      assets: 'build',
      fallback: undefined,
      precompress: false,
      strict: true
    }),
    alias: {
      $lib: './src/lib',
      '@/*': './src/lib/*'
    }
  }
};

export default config;
```

```svelte
<!-- src/routes/+layout.svelte -->
<script>
  export const prerender = true;
  export const ssr = false;
</script>

<slot />
```
**Source:** [Wails SvelteKit Guide](https://wails.io/docs/guides/sveltekit/)

### Anti-Patterns to Avoid

- **Using $effect() for derived state:** 90% of reactive code should use $derived, not $effect. Only use $effect for true side effects (DOM manipulation, logging, analytics).
- **Destructuring reactive state:** `let { done } = todos[0]; done = !done` won't update the original. Use object references or reassignment.
- **Hardcoded colors:** Use Tailwind's dark: variant and system color semantics instead of fixed colors.
- **Multiple handlers on same element:** Svelte 5 doesn't allow multiple `onclick` handlers. Combine logic into one function.
- **Importing SF Symbols directly:** Licensing prohibits web use. Use Lucide or Heroicons instead.

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Accessible dropdowns, dialogs, tooltips | Custom click handlers and z-index | shadcn-svelte components | ARIA attributes, keyboard navigation, focus management, screen reader support |
| Icon library | SVG imports or icon fonts | lucide-svelte | 1500+ icons, tree-shakable, consistent style, maintained |
| Dark mode detection | Manual localStorage + JS | Tailwind dark: with prefers-color-scheme | Respects system preferences, no flash of wrong theme |
| Component styling | Custom CSS classes | Tailwind utility classes | Consistency, design tokens, responsive, dark mode built-in |
| Reactivity system | Manual DOM updates | Svelte 5 runes | Compiler optimization, automatic tracking, better performance |
| macOS blur effects | CSS blur() filter | Tailwind backdrop-blur-* | Proper backdrop filtering (blurs behind element, not element itself) |

**Key insight:** Svelte 5's rune system handles reactivity at compile time with automatic dependency tracking. Custom reactivity solutions can't match compiler-level optimization and will have edge cases around timing, batching, and circular dependencies.

## Common Pitfalls

### Pitfall 1: Running Full Auto-Migration Script
**What goes wrong:** Performance drops significantly, app behavior changes unexpectedly
**Why it happens:** Migration script conservatively converts many $: statements to $effect() when $derived() would be more appropriate
**How to avoid:** Migrate file-by-file using VS Code command "Migrate Component to Svelte 5 Syntax" and manually review each conversion
**Warning signs:** App feels slower after migration, unnecessary re-renders, $effect() in most components

### Pitfall 2: Skipping Svelte 4 Migration
**What goes wrong:** Direct Svelte 3 → 5 migration may fail or produce incorrect code
**Why it happens:** Migration tooling expects Svelte 4 as baseline, Svelte 3 has different syntax patterns
**How to avoid:** Migrate Svelte 3 → 4 first, verify app works, then migrate 4 → 5
**Warning signs:** Migration script errors, components not rendering, TypeScript errors

### Pitfall 3: Using $effect() for State Updates
**What goes wrong:** Different behavior on client vs server, infinite loops, stale state
**Why it happens:** $effect() runs after render, creating timing issues when updating state
**How to avoid:** Use $derived for computed state (90% of cases), only use $effect for true side effects (DOM, logging, external APIs)
**Warning signs:** Hydration mismatches, state updates not reflecting, "maximum call stack" errors

### Pitfall 4: Installing shadcn Components Blindly
**What goes wrong:** Dependency auto-upgrades break app, incompatible bits-ui versions
**Why it happens:** CLI automatically upgrades bits-ui to latest, which may be incompatible
**How to avoid:** Check current Svelte version before running add commands, review package.json after adding components, pin bits-ui version if needed
**Warning signs:** Build errors after adding components, "peer dependency" warnings, components not rendering

### Pitfall 5: Hardcoding macOS Visual Elements
**What goes wrong:** App looks wrong in light mode or on non-macOS systems
**Why it happens:** Fixed colors/styles instead of using system-aware approaches
**How to avoid:** Use Tailwind's dark: variant, prefers-color-scheme, system font stack, semantic color variables
**Warning signs:** White text on white background in light mode, wrong fonts on Windows, no dark mode support

### Pitfall 6: SF Symbols Licensing Violation
**What goes wrong:** Legal issues, icons don't work on non-Apple devices
**Why it happens:** SF Symbols are only licensed for Apple platform native apps, not web
**How to avoid:** Use web-compatible icon libraries (Lucide, Heroicons) from the start
**Warning signs:** Icons only visible on macOS Safari, font loading errors, Unicode fallbacks

### Pitfall 7: SVG Icon Rendering Failure After Migration
**What goes wrong:** SVG-based icons fail to render in migrated Svelte 5 components
**Why it happens:** SVG namespace bug in migration script
**How to avoid:** Add `<svelte:options namespace='svg'>` to SVG components after migration
**Warning signs:** Icons disappear after migration, empty elements in DevTools

### Pitfall 8: createEventDispatcher Not Auto-Migrated
**What goes wrong:** Build errors about createEventDispatcher after migration
**Why it happens:** Migration script cannot automatically convert event dispatchers
**How to avoid:** Manually find all createEventDispatcher usage and convert to callback props before running migration
**Warning signs:** "createEventDispatcher is deprecated" warnings, type errors on dispatch calls

## Code Examples

Verified patterns from official sources:

### Complete Component Migration Example
```svelte
<!-- BEFORE: Svelte 3 Component -->
<script>
  import { createEventDispatcher } from 'svelte';

  export let count = 0;
  export let title;

  const dispatch = createEventDispatcher();

  let doubled;
  $: doubled = count * 2;

  $: {
    if (count > 10) {
      console.log('Count is high!');
    }
  }

  function increment() {
    count += 1;
    dispatch('change', { count });
  }
</script>

<div>
  <h2>{title}</h2>
  <p>Count: {count}, Doubled: {doubled}</p>
  <button on:click={increment}>Increment</button>
</div>

<!-- AFTER: Svelte 5 Component -->
<script>
  let { count = 0, title, onChange } = $props();

  const doubled = $derived(count * 2);

  $effect(() => {
    if (count > 10) {
      console.log('Count is high!');
    }
  });

  function increment() {
    count += 1;
    onChange?.({ count });
  }
</script>

<div>
  <h2>{title}</h2>
  <p>Count: {count}, Doubled: {doubled}</p>
  <button onclick={increment}>Increment</button>
</div>
```
**Source:** [Svelte 5 Migration Guide](https://svelte.dev/docs/svelte/v5-migration-guide)

### shadcn-svelte Component Usage
```svelte
<script>
  import { Button } from "$lib/components/ui/button/index.js";
  import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "$lib/components/ui/card/index.js";

  let count = $state(0);
</script>

<Card class="w-96">
  <CardHeader>
    <CardTitle>Sync Status</CardTitle>
    <CardDescription>Current sync progress</CardDescription>
  </CardHeader>
  <CardContent>
    <p>Files synced: {count}</p>
    <Button onclick={() => count++}>Sync More</Button>
  </CardContent>
</Card>
```
**Source:** [shadcn-svelte Installation](https://www.shadcn-svelte.com/docs/installation/sveltekit)

### macOS-Native Sidebar Layout
```svelte
<script>
  import { Menu, Settings, Folder } from "lucide-svelte";
</script>

<div class="flex h-screen">
  <!-- Sidebar with macOS vibrancy effect -->
  <aside class="
    w-64 h-full
    bg-white/30 dark:bg-gray-900/30
    backdrop-blur-md backdrop-saturate-150
    border-r border-gray-200/50 dark:border-gray-700/50
  ">
    <nav class="p-4 space-y-2">
      <button class="
        w-full flex items-center gap-3 px-3 py-2 rounded-lg
        text-gray-700 dark:text-gray-200
        hover:bg-gray-200/50 dark:hover:bg-gray-700/50
        transition-colors
      ">
        <Folder size={20} />
        <span>Folder Pairs</span>
      </button>
      <button class="
        w-full flex items-center gap-3 px-3 py-2 rounded-lg
        text-gray-700 dark:text-gray-200
        hover:bg-gray-200/50 dark:hover:bg-gray-700/50
        transition-colors
      ">
        <Menu size={20} />
        <span>Peers</span>
      </button>
      <button class="
        w-full flex items-center gap-3 px-3 py-2 rounded-lg
        text-gray-700 dark:text-gray-200
        hover:bg-gray-200/50 dark:hover:bg-gray-700/50
        transition-colors
      ">
        <Settings size={20} />
        <span>Settings</span>
      </button>
    </nav>
  </aside>

  <!-- Main content area -->
  <main class="flex-1 overflow-auto bg-white dark:bg-gray-950">
    <slot />
  </main>
</div>
```
**Source:** [Tailwind Backdrop Blur](https://tailwindcss.com/docs/backdrop-blur)

### Dark Mode Color Tokens
```css
/* app.css */
@import 'tailwindcss';

@theme {
  /* macOS system colors */
  --color-macos-blue: #007AFF;
  --color-macos-green: #34C759;
  --color-macos-orange: #FF9500;
  --color-macos-red: #FF3B30;
  --color-macos-gray: #8E8E93;

  /* Sidebar colors */
  --color-sidebar-light: rgba(255, 255, 255, 0.3);
  --color-sidebar-dark: rgba(17, 24, 39, 0.3);
}

/* System font */
body {
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto,
               "Helvetica Neue", Arial, sans-serif;
}
```
**Source:** [Tailwind CSS v4](https://tailwindcss.com/blog/tailwindcss-v4)

### $derived vs $effect Decision Tree
```svelte
<script>
  // ✅ GOOD: Use $derived for computed values
  let firstName = $state('John');
  let lastName = $state('Doe');
  const fullName = $derived(`${firstName} ${lastName}`);

  // ✅ GOOD: Use $derived.by for complex computations
  let numbers = $state([1, 2, 3, 4, 5]);
  const sum = $derived.by(() => {
    let total = 0;
    for (const n of numbers) {
      total += n;
    }
    return total;
  });

  // ✅ GOOD: Use $effect for side effects
  let count = $state(0);
  $effect(() => {
    console.log('Count changed:', count);
    document.title = `Count: ${count}`;
  });

  // ❌ BAD: Using $effect to compute state
  let doubled = $state(0);
  $effect(() => {
    doubled = count * 2; // This creates hydration issues!
  });

  // ✅ BETTER: Use $derived instead
  const doubledCorrect = $derived(count * 2);
</script>
```
**Source:** [Understanding Svelte 5 Runes: $derived vs $effect](https://dev.to/mikehtmlallthethings/understanding-svelte-5-runes-derived-vs-effect-1hh)

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Svelte 3 implicit reactivity (let) | Svelte 5 explicit runes ($state) | Oct 2024 | Better performance, clearer code, easier debugging |
| $: for computed values | $derived rune | Oct 2024 | Explicit dependencies, server-side compatible, cached |
| createEventDispatcher | Callback props | Oct 2024 | Simpler, type-safe, better TypeScript support |
| Slots | Snippets | Oct 2024 | More flexible, composable, better control flow |
| Tailwind v3 JS config | Tailwind v4 CSS-first @theme | Jan 2025 | 5x faster builds, simpler config, modern CSS features |
| Manual dark mode | prefers-color-scheme + dark: | 2018-present | Respects system preferences, no JS needed |
| SF Pro Download | System font stack (-apple-system) | 2015-present | No licensing issues, automatically uses San Francisco |

**Deprecated/outdated:**
- **createEventDispatcher:** Deprecated in Svelte 5, use callback props instead
- **Slots:** Replaced by snippets in Svelte 5 (backward compatible but deprecated)
- **Event modifiers (|preventDefault):** Removed in Svelte 5, use explicit event handling
- **Component class API (new Component()):** Replaced with mount() and hydrate() functions
- **$: reactive statements:** Still work but $derived/$effect are preferred for new code
- **Tailwind v3 JavaScript config:** v4 uses CSS @theme directive instead
- **SF Symbols for web:** Never officially supported, use Lucide/Heroicons instead

## Open Questions

Things that couldn't be fully resolved:

1. **shadcn-svelte CLI dependency auto-upgrades**
   - What we know: CLI can automatically upgrade bits-ui, potentially breaking apps
   - What's unclear: Best practice for version pinning vs allowing upgrades
   - Recommendation: Pin bits-ui version in package.json after initial install, manually upgrade when ready

2. **macOS system color CSS variables**
   - What we know: CSS custom properties can use prefers-color-scheme, Tailwind v4 has @theme
   - What's unclear: Exact values for macOS system colors (--system-blue, etc.) for 2026
   - Recommendation: Use Tailwind's built-in semantic colors, customize via @theme if needed

3. **Performance impact of backdrop-blur on low-end hardware**
   - What we know: backdrop-blur-* utilities work, WebKit has good support
   - What's unclear: Performance benchmarks for Wails v3 with heavy blur usage
   - Recommendation: Use backdrop-blur-sm for subtle effects, test on target hardware, provide option to disable vibrancy

4. **Wails v3 alpha stability for production**
   - What we know: Wails v3 is in alpha (v3.0.0-alpha.62), used in production by some
   - What's unclear: Timeline for stable release, breaking changes expected
   - Recommendation: Proceed with caution, pin version in package.json, monitor changelog

## Sources

### Primary (HIGH confidence)
- [Svelte 5 Migration Guide](https://svelte.dev/docs/svelte/v5-migration-guide) - Official migration documentation
- [Svelte 5 $state Rune](https://svelte.dev/docs/svelte/$state) - Reactivity patterns and examples
- [Svelte 5 $derived Rune](https://svelte.dev/docs/svelte/$derived) - Computed state patterns
- [Svelte 5 $effect Rune](https://svelte.dev/docs/svelte/$effect) - Side effects documentation
- [shadcn-svelte Installation](https://www.shadcn-svelte.com/docs/installation) - Setup guide
- [shadcn-svelte SvelteKit](https://www.shadcn-svelte.com/docs/installation/sveltekit) - SvelteKit-specific setup
- [Tailwind CSS v4](https://tailwindcss.com/blog/tailwindcss-v4) - v4 release announcement and features
- [Tailwind Backdrop Blur](https://tailwindcss.com/docs/backdrop-blur) - Vibrancy effects
- [WebKit System Font](https://webkit.org/blog/3709/using-the-system-font-in-web-content/) - San Francisco integration
- [Wails SvelteKit Guide](https://wails.io/docs/guides/sveltekit/) - Official Wails documentation

### Secondary (MEDIUM confidence)
- [Svelte 5 Release](https://svelte.dev/blog/svelte-5-is-alive) - Oct 2024 stable release
- [Understanding Svelte 5 Runes: $derived vs $effect](https://dev.to/mikehtmlallthethings/understanding-svelte-5-runes-derived-vs-effect-1hh) - Best practices verified with official docs
- [Dark Mode Support in WebKit](https://webkit.org/blog/8840/dark-mode-support-in-webkit/) - prefers-color-scheme implementation
- [Experiences and Caveats of Svelte 5 Migration](https://dev.to/kvetoslavnovak/experiences-and-caveats-of-svelte-5-migration-27cp) - Community migration insights
- [Lucide Svelte](https://lucide.dev/guide/packages/lucide-svelte) - Icon library documentation
- [Heroicons](https://heroicons.com/) - Alternative icon library by Tailwind Labs
- [shadcn-svelte GitHub Issues](https://github.com/huntabyte/shadcn-svelte/issues) - Known issues and troubleshooting
- [Apple Human Interface Guidelines](https://developer.apple.com/design/human-interface-guidelines/designing-for-macos) - macOS design principles

### Tertiary (LOW confidence)
- macOS Tailwind project (Storybook) - Found but couldn't access full documentation
- SF Symbols web usage - Verified as NOT supported for web, licensing prohibits
- macOS system color exact values - No authoritative source for 2026 values, use Tailwind defaults

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - Official documentation for all libraries, stable releases, clear version numbers
- Architecture: HIGH - Official migration guide, verified code examples, community validation
- Pitfalls: MEDIUM - Based on community reports and GitHub issues, not all officially documented
- macOS visual patterns: MEDIUM - Tailwind backdrop-blur verified, system colors approximated
- Performance: LOW - No benchmarks for Wails v3 + Svelte 5 + heavy blur effects

**Research date:** 2026-01-23
**Valid until:** 2026-02-23 (30 days - stable ecosystem, Svelte 5 is stable, Tailwind v4 is stable)

**Notes:**
- Svelte 5 released Oct 2024, ecosystem mature
- Tailwind v4 released Jan 2025, stable
- Wails v3 still in alpha, monitor for updates
- SF Symbols licensing prohibits web use - verified across multiple sources
- Migration script exists but manual review required for quality
- 90% of reactive code should use $derived, not $effect (critical for performance)
