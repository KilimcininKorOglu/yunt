<script lang="ts">
	import '../app.css';

	interface Props {
		children?: import('svelte').Snippet;
	}

	const { children }: Props = $props();
</script>

<div class="min-h-screen bg-secondary-50">
	{@render children?.()}
</div>
