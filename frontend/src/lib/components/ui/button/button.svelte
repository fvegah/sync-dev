<script>
  import { cn } from '$lib/utils.js';

  let {
    variant = 'default',
    size = 'default',
    class: className,
    children,
    ...restProps
  } = $props();

  const baseStyles = 'inline-flex items-center justify-center rounded-lg font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-macos-blue focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50';

  const variants = $derived({
    default: 'bg-macos-blue text-white hover:bg-macos-blue/90 active:bg-macos-blue/80',
    secondary: 'bg-white/10 dark:bg-white/5 text-gray-900 dark:text-white hover:bg-white/20 dark:hover:bg-white/10 active:bg-white/30 dark:active:bg-white/15',
    ghost: 'hover:bg-white/10 dark:hover:bg-white/5 active:bg-white/20 dark:active:bg-white/10',
    destructive: 'bg-macos-red text-white hover:bg-macos-red/90 active:bg-macos-red/80'
  });

  const sizes = $derived({
    default: 'h-9 px-4 py-2 text-sm',
    sm: 'h-8 px-3 text-xs',
    lg: 'h-11 px-8 text-base',
    icon: 'h-9 w-9'
  });

  const buttonClass = $derived(
    cn(baseStyles, variants[variant], sizes[size], className)
  );
</script>

<button class={buttonClass} {...restProps}>
  {@render children?.()}
</button>
