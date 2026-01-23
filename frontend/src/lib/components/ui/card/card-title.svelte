<script>
  import { cn } from '$lib/utils.js';

  let {
    class: className,
    children,
    ...restProps
  } = $props();

  const titleClass = $derived(
    cn('text-lg font-semibold leading-none tracking-tight', className)
  );
</script>

<h3 class={titleClass} {...restProps}>
  {@render children?.()}
</h3>
