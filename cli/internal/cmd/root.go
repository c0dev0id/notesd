package cmd

import (
	"fmt"
	"os"

	"github.com/c0dev0id/notesd/cli/internal/client"
	"github.com/spf13/cobra"
)

var cl *client.Client

var rootCmd = &cobra.Command{
	Use:   "notesd",
	Short: "notesd CLI client",
	Long:  "Command-line client for the notesd notes and todo server.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip client init for login/register (they set up the client themselves)
		if cmd.Name() == "login" || cmd.Name() == "register" {
			return nil
		}
		var err error
		cl, err = client.New()
		if err != nil {
			return err
		}
		if !cl.IsLoggedIn() && cmd.Name() != "help" {
			return fmt.Errorf("not logged in — run: notesd login")
		}
		return nil
	},
	SilenceUsage: true,
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
}

// requireLogin is a helper for commands that need auth.
func requireLogin() error {
	if cl == nil || !cl.IsLoggedIn() {
		return fmt.Errorf("not logged in — run: notesd login")
	}
	return nil
}
