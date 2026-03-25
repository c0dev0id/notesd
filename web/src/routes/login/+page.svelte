<script>
	import { login } from '$lib/api.js';
	import { auth } from '$lib/stores/auth.js';
	import { getDeviceId } from '$lib/device.js';
	import { goto } from '$app/navigation';

	let email = $state('');
	let password = $state('');
	let error = $state('');
	let loading = $state(false);

	$effect(() => {
		if ($auth?.accessToken) goto('/notes');
	});

	async function handleSubmit(e) {
		e.preventDefault();
		error = '';
		loading = true;
		try {
			await login(email, password, getDeviceId());
			goto('/notes');
		} catch (err) {
			error = err.message;
		} finally {
			loading = false;
		}
	}
</script>

<div class="min-h-screen flex items-center justify-center bg-gray-100">
	<div class="bg-white p-8 rounded shadow-md w-full max-w-sm">
		<h1 class="text-2xl font-bold mb-6 text-center">notesd</h1>

		{#if error}
			<div class="bg-red-50 text-red-600 p-3 rounded mb-4 text-sm">{error}</div>
		{/if}

		<form onsubmit={handleSubmit}>
			<label class="block mb-4">
				<span class="text-sm text-gray-600">Email</span>
				<input
					type="email"
					bind:value={email}
					required
					class="mt-1 block w-full px-3 py-2 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
				/>
			</label>

			<label class="block mb-6">
				<span class="text-sm text-gray-600">Password</span>
				<input
					type="password"
					bind:value={password}
					required
					class="mt-1 block w-full px-3 py-2 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
				/>
			</label>

			<button
				type="submit"
				disabled={loading}
				class="w-full bg-blue-600 text-white py-2 rounded hover:bg-blue-700 disabled:opacity-50"
			>
				{loading ? 'Logging in...' : 'Log in'}
			</button>
		</form>

		<p class="mt-4 text-center text-sm text-gray-500">
			No account? <a href="/register" class="text-blue-600 hover:underline">Register</a>
		</p>
	</div>
</div>
