import { writable } from 'svelte/store';

const STORAGE_KEY = 'notesd_session';

function createAuth() {
	let initial = null;
	if (typeof localStorage !== 'undefined') {
		try {
			const stored = localStorage.getItem(STORAGE_KEY);
			if (stored) initial = JSON.parse(stored);
		} catch {
			// Ignore corrupt storage
		}
	}

	const { subscribe, set } = writable(initial);

	return {
		subscribe,
		setSession(session) {
			set(session);
			if (typeof localStorage !== 'undefined') {
				localStorage.setItem(STORAGE_KEY, JSON.stringify(session));
			}
		},
		logout() {
			set(null);
			if (typeof localStorage !== 'undefined') {
				localStorage.removeItem(STORAGE_KEY);
			}
		}
	};
}

export const auth = createAuth();
