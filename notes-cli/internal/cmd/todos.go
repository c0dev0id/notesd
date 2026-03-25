package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/c0dev0id/notesd/notes-cli/internal/model"
	"github.com/spf13/cobra"
)

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
		todos, err := st.GetOverdueTodos(userID())
		if err != nil {
			return err
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
	todos, total, err := st.ListTodos(userID(), limit, offset)
	if err != nil {
		return err
	}
	if len(todos) == 0 {
		fmt.Println("No todos.")
		return nil
	}
	printTodos(todos)
	if total > offset+len(todos) {
		fmt.Printf("\nShowing %d-%d of %d todos\n", offset+1, offset+len(todos), total)
	}
	return nil
}

func runTodosShow(cmd *cobra.Command, args []string) error {
	t, err := st.GetTodo(args[0], userID())
	if err != nil {
		return err
	}
	check := "[ ]"
	if t.Completed {
		check = "[x]"
	}
	fmt.Printf("ID:        %s\n", t.ID)
	fmt.Printf("Status:    %s\n", check)
	fmt.Printf("Content:   %s\n", t.Content)
	if t.DueDate != nil {
		fmt.Printf("Due:       %s\n", t.DueDate.Local().Format("2006-01-02"))
	}
	if t.NoteID != nil {
		fmt.Printf("Note:      %s\n", *t.NoteID)
	}
	fmt.Printf("Modified:  %s\n", t.ModifiedAt.Local().Format(time.RFC3339))
	fmt.Printf("Created:   %s\n", t.CreatedAt.Local().Format(time.RFC3339))
	return nil
}

func runTodosCreate(cmd *cobra.Command, args []string) error {
	content := strings.Join(args, " ")

	now := model.NowMillis()
	t := &model.Todo{
		ID:               model.NewID(),
		UserID:           userID(),
		Content:          content,
		ModifiedAt:       now,
		ModifiedByDevice: cl.DeviceID(),
		CreatedAt:        now,
	}

	dueStr, _ := cmd.Flags().GetString("due")
	if dueStr != "" {
		due, err := time.Parse("2006-01-02", dueStr)
		if err != nil {
			return fmt.Errorf("invalid due date (use YYYY-MM-DD): %w", err)
		}
		t.DueDate = &due
	}

	noteID, _ := cmd.Flags().GetString("note")
	if noteID != "" {
		t.NoteID = &noteID
	}

	if err := st.CreateTodo(t); err != nil {
		return err
	}
	fmt.Printf("Created todo %s\n", t.ID)
	go syncQuietly()
	return nil
}

func runTodosComplete(cmd *cobra.Command, args []string) error {
	t, err := st.GetTodo(args[0], userID())
	if err != nil {
		return err
	}
	t.Completed = true
	t.ModifiedAt = model.NowMillis()
	t.ModifiedByDevice = cl.DeviceID()
	if err := st.UpdateTodo(t); err != nil {
		return err
	}
	fmt.Printf("Completed: %s\n", t.Content)
	go syncQuietly()
	return nil
}

func runTodosDelete(cmd *cobra.Command, args []string) error {
	now := model.NowMillis()
	if err := st.DeleteTodo(args[0], userID(), now.UnixMilli(), cl.DeviceID()); err != nil {
		return err
	}
	fmt.Printf("Deleted todo %s\n", args[0])
	go syncQuietly()
	return nil
}

func printTodos(todos []model.Todo) {
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
