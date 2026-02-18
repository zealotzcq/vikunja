Add an existing user as a subordinate. Use this when the user requests to add a subordinate/staff member or add someone to the company.

This tool performs the following operations:
1. Validates that the current user is the company creator
2. Checks if the target user exists in the system
3. Adds the target user to the company as staff (if not already a member)
4. Creates a new project for managing the subordinate's tasks (title format: "to {username}")
5. Shares the project with the subordinate (Read & Write permission)
6. Creates a supervisor-subordinate relationship in the company
7. Optionally updates the subordinate's display name/nickname and makes them discoverable

Parameters:
- username (required): The username of the existing user to add as subordinate
- nickname (optional): Display name/nickname to set for the user. If provided, updates the user's name and makes them discoverable by name and email.

Important notes:
- Only the company creator can add subordinates
- The target user must already exist in the system (this tool does NOT create new users)
- If the user is already in the company but not a subordinate, they will be promoted
- If the user is already a subordinate, the tool will return the existing relationship info
- A new project will be created for task management between the supervisor and subordinate

Example usage:
- "添加张三为下属" -> username="张三", nickname=""
- "把李四加为下属，昵称叫小李" -> username="李四", nickname="小李"
- "Add userjohn as my subordinate with nickname John Doe" -> username="userjohn", nickname="John Doe"

Error cases:
- User is not the company creator: Returns error "Only the company creator can add subordinates"
- Target user does not exist: Returns error "User '{username}' does not exist"
- User already a subordinate: Returns success message with existing project info
