# User's Guide

## What is notesd?

notesd is a personal notes and todo application. You can create notes with rich
text formatting, embed todos inside notes, set due dates, and see what's overdue.

All your data stays on your devices. When an internet connection is available,
changes sync automatically across all your devices.

## Getting Started

### Creating an Account

Register with your email address, a password, and a display name. This account
is used to sync your notes between devices.

### Working Offline

notesd works fully offline. You can create, edit, and delete notes and todos
without an internet connection. When you reconnect, your changes will sync
automatically.

### Notes

Notes are the primary way to store information. Each note has a title and can
contain rich text content.

There are two types of notes:

- **Standard notes** — free-form text with formatting
- **Todo lists** — each line in the note is treated as a todo item

### Todos

Todos can exist on their own or be embedded within notes. Each todo can
optionally have a due date.

Overdue todos (past their due date and not yet completed) are highlighted so
you can stay on top of deadlines.

### Calendar

Todos with due dates appear in the calendar view, organized by date. The "today"
view shows both today's tasks and any overdue items.

### Sync and Conflicts

When the same note is edited on two devices while offline, the most recent edit
wins when sync happens. You won't lose data — the newer version is kept.

Deleted items are synced across all devices so removals propagate everywhere.

## Command-Line Interface

The CLI lets you manage notes and todos from the terminal.

### Setup

Register an account and log in:

```
notesd register -s http://your-server:8080
notesd login -s http://your-server:8080
```

After login, the server URL and credentials are stored in `~/.notesd/` and
reused for subsequent commands.

### Managing Notes

```
notesd notes list                   # list all notes
notesd notes create -t "Title"      # create with title
notesd notes create                 # create in $EDITOR
notesd notes show <id>              # display a note
notesd notes edit <id>              # edit in $EDITOR
notesd notes delete <id>            # delete a note
notesd search <query>               # search notes
```

### Managing Todos

```
notesd todos list                   # list all todos
notesd todos list --overdue         # show overdue only
notesd todos create "Buy groceries" # create a todo
notesd todos create "Task" -d 2026-03-15  # with due date
notesd todos complete <id>          # mark as done
notesd todos delete <id>            # delete a todo
```

### Logging Out

```
notesd logout
```

This revokes your tokens on the server and removes the local session.
