<script>
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { auth } from '$lib/stores/auth.js';
	import TodoItem from '$lib/components/TodoItem.svelte';
	import { listTodos, createTodo, updateTodo, deleteTodo, getOverdueTodos } from '$lib/api.js';
	import { getDeviceId } from '$lib/device.js';

	let todos = $state([]);
	let overdue = $state([]);
	let newContent = $state('');
	let newDueDate = $state('');
	let filter = $state('all'); // all | active | completed | overdue
	let loading = $state(false);

	$effect(() => {
		if (!$auth?.accessToken) goto('/login');
	});

	onMount(loadTodos);

	async function loadTodos() {
		loading = true;
		try {
			const [todoResp, overdueResp] = await Promise.all([
				listTodos(200),
				getOverdueTodos()
			]);
			todos = todoResp.todos;
			overdue = overdueResp;
		} catch (err) {
			console.error('load todos:', err);
		} finally {
			loading = false;
		}
	}

	async function handleCreate(e) {
		e.preventDefault();
		if (!newContent.trim()) return;

		try {
			const body = { content: newContent.trim(), device_id: getDeviceId() };
			if (newDueDate) {
				body.due_date = new Date(newDueDate).toISOString();
			}

			const todo = await createTodo(
				newContent.trim(),
				getDeviceId(),
				newDueDate ? new Date(newDueDate).toISOString() : null
			);
			todos = [todo, ...todos];
			newContent = '';
			newDueDate = '';
		} catch (err) {
			console.error('create todo:', err);
		}
	}

	async function handleToggle(todo) {
		try {
			const updated = await updateTodo(todo.id, {
				completed: !todo.completed
			}, getDeviceId());
			todos = todos.map(t => t.id === updated.id ? updated : t);
		} catch (err) {
			console.error('toggle todo:', err);
		}
	}

	async function handleDelete(todo) {
		try {
			await deleteTodo(todo.id);
			todos = todos.filter(t => t.id !== todo.id);
		} catch (err) {
			console.error('delete todo:', err);
		}
	}

	let filteredTodos = $derived.by(() => {
		switch (filter) {
			case 'active':
				return todos.filter(t => !t.completed);
			case 'completed':
				return todos.filter(t => t.completed);
			case 'overdue':
				return overdue;
			default:
				return todos;
		}
	});

	let activeCount = $derived(todos.filter(t => !t.completed).length);
	let overdueCount = $derived(overdue.length);
</script>

<div class="max-w-2xl mx-auto p-6">
	<h1 class="text-2xl font-bold mb-6">Todos</h1>

	<!-- Create form -->
	<form onsubmit={handleCreate} class="flex gap-2 mb-6">
		<input
			type="text"
			bind:value={newContent}
			placeholder="Add a todo..."
			class="flex-1 px-3 py-2 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
		/>
		<input
			type="date"
			bind:value={newDueDate}
			class="px-3 py-2 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
		/>
		<button
			type="submit"
			class="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700"
		>Add</button>
	</form>

	<!-- Filters -->
	<div class="flex gap-2 mb-4 text-sm">
		<button
			class="px-3 py-1 rounded"
			class:bg-gray-200={filter === 'all'}
			class:text-gray-600={filter !== 'all'}
			onclick={() => filter = 'all'}
		>All ({todos.length})</button>
		<button
			class="px-3 py-1 rounded"
			class:bg-gray-200={filter === 'active'}
			class:text-gray-600={filter !== 'active'}
			onclick={() => filter = 'active'}
		>Active ({activeCount})</button>
		<button
			class="px-3 py-1 rounded"
			class:bg-gray-200={filter === 'completed'}
			class:text-gray-600={filter !== 'completed'}
			onclick={() => filter = 'completed'}
		>Completed ({todos.length - activeCount})</button>
		{#if overdueCount > 0}
			<button
				class="px-3 py-1 rounded text-red-600"
				class:bg-red-100={filter === 'overdue'}
				onclick={() => filter = 'overdue'}
			>Overdue ({overdueCount})</button>
		{/if}
	</div>

	<!-- Todo list -->
	<div class="bg-white rounded shadow">
		{#if loading}
			<p class="p-6 text-center text-gray-400">Loading...</p>
		{:else if filteredTodos.length === 0}
			<p class="p-6 text-center text-gray-400">
				{filter === 'all' ? 'No todos yet' : `No ${filter} todos`}
			</p>
		{:else}
			{#each filteredTodos as todo (todo.id)}
				<TodoItem {todo} ontoggle={handleToggle} ondelete={handleDelete} />
			{/each}
		{/if}
	</div>
</div>
