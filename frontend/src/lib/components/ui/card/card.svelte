<script>
  import { cn } from '$lib/utils.js';

  let {
    class: className,
    children,
    ...restProps
  } = $props();

  const cardClass = $derived(
    cn(
      'rounded-xl border border-white/10 bg-white/5 dark:bg-white/3 backdrop-blur-sm shadow-sm',
      className
    )
  );
</script>

<div class={cardClass} {...restProps}>
  {@render children?.()}
</div>
