Update a user's nickname/display name. Use this when user explicitly requests to change a nickname or display name.

This tool performs the following operations:
1. Validates that target user exists
2. Checks if current user is company creator
3. Updates the user's display name to the provided nickname
4. Sets the user as discoverable by name and email

Permissions:
- Only company creators can update nicknames of company members

Parameters:
- username (required): The username of the user whose nickname to update
- nickname (required): The new nickname/display name to set

Important notes:
- The target user must exist in the system
- Only company creators (creator role) can update member nicknames
- Admins and staff members cannot update nicknames, even their own
- Updating a nickname automatically makes the user discoverable by name and email
- If a user currently has no display name, their username will be shown as the old name

Example usage:
- "把张三的昵称改成小李" -> username="张三", nickname="小李"
- "把公司里李四的昵称改成Lisi" -> username="李四", nickname="Lisi"
- "Change userjohn's nickname to John Doe" -> username="userjohn", nickname="John Doe"
- "Update staff member's display name to Alice Smith" -> username="alice", nickname="Alice Smith"

Error cases:
- User does not exist: Returns error "User '{username}' does not exist"
- Not a creator: Returns error "Only company creator can update member nicknames."
- Empty nickname: Returns error "nickname is required"
