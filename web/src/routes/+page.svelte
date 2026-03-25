<script>
	import { auth } from '$lib/stores/auth.js';
	import { goto } from '$app/navigation';

	$effect(() => {
		if ($auth?.accessToken) {
			goto('/notes');
		} else {
			goto('/login');
		}
	});
</script>
