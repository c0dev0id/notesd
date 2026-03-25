package cmd

import (
	"fmt"

	internalsync "github.com/c0dev0id/notesd/notes-cli/internal/sync"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Synchronise local store with the server",
	Long: `Pull server changes, push local changes, and resolve any conflicts.
Prints a detailed summary of what was transferred.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		result, err := sy.Sync()
		if err != nil {
			return fmt.Errorf("sync failed: %w", err)
		}
		fmt.Println(internalsync.FormatResult(result))
		return nil
	},
}
