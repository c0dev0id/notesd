package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/c0dev0id/notesd/notes-cli/internal/client"
	"github.com/c0dev0id/notesd/notes-cli/internal/store"
	"github.com/c0dev0id/notesd/notes-cli/internal/sync"
	"github.com/spf13/cobra"
)

var cl *client.Client
var st *store.Store
var sy *sync.Syncer

var rootCmd = &cobra.Command{
	Use:          "notes-cli",
	Short:        "notes-cli — offline-first notes and todo client",
	SilenceUsage: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Name() == "login" || cmd.Name() == "register" {
			return nil
		}
		var err error
		cl, err = client.New()
		if err != nil {
			return err
		}
		if !cl.IsLoggedIn() && cmd.Name() != "help" {
			return fmt.Errorf("not logged in — run: notes-cli login")
		}

		dbPath := filepath.Join(cl.ConfigDir(), "notes.db")
		st, err = store.Open(dbPath)
		if err != nil {
			return fmt.Errorf("open local store: %w", err)
		}
		sy = sync.New(st, cl, userID())
		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(registerCmd)
	rootCmd.AddCommand(logoutCmd)
	rootCmd.AddCommand(notesCmd)
	rootCmd.AddCommand(todosCmd)
	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(syncCmd)
}

func userID() string {
	if cl == nil || cl.SessionInfo() == nil {
		return ""
	}
	return cl.SessionInfo().UserID
}

// syncQuietly runs a sync after a write command. Errors go to stderr; success
// is silent so as not to clutter command output.
func syncQuietly() {
	if sy == nil {
		return
	}
	if _, err := sy.Sync(); err != nil {
		fmt.Fprintf(os.Stderr, "sync: %v\n", err)
	}
}
