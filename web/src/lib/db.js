import Dexie from 'dexie';

export const db = new Dexie('notesd');

db.version(1).stores({
	notes: 'id, user_id, modified_at, deleted_at, type',
	todos: 'id, user_id, modified_at, deleted_at, due_date, completed, note_id',
	meta: 'key'
});

// Local CRUD operations — work offline against IndexedDB.

export async function localGetNotes() {
	return db.notes
		.filter(n => !n.deleted_at)
		.reverse()
		.sortBy('modified_at');
}

export async function localGetNote(id) {
	const note = await db.notes.get(id);
	if (!note || note.deleted_at) return null;
	return note;
}

export async function localPutNote(note) {
	await db.notes.put(note);
}

export async function localDeleteNote(id, now, deviceId) {
	const note = await db.notes.get(id);
	if (!note) return;
	note.deleted_at = now;
	note.modified_at = now;
	note.modified_by_device = deviceId;
	await db.notes.put(note);
}

export async function localSearchNotes(query) {
	const q = query.toLowerCase();
	return db.notes
		.filter(n => !n.deleted_at && (
			(n.title || '').toLowerCase().includes(q) ||
			(n.content || '').toLowerCase().includes(q)
		))
		.reverse()
		.sortBy('modified_at');
}

export async function localGetTodos() {
	return db.todos
		.filter(t => !t.deleted_at)
		.reverse()
		.sortBy('modified_at');
}

export async function localGetTodo(id) {
	const todo = await db.todos.get(id);
	if (!todo || todo.deleted_at) return null;
	return todo;
}

export async function localPutTodo(todo) {
	await db.todos.put(todo);
}

export async function localDeleteTodo(id, now, deviceId) {
	const todo = await db.todos.get(id);
	if (!todo) return;
	todo.deleted_at = now;
	todo.modified_at = now;
	todo.modified_by_device = deviceId;
	await db.todos.put(todo);
}

export async function localGetOverdueTodos() {
	const now = new Date().toISOString();
	return db.todos
		.filter(t => !t.deleted_at && !t.completed && t.due_date && t.due_date < now)
		.sortBy('due_date');
}

// Sync metadata

export async function getLastSync() {
	const meta = await db.meta.get('last_sync');
	return meta?.value || 0;
}

export async function setLastSync(timestamp) {
	await db.meta.put({ key: 'last_sync', value: timestamp });
}

// Get locally modified items since last sync for push
export async function getLocalChanges(sinceMs) {
	const notes = await db.notes
		.filter(n => new Date(n.modified_at).getTime() > sinceMs)
		.toArray();
	const todos = await db.todos
		.filter(t => new Date(t.modified_at).getTime() > sinceMs)
		.toArray();
	return { notes, todos };
}

// Apply server changes locally (LWW — server version wins on conflict)
export async function applyServerChanges(notes, todos) {
	await db.transaction('rw', db.notes, db.todos, async () => {
		for (const note of notes) {
			const local = await db.notes.get(note.id);
			if (!local || new Date(note.modified_at) >= new Date(local.modified_at)) {
				await db.notes.put(note);
			}
		}
		for (const todo of todos) {
			const local = await db.todos.get(todo.id);
			if (!local || new Date(todo.modified_at) >= new Date(local.modified_at)) {
				await db.todos.put(todo);
			}
		}
	});
}

export async function clearLocalData() {
	await db.notes.clear();
	await db.todos.clear();
	await db.meta.clear();
}
