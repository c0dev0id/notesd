<script>
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { auth } from '$lib/stores/auth.js';
	import NoteList from '$lib/components/NoteList.svelte';
	import Editor from '$lib/components/Editor.svelte';
	import { listNotes, createNote, updateNote, deleteNote, getNote, searchNotes } from '$lib/api.js';
	import { getDeviceId } from '$lib/device.js';

	let notes = $state([]);
	let selectedId = $state(null);
	let selectedNote = $state(null);
	let title = $state('');
	let saving = $state(false);
	let searchQuery = $state('');
	let searchTimeout;
	let editor = $state(null);
	let saveTimeout;

	$effect(() => {
		if (!$auth?.accessToken) goto('/login');
	});

	onMount(loadNotes);

	async function loadNotes() {
		try {
			const resp = await listNotes(200);
			notes = resp.notes;
		} catch (err) {
			console.error('load notes:', err);
		}
	}

	async function selectNote(id) {
		// Save current note before switching
		if (selectedId && selectedNote) {
			await saveCurrentNote();
		}

		selectedId = id;
		try {
			selectedNote = await getNote(id);
			title = selectedNote.title;
			if (editor) {
				editor.setContent(selectedNote.content || '');
			}
		} catch (err) {
			console.error('get note:', err);
			selectedNote = null;
		}
	}

	async function handleCreate() {
		try {
			const note = await createNote('', '', 'note', getDeviceId());
			notes = [note, ...notes];
			await selectNote(note.id);
		} catch (err) {
			console.error('create note:', err);
		}
	}

	function handleContentUpdate(html) {
		if (!selectedNote) return;
		// Debounced auto-save
		clearTimeout(saveTimeout);
		saveTimeout = setTimeout(() => saveCurrentNote(html), 1000);
	}

	function handleTitleInput() {
		clearTimeout(saveTimeout);
		saveTimeout = setTimeout(() => saveCurrentNote(), 1000);
	}

	async function saveCurrentNote(contentOverride) {
		if (!selectedId || saving) return;
		saving = true;
		try {
			const content = contentOverride ?? editor?.getHTML() ?? '';
			const updated = await updateNote(selectedId, {
				title,
				content
			}, getDeviceId());
			selectedNote = updated;
			// Update in list
			notes = notes.map(n => n.id === updated.id ? updated : n);
		} catch (err) {
			console.error('save note:', err);
		} finally {
			saving = false;
		}
	}

	async function handleDelete() {
		if (!selectedId) return;
		try {
			await deleteNote(selectedId);
			notes = notes.filter(n => n.id !== selectedId);
			selectedId = null;
			selectedNote = null;
			title = '';
			if (editor) editor.setContent('');
		} catch (err) {
			console.error('delete note:', err);
		}
	}

	async function handleSearch() {
		clearTimeout(searchTimeout);
		searchTimeout = setTimeout(async () => {
			if (!searchQuery.trim()) {
				await loadNotes();
				return;
			}
			try {
				const resp = await searchNotes(searchQuery.trim());
				notes = resp.notes;
			} catch (err) {
				console.error('search:', err);
			}
		}, 300);
	}
</script>

<div class="flex h-full">
	<!-- Sidebar -->
	<div class="w-72 border-r border-gray-200 flex flex-col bg-white">
		<div class="p-3 border-b border-gray-200">
			<input
				type="text"
				placeholder="Search notes..."
				bind:value={searchQuery}
				oninput={handleSearch}
				class="w-full px-3 py-1.5 border border-gray-300 rounded text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
			/>
		</div>
		<NoteList
			{notes}
			selected={selectedId}
			onselect={selectNote}
			oncreate={handleCreate}
		/>
	</div>

	<!-- Editor pane -->
	<div class="flex-1 flex flex-col bg-white">
		{#if selectedNote}
			<div class="border-b border-gray-200 p-3 flex items-center gap-2">
				<input
					type="text"
					bind:value={title}
					oninput={handleTitleInput}
					placeholder="Note title"
					class="flex-1 text-lg font-medium focus:outline-none"
				/>
				<span class="text-xs text-gray-400">
					{saving ? 'Saving...' : 'Saved'}
				</span>
				<button
					onclick={handleDelete}
					class="text-sm text-gray-400 hover:text-red-500 px-2"
				>Delete</button>
			</div>
			<div class="flex-1 overflow-y-auto">
				<Editor
					bind:this={editor}
					content={selectedNote.content || ''}
					onupdate={handleContentUpdate}
				/>
			</div>
		{:else}
			<div class="flex-1 flex items-center justify-center text-gray-400">
				<div class="text-center">
					<p class="text-lg mb-2">Select a note or create a new one</p>
					<button
						onclick={handleCreate}
						class="text-blue-600 hover:underline text-sm"
					>Create new note</button>
				</div>
			</div>
		{/if}
	</div>
</div>
