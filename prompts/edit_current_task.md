Edit the current task. Use this when user wants to modify properties of the task they're currently working with.

The current task is provided in the context with:
- Task ID
- Title
- Project ID

This tool supports modifying:
1. Assignee: Change the person assigned to the task. When changing the assignee, the task will be automatically moved to that person's associated project.
2. Title: Update the task title.
3. Description: Update the task description/content.
4. Due date: Change when the task is due.

Parameters:
- user_identifier: Optional. The user ID (integer) of the person to assign the task to. If provided, the task will be reassigned and moved to that person's project.
- title: Optional. The new title for the task.
- description: Optional. The new description/content for the task.
- time_expression: Optional. A natural language time expression (e.g., 'tomorrow', 'Next Friday', '2 hours later', '2024-12-25'). If provided, this updates the task's due date.

Note: At least one of the parameters must be provided. You can combine multiple parameters in one call.

When modifying assignee:
- The task will be moved to the assignee's project (retrieved from SubordinateStaff or current user's project)
- All existing assignees will be replaced with the new assignee
- The project_id will be updated accordingly

Current date/time is {{ .CurrentDate }}.

Example usage:
- "Change this task's title to 'Write report'" -> {"title": "Write report"}
- "Reassign this task to 小王" -> {"user_identifier": 123}
- "Update description to include more details" -> {"description": "New detailed description"}
- "Set due date to tomorrow" -> {"time_expression": "tomorrow"}
- "Assign to 小李 and change title" -> {"user_identifier": 456, "title": "New Title"}
