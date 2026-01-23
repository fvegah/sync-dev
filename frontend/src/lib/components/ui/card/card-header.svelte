<script>
  import { cn } from '$lib/utils.js';

  let {
    class: className,
    children,
    ...restProps
  } = $props();

  const headerClass = $derived(
    cn('flex flex-col space-y-1.5 p-6', className)
  );
</script>

<div class={headerClass} {...restProps}>
  {@render children?.()}
</div>
