<script lang="ts">
	let mailCount = $state(0);
</script>

<svelte:head>
	<title>Yunt - Development Mail Server</title>
</svelte:head>

<main class="flex min-h-screen flex-col items-center justify-center p-8">
	<div class="card max-w-md p-8 text-center">
		<h1 class="mb-4 text-3xl font-bold text-primary-600">Yunt</h1>
		<p class="mb-6 text-secondary-600">Development Mail Server</p>

		<div class="mb-6 rounded-lg bg-secondary-100 p-4">
			<p class="text-sm text-secondary-500">Messages captured</p>
			<p class="text-4xl font-bold text-secondary-900">{mailCount}</p>
		</div>

		<div class="space-y-2 text-left text-sm">
			<div class="flex justify-between">
				<span class="text-secondary-500">SMTP:</span>
				<span class="font-mono text-secondary-700">localhost:1025</span>
			</div>
			<div class="flex justify-between">
				<span class="text-secondary-500">IMAP:</span>
				<span class="font-mono text-secondary-700">localhost:1143</span>
			</div>
			<div class="flex justify-between">
				<span class="text-secondary-500">Web UI:</span>
				<span class="font-mono text-secondary-700">localhost:8025</span>
			</div>
		</div>

		<button class="btn-primary mt-6 w-full" onclick={() => (mailCount += 1)}>Refresh</button>
	</div>
</main>
