package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/c0dev0id/notesd/notes-cli/internal/client"
	"github.com/c0dev0id/notesd/notes-cli/internal/store"
	"github.com/c0dev0id/notesd/notes-cli/internal/tui"
)

func main() {
	cl, err := client.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "notes-tui: %v\n", err)
		os.Exit(1)
	}

	dbPath := filepath.Join(cl.ConfigDir(), "notes.db")
	st, err := store.Open(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "notes-tui: open store: %v\n", err)
		os.Exit(1)
	}
	defer st.Close()

	if err := tui.Run(cl, st); err != nil {
		fmt.Fprintf(os.Stderr, "notes-tui: %v\n", err)
		os.Exit(1)
	}
}
