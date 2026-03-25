package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/c0dev0id/notesd/notes-cli/internal/model"
	"github.com/spf13/cobra"
)

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

	notes, total, err := st.ListNotes(userID(), limit, offset)
	if err != nil {
		return err
	}
	if len(notes) == 0 {
		fmt.Println("No notes.")
		return nil
	}
	for _, n := range notes {
		title := n.Title
		if title == "" {
			title = "(untitled)"
		}
		fmt.Printf("%-38s  %-6s  %s  %s\n",
			n.ID, n.Type, n.ModifiedAt.Local().Format("2006-01-02 15:04"), title)
	}
	if total > offset+len(notes) {
		fmt.Printf("\nShowing %d-%d of %d notes\n", offset+1, offset+len(notes), total)
	}
	return nil
}

func runNotesShow(cmd *cobra.Command, args []string) error {
	n, err := st.GetNote(args[0], userID())
	if err != nil {
		return err
	}
	fmt.Printf("ID:       %s\n", n.ID)
	fmt.Printf("Title:    %s\n", n.Title)
	fmt.Printf("Type:     %s\n", n.Type)
	fmt.Printf("Modified: %s\n", n.ModifiedAt.Local().Format(time.RFC3339))
	fmt.Printf("Created:  %s\n", n.CreatedAt.Local().Format(time.RFC3339))
	if n.Content != "" {
		fmt.Println()
		fmt.Println(n.Content)
	}
	return nil
}

func runNotesCreate(cmd *cobra.Command, args []string) error {
	title, _ := cmd.Flags().GetString("title")
	content, _ := cmd.Flags().GetString("content")
	noteType, _ := cmd.Flags().GetString("type")

	if content == "" && title == "" {
		var err error
		title, content, err = editInEditor("", "")
		if err != nil {
			return err
		}
	}

	now := model.NowMillis()
	n := &model.Note{
		ID:               model.NewID(),
		UserID:           userID(),
		Title:            title,
		Content:          content,
		Type:             noteType,
		ModifiedAt:       now,
		ModifiedByDevice: cl.DeviceID(),
		CreatedAt:        now,
	}
	if err := st.CreateNote(n); err != nil {
		return err
	}
	fmt.Printf("Created note %s\n", n.ID)
	go syncQuietly()
	return nil
}

func runNotesEdit(cmd *cobra.Command, args []string) error {
	n, err := st.GetNote(args[0], userID())
	if err != nil {
		return err
	}
	newTitle, newContent, err := editInEditor(n.Title, n.Content)
	if err != nil {
		return err
	}
	if newTitle == n.Title && newContent == n.Content {
		fmt.Println("No changes.")
		return nil
	}
	n.Title = newTitle
	n.Content = newContent
	n.ModifiedAt = model.NowMillis()
	n.ModifiedByDevice = cl.DeviceID()
	if err := st.UpdateNote(n); err != nil {
		return err
	}
	fmt.Printf("Updated note %s\n", n.ID)
	go syncQuietly()
	return nil
}

func runNotesDelete(cmd *cobra.Command, args []string) error {
	now := model.NowMillis()
	if err := st.DeleteNote(args[0], userID(), now.UnixMilli(), cl.DeviceID()); err != nil {
		return err
	}
	fmt.Printf("Deleted note %s\n", args[0])
	go syncQuietly()
	return nil
}

// editInEditor opens $EDITOR with note content in the format:
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

	initialData, _ := os.ReadFile(tmpPath)

	c := exec.Command(editor, tmpPath)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		return "", "", fmt.Errorf("editor: %w", err)
	}

	data, err := os.ReadFile(tmpPath)
	if err != nil {
		return "", "", fmt.Errorf("read temp file: %w", err)
	}
	if string(data) == string(initialData) {
		return title, content, nil
	}
	return parseEditorContent(string(data))
}

func parseEditorContent(s string) (string, string, error) {
	parts := strings.SplitN(s, "\n---\n", 2)
	title := strings.TrimSpace(strings.TrimPrefix(parts[0], "Title: "))
	var content string
	if len(parts) == 2 {
		content = parts[1]
	}
	return title, content, nil
}
