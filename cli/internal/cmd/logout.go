package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Log out and revoke tokens",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := cl.Logout(); err != nil {
			return fmt.Errorf("logout: %w", err)
		}
		fmt.Println("Logged out.")
		return nil
	},
}
