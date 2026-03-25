import { syncChanges, syncPush } from './api.js';
import { getLastSync, setLastSync, getLocalChanges, applyServerChanges } from './db.js';
import { auth } from './stores/auth.js';
import { get } from 'svelte/store';
import { writable } from 'svelte/store';

export const syncStatus = writable('idle'); // idle | syncing | error | offline

let syncTimer = null;

export function startSync(intervalMs = 30000) {
	stopSync();
	doSync();
	syncTimer = setInterval(doSync, intervalMs);
}

export function stopSync() {
	if (syncTimer) {
		clearInterval(syncTimer);
		syncTimer = null;
	}
}

export async function doSync() {
	const session = get(auth);
	if (!session?.accessToken) return;

	if (!navigator.onLine) {
		syncStatus.set('offline');
		return;
	}

	syncStatus.set('syncing');

	try {
		// Pull server changes
		const lastSync = await getLastSync();
		const changes = await syncChanges(lastSync);
		await applyServerChanges(changes.notes, changes.todos);

		// Push local changes
		const local = await getLocalChanges(lastSync);
		if (local.notes.length > 0 || local.todos.length > 0) {
			const deviceId = `web-${session.user.id.slice(0, 8)}`;
			await syncPush(local.notes, local.todos, deviceId);
		}

		await setLastSync(changes.sync_timestamp);
		syncStatus.set('idle');
	} catch (err) {
		console.error('sync error:', err);
		syncStatus.set('error');
	}
}
