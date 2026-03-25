package cmd

import (
	"fmt"
	"net/http"
	"net/url"

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
	query := joinArgs(args)
	limit, _ := cmd.Flags().GetInt("limit")

	var resp NoteListResponse
	path := fmt.Sprintf("/api/v1/notes/search?q=%s&limit=%d", url.QueryEscape(query), limit)
	status, err := cl.DoJSON("GET", path, nil, &resp)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("unexpected status %d", status)
	}

	if len(resp.Notes) == 0 {
		fmt.Println("No results.")
		return nil
	}

	fmt.Printf("Found %d notes matching %q:\n\n", resp.Total, query)
	for _, n := range resp.Notes {
		title := n.Title
		if title == "" {
			title = "(untitled)"
		}
		fmt.Printf("%-38s  %s  %s\n", n.ID, n.ModifiedAt.Local().Format("2006-01-02"), title)
	}
	return nil
}
