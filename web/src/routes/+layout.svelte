<script>
	import '../app.css';
	import { auth } from '$lib/stores/auth.js';
	import { logout } from '$lib/api.js';
	import { syncStatus, startSync, stopSync } from '$lib/sync.js';
	import { clearLocalData } from '$lib/db.js';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { onDestroy } from 'svelte';

	let { children } = $props();

	let session = $derived($auth);
	let status = $derived($syncStatus);

	$effect(() => {
		if (session?.accessToken) {
			startSync();
		} else {
			stopSync();
		}
	});

	onDestroy(stopSync);

	async function handleLogout() {
		stopSync();
		await logout();
		await clearLocalData();
		goto('/login');
	}

	function statusLabel(s) {
		if (s === 'syncing') return 'Syncing...';
		if (s === 'error') return 'Sync error';
		if (s === 'offline') return 'Offline';
		return 'Synced';
	}

	function statusColor(s) {
		if (s === 'error') return 'text-red-500';
		if (s === 'offline') return 'text-yellow-500';
		if (s === 'syncing') return 'text-blue-500';
		return 'text-green-500';
	}

	let isAuthPage = $derived(
		page.url?.pathname === '/login' || page.url?.pathname === '/register'
	);
</script>

{#if session && !isAuthPage}
<div class="h-screen flex flex-col">
	<nav class="bg-gray-800 text-white px-4 py-2 flex items-center justify-between">
		<div class="flex items-center gap-4">
			<a href="/" class="font-bold text-lg">notesd</a>
			<a href="/notes" class="text-sm hover:text-gray-300"
				class:text-white={page.url?.pathname?.startsWith('/notes')}
				class:text-gray-400={!page.url?.pathname?.startsWith('/notes')}
			>Notes</a>
			<a href="/todos" class="text-sm hover:text-gray-300"
				class:text-white={page.url?.pathname?.startsWith('/todos')}
				class:text-gray-400={!page.url?.pathname?.startsWith('/todos')}
			>Todos</a>
		</div>
		<div class="flex items-center gap-4 text-sm">
			<span class={statusColor(status)}>{statusLabel(status)}</span>
			<span class="text-gray-400">{session.user?.display_name}</span>
			<button onclick={handleLogout} class="text-gray-400 hover:text-white">Logout</button>
		</div>
	</nav>

	<main class="flex-1 overflow-hidden">
		{@render children()}
	</main>
</div>
{:else}
	{@render children()}
{/if}
