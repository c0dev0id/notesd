<script>
	import { onMount, onDestroy } from 'svelte';
	import { Editor } from '@tiptap/core';
	import StarterKit from '@tiptap/starter-kit';
	import Placeholder from '@tiptap/extension-placeholder';

	let { content = '', onupdate = () => {}, editable = true } = $props();

	let element;
	let editor;

	onMount(() => {
		editor = new Editor({
			element,
			extensions: [
				StarterKit,
				Placeholder.configure({ placeholder: 'Start writing...' })
			],
			content,
			editable,
			onUpdate({ editor: e }) {
				onupdate(e.getHTML());
			},
			onTransaction() {
				// Force Svelte reactivity
				editor = editor;
			}
		});
	});

	onDestroy(() => {
		if (editor) editor.destroy();
	});

	export function getHTML() {
		return editor?.getHTML() || '';
	}

	export function setContent(html) {
		if (editor) editor.commands.setContent(html);
	}
</script>

{#if editor}
<div class="border-b border-gray-200 bg-gray-50 px-2 py-1 flex flex-wrap gap-1">
	<button
		class="px-2 py-1 rounded text-sm hover:bg-gray-200"
		class:bg-gray-300={editor.isActive('bold')}
		onclick={() => editor.chain().focus().toggleBold().run()}
	>B</button>
	<button
		class="px-2 py-1 rounded text-sm hover:bg-gray-200 italic"
		class:bg-gray-300={editor.isActive('italic')}
		onclick={() => editor.chain().focus().toggleItalic().run()}
	>I</button>
	<button
		class="px-2 py-1 rounded text-sm hover:bg-gray-200 line-through"
		class:bg-gray-300={editor.isActive('strike')}
		onclick={() => editor.chain().focus().toggleStrike().run()}
	>S</button>
	<span class="border-l border-gray-300 mx-1"></span>
	<button
		class="px-2 py-1 rounded text-sm hover:bg-gray-200"
		class:bg-gray-300={editor.isActive('heading', { level: 1 })}
		onclick={() => editor.chain().focus().toggleHeading({ level: 1 }).run()}
	>H1</button>
	<button
		class="px-2 py-1 rounded text-sm hover:bg-gray-200"
		class:bg-gray-300={editor.isActive('heading', { level: 2 })}
		onclick={() => editor.chain().focus().toggleHeading({ level: 2 }).run()}
	>H2</button>
	<button
		class="px-2 py-1 rounded text-sm hover:bg-gray-200"
		class:bg-gray-300={editor.isActive('heading', { level: 3 })}
		onclick={() => editor.chain().focus().toggleHeading({ level: 3 }).run()}
	>H3</button>
	<span class="border-l border-gray-300 mx-1"></span>
	<button
		class="px-2 py-1 rounded text-sm hover:bg-gray-200"
		class:bg-gray-300={editor.isActive('bulletList')}
		onclick={() => editor.chain().focus().toggleBulletList().run()}
	>List</button>
	<button
		class="px-2 py-1 rounded text-sm hover:bg-gray-200"
		class:bg-gray-300={editor.isActive('orderedList')}
		onclick={() => editor.chain().focus().toggleOrderedList().run()}
	>1.</button>
	<button
		class="px-2 py-1 rounded text-sm hover:bg-gray-200"
		class:bg-gray-300={editor.isActive('blockquote')}
		onclick={() => editor.chain().focus().toggleBlockquote().run()}
	>Quote</button>
	<button
		class="px-2 py-1 rounded text-sm hover:bg-gray-200"
		class:bg-gray-300={editor.isActive('codeBlock')}
		onclick={() => editor.chain().focus().toggleCodeBlock().run()}
	>Code</button>
	<span class="border-l border-gray-300 mx-1"></span>
	<button
		class="px-2 py-1 rounded text-sm hover:bg-gray-200"
		onclick={() => editor.chain().focus().setHorizontalRule().run()}
	>HR</button>
</div>
{/if}

<div bind:this={element} class="tiptap"></div>
