package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

type Note struct {
	ID               string     `json:"id"`
	Title            string     `json:"title"`
	Content          string     `json:"content"`
	Type             string     `json:"type"`
	ModifiedAt       time.Time  `json:"modified_at"`
	ModifiedByDevice string     `json:"modified_by_device"`
	DeletedAt        *time.Time `json:"deleted_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
}

type NoteListResponse struct {
	Notes  []Note `json:"notes"`
	Total  int    `json:"total"`
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
}

var notesCmd = &cobra.Command{
	Use:   "notes",
	Short: "Manage notes",
}

var notesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all notes",
	RunE:  runNotesList,
}

var notesShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show a note",
	Args:  cobra.ExactArgs(1),
	RunE:  runNotesShow,
}

var notesCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new note",
	RunE:  runNotesCreate,
}

var notesEditCmd = &cobra.Command{
	Use:   "edit <id>",
	Short: "Edit a note in $EDITOR",
	Args:  cobra.ExactArgs(1),
	RunE:  runNotesEdit,
}

var notesDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a note",
	Args:  cobra.ExactArgs(1),
	RunE:  runNotesDelete,
}

func init() {
	notesCmd.AddCommand(notesListCmd, notesShowCmd, notesCreateCmd, notesEditCmd, notesDeleteCmd)

	notesListCmd.Flags().IntP("limit", "l", 20, "Number of notes to show")
	notesListCmd.Flags().IntP("offset", "o", 0, "Offset for pagination")

	notesCreateCmd.Flags().StringP("title", "t", "", "Note title")
	notesCreateCmd.Flags().StringP("content", "c", "", "Note content")
	notesCreateCmd.Flags().String("type", "note", "Note type (note, todo_list)")
}

func runNotesList(cmd *cobra.Command, args []string) error {
	limit, _ := cmd.Flags().GetInt("limit")
	offset, _ := cmd.Flags().GetInt("offset")

	var resp NoteListResponse
	status, err := cl.DoJSON("GET", fmt.Sprintf("/api/v1/notes?limit=%d&offset=%d", limit, offset), nil, &resp)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("unexpected status %d", status)
	}

	if len(resp.Notes) == 0 {
		fmt.Println("No notes.")
		return nil
	}

	for _, n := range resp.Notes {
		title := n.Title
		if title == "" {
			title = "(untitled)"
		}
		fmt.Printf("%-38s  %-6s  %s  %s\n",
			n.ID, n.Type, n.ModifiedAt.Local().Format("2006-01-02 15:04"), title)
	}
	if resp.Total > resp.Offset+len(resp.Notes) {
		fmt.Printf("\nShowing %d-%d of %d notes\n",
			resp.Offset+1, resp.Offset+len(resp.Notes), resp.Total)
	}
	return nil
}

func runNotesShow(cmd *cobra.Command, args []string) error {
	var note Note
	status, err := cl.DoJSON("GET", "/api/v1/notes/"+args[0], nil, &note)
	if err != nil {
		return err
	}
	if status == http.StatusNotFound {
		return fmt.Errorf("note not found")
	}

	fmt.Printf("ID:       %s\n", note.ID)
	fmt.Printf("Title:    %s\n", note.Title)
	fmt.Printf("Type:     %s\n", note.Type)
	fmt.Printf("Modified: %s\n", note.ModifiedAt.Local().Format(time.RFC3339))
	fmt.Printf("Created:  %s\n", note.CreatedAt.Local().Format(time.RFC3339))
	if note.Content != "" {
		fmt.Println()
		fmt.Println(note.Content)
	}
	return nil
}

func runNotesCreate(cmd *cobra.Command, args []string) error {
	title, _ := cmd.Flags().GetString("title")
	content, _ := cmd.Flags().GetString("content")
	noteType, _ := cmd.Flags().GetString("type")

	// If no content flag, open $EDITOR
	if content == "" && title == "" {
		var err error
		title, content, err = editInEditor("", "")
		if err != nil {
			return err
		}
	}

	req := map[string]string{
		"title":     title,
		"content":   content,
		"type":      noteType,
		"device_id": cl.DeviceID,
	}

	var note Note
	status, err := cl.DoJSON("POST", "/api/v1/notes", req, &note)
	if err != nil {
		return err
	}
	if status != http.StatusCreated {
		return fmt.Errorf("unexpected status %d", status)
	}

	fmt.Printf("Created note %s\n", note.ID)
	return nil
}

func runNotesEdit(cmd *cobra.Command, args []string) error {
	// Fetch current note
	var note Note
	status, err := cl.DoJSON("GET", "/api/v1/notes/"+args[0], nil, &note)
	if err != nil {
		return err
	}
	if status == http.StatusNotFound {
		return fmt.Errorf("note not found")
	}

	// Open in editor
	newTitle, newContent, err := editInEditor(note.Title, note.Content)
	if err != nil {
		return err
	}

	if newTitle == note.Title && newContent == note.Content {
		fmt.Println("No changes.")
		return nil
	}

	req := map[string]any{
		"title":     newTitle,
		"content":   newContent,
		"device_id": cl.DeviceID,
	}

	var updated Note
	status, err = cl.DoJSON("PUT", "/api/v1/notes/"+args[0], req, &updated)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("unexpected status %d", status)
	}

	fmt.Printf("Updated note %s\n", updated.ID)
	return nil
}

func runNotesDelete(cmd *cobra.Command, args []string) error {
	status, err := cl.DoJSON("DELETE", "/api/v1/notes/"+args[0], nil, nil)
	if err != nil {
		return err
	}
	if status == http.StatusNotFound {
		return fmt.Errorf("note not found")
	}
	if status != http.StatusNoContent {
		return fmt.Errorf("unexpected status %d", status)
	}
	fmt.Printf("Deleted note %s\n", args[0])
	return nil
}

// editInEditor opens $EDITOR with a note in a simple text format:
//
//	Title: <title>
//	---
//	<content>
func editInEditor(title, content string) (string, string, error) {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		editor = "vi"
	}

	tmpfile, err := os.CreateTemp("", "notesd-*.md")
	if err != nil {
		return "", "", fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmpfile.Name()
	defer os.Remove(tmpPath)

	initial := fmt.Sprintf("Title: %s\n---\n%s", title, content)
	if _, err := tmpfile.WriteString(initial); err != nil {
		tmpfile.Close()
		return "", "", err
	}
	tmpfile.Close()

	// Get initial checksum
	initialData, _ := os.ReadFile(tmpPath)

	cmd := exec.Command(editor, tmpPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", "", fmt.Errorf("editor: %w", err)
	}

	data, err := os.ReadFile(tmpPath)
	if err != nil {
		return "", "", fmt.Errorf("read temp file: %w", err)
	}

	// Check if unchanged
	if string(data) == string(initialData) {
		return title, content, nil
	}

	return parseEditorContent(string(data))
}

func parseEditorContent(s string) (string, string, error) {
	// Format: "Title: <title>\n---\n<content>"
	parts := strings.SplitN(s, "\n---\n", 2)
	if len(parts) < 1 {
		return "", "", fmt.Errorf("invalid format")
	}

	titleLine := parts[0]
	title := strings.TrimPrefix(titleLine, "Title: ")
	title = strings.TrimSpace(title)

	var content string
	if len(parts) == 2 {
		content = parts[1]
	}

	return title, content, nil
}

// printNoteJSON outputs a note as formatted JSON (for piping).
func printNoteJSON(note Note) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(note)
}
