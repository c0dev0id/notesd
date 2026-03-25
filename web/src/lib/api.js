import { auth } from './stores/auth.js';
import { get } from 'svelte/store';

const BASE = '/api/v1';

async function request(method, path, body) {
	const session = get(auth);
	const headers = { 'Content-Type': 'application/json' };

	if (session?.accessToken) {
		headers['Authorization'] = `Bearer ${session.accessToken}`;
	}

	const opts = { method, headers };
	if (body) {
		opts.body = JSON.stringify(body);
	}

	let resp = await fetch(BASE + path, opts);

	// Auto-refresh on 401
	if (resp.status === 401 && session?.refreshToken) {
		const refreshed = await refreshTokens(session.refreshToken);
		if (refreshed) {
			headers['Authorization'] = `Bearer ${refreshed.accessToken}`;
			opts.headers = headers;
			resp = await fetch(BASE + path, opts);
		}
	}

	return resp;
}

async function refreshTokens(refreshToken) {
	const resp = await fetch(BASE + '/auth/refresh', {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ refresh_token: refreshToken })
	});

	if (!resp.ok) {
		auth.logout();
		return null;
	}

	const data = await resp.json();
	auth.setSession({
		accessToken: data.access_token,
		refreshToken: data.refresh_token,
		user: data.user
	});
	return { accessToken: data.access_token };
}

async function jsonOrError(resp) {
	if (resp.status === 204) return null;
	const data = await resp.json();
	if (!resp.ok) {
		throw new Error(data.error || `HTTP ${resp.status}`);
	}
	return data;
}

// Auth

export async function login(email, password, deviceId) {
	const resp = await fetch(BASE + '/auth/login', {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ email, password, device_id: deviceId })
	});
	const data = await jsonOrError(resp);
	auth.setSession({
		accessToken: data.access_token,
		refreshToken: data.refresh_token,
		user: data.user
	});
	return data.user;
}

export async function register(email, password, displayName) {
	const resp = await fetch(BASE + '/auth/register', {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ email, password, display_name: displayName })
	});
	return jsonOrError(resp);
}

export async function logout() {
	try {
		await request('POST', '/auth/logout');
	} catch {
		// Best effort
	}
	auth.logout();
}

// Notes

export async function listNotes(limit = 50, offset = 0) {
	const resp = await request('GET', `/notes?limit=${limit}&offset=${offset}`);
	return jsonOrError(resp);
}

export async function getNote(id) {
	const resp = await request('GET', `/notes/${id}`);
	return jsonOrError(resp);
}

export async function createNote(title, content, type, deviceId) {
	const resp = await request('POST', '/notes', {
		title, content, type, device_id: deviceId
	});
	return jsonOrError(resp);
}

export async function updateNote(id, updates, deviceId) {
	const resp = await request('PUT', `/notes/${id}`, {
		...updates, device_id: deviceId
	});
	return jsonOrError(resp);
}

export async function deleteNote(id) {
	const resp = await request('DELETE', `/notes/${id}`);
	return jsonOrError(resp);
}

export async function searchNotes(query, limit = 50) {
	const resp = await request('GET', `/notes/search?q=${encodeURIComponent(query)}&limit=${limit}`);
	return jsonOrError(resp);
}

// Todos

export async function listTodos(limit = 50, offset = 0) {
	const resp = await request('GET', `/todos?limit=${limit}&offset=${offset}`);
	return jsonOrError(resp);
}

export async function getTodo(id) {
	const resp = await request('GET', `/todos/${id}`);
	return jsonOrError(resp);
}

export async function createTodo(content, deviceId, dueDate, noteId) {
	const body = { content, device_id: deviceId };
	if (dueDate) body.due_date = dueDate;
	if (noteId) body.note_id = noteId;
	const resp = await request('POST', '/todos', body);
	return jsonOrError(resp);
}

export async function updateTodo(id, updates, deviceId) {
	const resp = await request('PUT', `/todos/${id}`, {
		...updates, device_id: deviceId
	});
	return jsonOrError(resp);
}

export async function deleteTodo(id) {
	const resp = await request('DELETE', `/todos/${id}`);
	return jsonOrError(resp);
}

export async function getOverdueTodos() {
	const resp = await request('GET', '/todos/overdue');
	return jsonOrError(resp);
}

// Sync

export async function syncChanges(sinceMs) {
	const resp = await request('GET', `/sync/changes?since=${sinceMs}`);
	return jsonOrError(resp);
}

export async function syncPush(notes, todos, deviceId) {
	const resp = await request('POST', '/sync/push', {
		notes, todos, device_id: deviceId
	});
	return jsonOrError(resp);
}
