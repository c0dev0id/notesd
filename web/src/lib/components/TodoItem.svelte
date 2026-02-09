<script>
	let { todo, ontoggle = () => {}, ondelete = () => {} } = $props();

	function formatDue(dateStr) {
		if (!dateStr) return '';
		const d = new Date(dateStr);
		return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' });
	}

	function isOverdue(dateStr) {
		if (!dateStr) return false;
		return new Date(dateStr) < new Date() && !todo.completed;
	}
</script>

<div class="flex items-start gap-3 p-3 border-b border-gray-100 hover:bg-gray-50 group">
	<input
		type="checkbox"
		checked={todo.completed}
		onchange={() => ontoggle(todo)}
		class="mt-1 h-4 w-4 rounded border-gray-300"
	/>
	<div class="flex-1 min-w-0">
		<p class="text-sm" class:line-through={todo.completed} class:text-gray-400={todo.completed}>
			{todo.content}
		</p>
		{#if todo.due_date}
			<p class="text-xs mt-0.5"
				class:text-red-500={isOverdue(todo.due_date)}
				class:text-gray-400={!isOverdue(todo.due_date)}
			>
				Due: {formatDue(todo.due_date)}
			</p>
		{/if}
	</div>
	<button
		onclick={() => ondelete(todo)}
		class="text-gray-300 hover:text-red-500 opacity-0 group-hover:opacity-100 text-sm"
	>Delete</button>
</div>
