<script>
	let { notes = [], selected = null, onselect = () => {}, oncreate = () => {} } = $props();

	function formatDate(dateStr) {
		if (!dateStr) return '';
		const d = new Date(dateStr);
		return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric' });
	}

	function truncate(s, len) {
		if (!s || s.length <= len) return s || '';
		return s.slice(0, len) + '...';
	}

	function stripHtml(html) {
		if (!html) return '';
		return html.replace(/<[^>]*>/g, ' ').replace(/\s+/g, ' ').trim();
	}
</script>

<div class="flex flex-col h-full">
	<div class="p-3 border-b border-gray-200">
		<button
			onclick={oncreate}
			class="w-full px-3 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 text-sm"
		>New Note</button>
	</div>

	<div class="flex-1 overflow-y-auto">
		{#if notes.length === 0}
			<p class="p-4 text-gray-400 text-sm text-center">No notes yet</p>
		{/if}
		{#each notes as note (note.id)}
			<button
				class="w-full text-left p-3 border-b border-gray-100 hover:bg-gray-50 block"
				class:bg-blue-50={selected === note.id}
				onclick={() => onselect(note.id)}
			>
				<div class="flex justify-between items-start">
					<span class="font-medium text-sm truncate">
						{note.title || '(untitled)'}
					</span>
					<span class="text-xs text-gray-400 ml-2 whitespace-nowrap">
						{formatDate(note.modified_at)}
					</span>
				</div>
				<p class="text-xs text-gray-500 mt-1 truncate">
					{truncate(stripHtml(note.content), 80)}
				</p>
			</button>
		{/each}
	</div>
</div>
