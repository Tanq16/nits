package genericsCmd

import (
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/cobra"
	"github.com/tanq16/nits/internal/generics"
	u "github.com/tanq16/nits/utils"
)

var tasksFlags struct {
	done   bool
	filter string
}

var TasksCmd = &cobra.Command{
	Use:   "tasks",
	Short: "Lightweight personal task tracking with pending/done status",
}

var tasksListCmd = &cobra.Command{
	Use:   "list",
	Short: "List tasks (pending by default, use --done to include completed)",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		tasks, err := generics.GetFilteredTasks(tasksFlags.done, tasksFlags.filter)
		if err != nil {
			u.PrintFatal("failed to list tasks", err)
		}
		if len(tasks) == 0 {
			if tasksFlags.filter != "" {
				u.PrintInfo("No matching tasks")
			} else {
				u.PrintInfo("No tasks found")
			}
			return
		}

		var headers []string
		if tasksFlags.done {
			headers = []string{"ID", "Task", "Status", "Added"}
		} else {
			headers = []string{"ID", "Task", "Added"}
		}

		var rows [][]string
		for _, task := range tasks {
			timeAgo := generics.FormatTimeAgo(time.Since(task.CreatedAt))
			if tasksFlags.done {
				rows = append(rows, []string{
					strconv.Itoa(task.ID),
					task.Task,
					string(task.Status),
					timeAgo,
				})
			} else {
				rows = append(rows, []string{
					strconv.Itoa(task.ID),
					task.Task,
					timeAgo,
				})
			}
		}

		u.PrintTable(headers, rows)
	},
}

var tasksAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new task interactively",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		task, err := u.PromptInput("Enter task:", "What needs to be done?")
		if err != nil {
			u.PrintFatal("failed to read input", err)
		}
		if task == "" {
			u.PrintFatal("no task provided", nil)
		}
		entry, err := generics.TasksAdd(task)
		if err != nil {
			u.PrintFatal("failed to add task", err)
		}
		u.PrintSuccess(fmt.Sprintf("%s → added (ID: %d)", task, entry.ID))
	},
}

var tasksDoneCmd = &cobra.Command{
	Use:   "done [id]",
	Short: "Mark a task as done by its ID (interactive if ID omitted)",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var id int
		if len(args) == 0 {
			pending, err := generics.GetPendingTasks()
			if err != nil {
				u.PrintFatal("failed to load tasks", err)
			}
			if len(pending) == 0 {
				u.PrintInfo("No pending tasks to mark done")
				return
			}
			options := make([]string, len(pending))
			for i, t := range pending {
				options[i] = strconv.Itoa(t.ID) + ": " + t.Task
			}
			idx, err := u.PromptSelect("Select task to mark done:", options)
			if err != nil {
				u.PrintFatal("failed to read selection", err)
			}
			if idx < 0 {
				return
			}
			id = pending[idx].ID
		} else {
			parsedID, err := strconv.Atoi(args[0])
			if err != nil {
				u.PrintFatal("invalid task ID", nil)
			}
			id = parsedID
		}

		if err := generics.TasksDone(id); err != nil {
			u.PrintFatal("failed to mark task done", err)
		}
		u.PrintSuccess(fmt.Sprintf("marked done (ID: %d)", id))
	},
}

var tasksDeleteCmd = &cobra.Command{
	Use:   "delete [id]",
	Short: "Delete a task by its ID (interactive if ID omitted)",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var id int
		if len(args) == 0 {
			all, err := generics.GetAllTasks()
			if err != nil {
				u.PrintFatal("failed to load tasks", err)
			}
			if len(all) == 0 {
				u.PrintInfo("No tasks to delete")
				return
			}
			options := make([]string, len(all))
			for i, t := range all {
				options[i] = strconv.Itoa(t.ID) + ": " + t.Task + " [" + string(t.Status) + "]"
			}
			idx, err := u.PromptSelect("Select task to delete:", options)
			if err != nil {
				u.PrintFatal("failed to read selection", err)
			}
			if idx < 0 {
				return
			}
			id = all[idx].ID
		} else {
			parsedID, err := strconv.Atoi(args[0])
			if err != nil {
				u.PrintFatal("invalid task ID", nil)
			}
			id = parsedID
		}

		if err := generics.TasksDelete(id); err != nil {
			u.PrintFatal("failed to delete task", err)
		}
		u.PrintSuccess(fmt.Sprintf("deleted (ID: %d)", id))
	},
}

func init() {
	tasksListCmd.Flags().BoolVar(&tasksFlags.done, "done", false, "Show completed tasks alongside pending")
	tasksListCmd.Flags().StringVar(&tasksFlags.filter, "filter", "", "Filter tasks by regex pattern")
	TasksCmd.AddCommand(tasksListCmd)
	TasksCmd.AddCommand(tasksAddCmd)
	TasksCmd.AddCommand(tasksDoneCmd)
	TasksCmd.AddCommand(tasksDeleteCmd)
}

