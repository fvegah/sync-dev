<script>
  import { cn } from '$lib/utils.js';

  let {
    class: className,
    children,
    ...restProps
  } = $props();

  const contentClass = $derived(
    cn('p-6 pt-0', className)
  );
</script>

<div class={contentClass} {...restProps}>
  {@render children?.()}
</div>
