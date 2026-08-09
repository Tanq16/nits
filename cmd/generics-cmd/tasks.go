package genericsCmd

import (
	"strconv"

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
		if err := generics.TasksList(tasksFlags.done, tasksFlags.filter); err != nil {
			u.PrintFatal("failed to list tasks", err)
		}
	},
}

var tasksAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new task interactively",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if err := generics.TasksAdd(); err != nil {
			u.PrintFatal("failed to add task", err)
		}
	},
}

var tasksDoneCmd = &cobra.Command{
	Use:   "done [id]",
	Short: "Mark a task as done by its ID (interactive if ID omitted)",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
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
			if err := generics.TasksDone(pending[idx].ID); err != nil {
				u.PrintFatal("failed to mark task done", err)
			}
			return
		}

		id, err := strconv.Atoi(args[0])
		if err != nil {
			u.PrintFatal("invalid task ID", nil)
		}
		if err := generics.TasksDone(id); err != nil {
			u.PrintFatal("failed to mark task done", err)
		}
	},
}

var tasksDeleteCmd = &cobra.Command{
	Use:   "delete [id]",
	Short: "Delete a task by its ID (interactive if ID omitted)",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
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
			if err := generics.TasksDelete(all[idx].ID); err != nil {
				u.PrintFatal("failed to delete task", err)
			}
			return
		}

		id, err := strconv.Atoi(args[0])
		if err != nil {
			u.PrintFatal("invalid task ID", nil)
		}
		if err := generics.TasksDelete(id); err != nil {
			u.PrintFatal("failed to delete task", err)
		}
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
