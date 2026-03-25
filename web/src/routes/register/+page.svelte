<script>
	import { register } from '$lib/api.js';
	import { auth } from '$lib/stores/auth.js';
	import { goto } from '$app/navigation';

	let email = $state('');
	let password = $state('');
	let confirmPassword = $state('');
	let displayName = $state('');
	let error = $state('');
	let loading = $state(false);

	$effect(() => {
		if ($auth?.accessToken) goto('/notes');
	});

	async function handleSubmit(e) {
		e.preventDefault();
		error = '';

		if (password !== confirmPassword) {
			error = 'Passwords do not match';
			return;
		}

		loading = true;
		try {
			await register(email, password, displayName);
			goto('/login');
		} catch (err) {
			error = err.message;
		} finally {
			loading = false;
		}
	}
</script>

<div class="min-h-screen flex items-center justify-center bg-gray-100">
	<div class="bg-white p-8 rounded shadow-md w-full max-w-sm">
		<h1 class="text-2xl font-bold mb-6 text-center">Create Account</h1>

		{#if error}
			<div class="bg-red-50 text-red-600 p-3 rounded mb-4 text-sm">{error}</div>
		{/if}

		<form onsubmit={handleSubmit}>
			<label class="block mb-4">
				<span class="text-sm text-gray-600">Display name</span>
				<input
					type="text"
					bind:value={displayName}
					required
					class="mt-1 block w-full px-3 py-2 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
				/>
			</label>

			<label class="block mb-4">
				<span class="text-sm text-gray-600">Email</span>
				<input
					type="email"
					bind:value={email}
					required
					class="mt-1 block w-full px-3 py-2 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
				/>
			</label>

			<label class="block mb-4">
				<span class="text-sm text-gray-600">Password</span>
				<input
					type="password"
					bind:value={password}
					required
					class="mt-1 block w-full px-3 py-2 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
				/>
			</label>

			<label class="block mb-6">
				<span class="text-sm text-gray-600">Confirm password</span>
				<input
					type="password"
					bind:value={confirmPassword}
					required
					class="mt-1 block w-full px-3 py-2 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
				/>
			</label>

			<button
				type="submit"
				disabled={loading}
				class="w-full bg-blue-600 text-white py-2 rounded hover:bg-blue-700 disabled:opacity-50"
			>
				{loading ? 'Creating...' : 'Create Account'}
			</button>
		</form>

		<p class="mt-4 text-center text-sm text-gray-500">
			Already have an account? <a href="/login" class="text-blue-600 hover:underline">Log in</a>
		</p>
	</div>
</div>
