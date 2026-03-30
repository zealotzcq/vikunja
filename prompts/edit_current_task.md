Edit the current task. Use this when user wants to modify properties of the task they're currently working with.

The current task is provided in the context with:
- Task ID
- Title
- Project ID

This tool supports modifying:
1. Assignee: Change the person assigned to the task.
2. Title: Update the task title.
3. Description: Update the task description/content (replace or append).
4. Due date: Change when the task is due.

Parameters:
- user_identifier: Optional. The user ID (integer) of the person to assign the task to. If provided, the task will be reassigned to that person.
- title: Optional. The new title for the task.
- description: Optional. The description/content/detail for the task.
- append_description: Optional (default: false). If true, appends the description to existing content instead of replacing it. When appending, the new content is added after a blank line.
- time_expression: Optional. A natural language time expression (e.g., 'tomorrow', 'Next Friday', '2 hours later', '2024-12-25'). If provided, this updates the task's due date.

Note: At least one of the parameters must be provided. You can combine multiple parameters in one call.

When modifying assignee:
- Simply provide the user_identifier of the target person

When modifying description:
- Use append_description=false (default) to completely replace the existing description
- Use append_description=true to add content to the end of existing description
- When appending, the new content will be added after a blank line
- **IMPORTANT**: Process and refine the user's input before setting the description:
  - Keep the original semantic meaning intact
  - Correct typos, grammar, and punctuation errors
  - Improve tone and phrasing to be more professional and clear
  - Fix formatting issues (e.g., ensure proper line breaks with \n\n)
  - Do NOT add new information or change the intent of the user's input
  - If the user's input is already well-written, use it as-is or make minimal changes
  - Examples of refinement:
    * "这个任务要做完事" → "这个任务需要完成" (correcting typo 事→事)
    * "搞定这个东西" → "完成这项工作" (more professional tone)
    * "明天搞完" → "明天完成" (clearer expression)
    * "写个报告啥的" → "撰写报告" (more professional)

Current date/time is {{ .CurrentDate }}.

Example usage:
- "Change this task's title to 'Write report'" -> {"title": "Write report"}
- "Reassign this task to 小王" -> {"user_identifier": 123}
- "Replace description with new details" -> {"description": "New detailed description"}
- "Add more details to the description" -> {"description": "Additional details", "append_description": true}
- "Set due date to tomorrow" -> {"time_expression": "tomorrow"}
- "Assign to 小李 and change title" -> {"user_identifier": 456, "title": "New Title"}
