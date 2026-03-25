package cmd

import (
	"fmt"
	"net/http"
	"time"

	"github.com/spf13/cobra"
)

type Todo struct {
	ID               string     `json:"id"`
	Content          string     `json:"content"`
	Completed        bool       `json:"completed"`
	DueDate          *time.Time `json:"due_date,omitempty"`
	NoteID           *string    `json:"note_id,omitempty"`
	ModifiedAt       time.Time  `json:"modified_at"`
	ModifiedByDevice string     `json:"modified_by_device"`
	CreatedAt        time.Time  `json:"created_at"`
}

type TodoListResponse struct {
	Todos  []Todo `json:"todos"`
	Total  int    `json:"total"`
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
}

var todosCmd = &cobra.Command{
	Use:   "todos",
	Short: "Manage todos",
}

var todosListCmd = &cobra.Command{
	Use:   "list",
	Short: "List todos",
	RunE:  runTodosList,
}

var todosShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show a todo",
	Args:  cobra.ExactArgs(1),
	RunE:  runTodosShow,
}

var todosCreateCmd = &cobra.Command{
	Use:   "create <content>",
	Short: "Create a todo",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runTodosCreate,
}

var todosCompleteCmd = &cobra.Command{
	Use:   "complete <id>",
	Short: "Mark a todo as completed",
	Args:  cobra.ExactArgs(1),
	RunE:  runTodosComplete,
}

var todosDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a todo",
	Args:  cobra.ExactArgs(1),
	RunE:  runTodosDelete,
}

func init() {
	todosCmd.AddCommand(todosListCmd, todosShowCmd, todosCreateCmd, todosCompleteCmd, todosDeleteCmd)

	todosListCmd.Flags().Bool("overdue", false, "Show only overdue todos")
	todosListCmd.Flags().IntP("limit", "l", 20, "Number of todos to show")
	todosListCmd.Flags().IntP("offset", "o", 0, "Offset for pagination")

	todosCreateCmd.Flags().StringP("due", "d", "", "Due date (YYYY-MM-DD)")
	todosCreateCmd.Flags().String("note", "", "Attach to note ID")
}

func runTodosList(cmd *cobra.Command, args []string) error {
	overdue, _ := cmd.Flags().GetBool("overdue")

	if overdue {
		var todos []Todo
		status, err := cl.DoJSON("GET", "/api/v1/todos/overdue", nil, &todos)
		if err != nil {
			return err
		}
		if status != http.StatusOK {
			return fmt.Errorf("unexpected status %d", status)
		}
		if len(todos) == 0 {
			fmt.Println("No overdue todos.")
			return nil
		}
		printTodos(todos)
		return nil
	}

	limit, _ := cmd.Flags().GetInt("limit")
	offset, _ := cmd.Flags().GetInt("offset")

	var resp TodoListResponse
	status, err := cl.DoJSON("GET", fmt.Sprintf("/api/v1/todos?limit=%d&offset=%d", limit, offset), nil, &resp)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("unexpected status %d", status)
	}

	if len(resp.Todos) == 0 {
		fmt.Println("No todos.")
		return nil
	}

	printTodos(resp.Todos)
	if resp.Total > resp.Offset+len(resp.Todos) {
		fmt.Printf("\nShowing %d-%d of %d todos\n",
			resp.Offset+1, resp.Offset+len(resp.Todos), resp.Total)
	}
	return nil
}

func runTodosShow(cmd *cobra.Command, args []string) error {
	var todo Todo
	status, err := cl.DoJSON("GET", "/api/v1/todos/"+args[0], nil, &todo)
	if err != nil {
		return err
	}
	if status == http.StatusNotFound {
		return fmt.Errorf("todo not found")
	}

	check := "[ ]"
	if todo.Completed {
		check = "[x]"
	}
	fmt.Printf("ID:        %s\n", todo.ID)
	fmt.Printf("Status:    %s\n", check)
	fmt.Printf("Content:   %s\n", todo.Content)
	if todo.DueDate != nil {
		fmt.Printf("Due:       %s\n", todo.DueDate.Local().Format("2006-01-02"))
	}
	if todo.NoteID != nil {
		fmt.Printf("Note:      %s\n", *todo.NoteID)
	}
	fmt.Printf("Modified:  %s\n", todo.ModifiedAt.Local().Format(time.RFC3339))
	fmt.Printf("Created:   %s\n", todo.CreatedAt.Local().Format(time.RFC3339))
	return nil
}

func runTodosCreate(cmd *cobra.Command, args []string) error {
	content := joinArgs(args)

	req := map[string]any{
		"content":   content,
		"device_id": cl.DeviceID,
	}

	dueStr, _ := cmd.Flags().GetString("due")
	if dueStr != "" {
		due, err := time.Parse("2006-01-02", dueStr)
		if err != nil {
			return fmt.Errorf("invalid due date (use YYYY-MM-DD): %w", err)
		}
		req["due_date"] = due.UTC()
	}

	noteID, _ := cmd.Flags().GetString("note")
	if noteID != "" {
		req["note_id"] = noteID
	}

	var todo Todo
	status, err := cl.DoJSON("POST", "/api/v1/todos", req, &todo)
	if err != nil {
		return err
	}
	if status != http.StatusCreated {
		return fmt.Errorf("unexpected status %d", status)
	}

	fmt.Printf("Created todo %s\n", todo.ID)
	return nil
}

func runTodosComplete(cmd *cobra.Command, args []string) error {
	req := map[string]any{
		"completed": true,
		"device_id": cl.DeviceID,
	}

	var todo Todo
	status, err := cl.DoJSON("PUT", "/api/v1/todos/"+args[0], req, &todo)
	if err != nil {
		return err
	}
	if status == http.StatusNotFound {
		return fmt.Errorf("todo not found")
	}
	if status != http.StatusOK {
		return fmt.Errorf("unexpected status %d", status)
	}

	fmt.Printf("Completed: %s\n", todo.Content)
	return nil
}

func runTodosDelete(cmd *cobra.Command, args []string) error {
	status, err := cl.DoJSON("DELETE", "/api/v1/todos/"+args[0], nil, nil)
	if err != nil {
		return err
	}
	if status == http.StatusNotFound {
		return fmt.Errorf("todo not found")
	}
	if status != http.StatusNoContent {
		return fmt.Errorf("unexpected status %d", status)
	}
	fmt.Printf("Deleted todo %s\n", args[0])
	return nil
}

func printTodos(todos []Todo) {
	for _, t := range todos {
		check := "[ ]"
		if t.Completed {
			check = "[x]"
		}
		due := "          "
		if t.DueDate != nil {
			due = t.DueDate.Local().Format("2006-01-02")
		}
		fmt.Printf("%s  %s  %s  %s\n", check, t.ID, due, t.Content)
	}
}

func joinArgs(args []string) string {
	result := ""
	for i, a := range args {
		if i > 0 {
			result += " "
		}
		result += a
	}
	return result
}
