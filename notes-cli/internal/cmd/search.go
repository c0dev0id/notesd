package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search notes by title and content",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runSearch,
}

func init() {
	searchCmd.Flags().IntP("limit", "l", 20, "Number of results")
}

func runSearch(cmd *cobra.Command, args []string) error {
	query := strings.Join(args, " ")
	limit, _ := cmd.Flags().GetInt("limit")

	notes, total, err := st.SearchNotes(userID(), query, limit, 0)
	if err != nil {
		return err
	}
	if len(notes) == 0 {
		fmt.Println("No results.")
		return nil
	}
	fmt.Printf("Found %d notes matching %q:\n\n", total, query)
	for _, n := range notes {
		title := n.Title
		if title == "" {
			title = "(untitled)"
		}
		fmt.Printf("%-38s  %s  %s\n", n.ID, n.ModifiedAt.Local().Format("2006-01-02"), title)
	}
	return nil
}
