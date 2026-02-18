为员工增加下属，当用户明确要求时使用

Parameters:
- username (required): The username of the existing user to add as subordinate
- nickname (optional): Display name/nickname to set for the user. If provided, updates the user's name.

Example usage:
- "添加张三为下属" -> username="张三", nickname=""
- "把李四加为下属，昵称叫小李" -> username="李四", nickname="小李"
- "Add userjohn as my subordinate with nickname John Doe" -> username="userjohn", nickname="John Doe"

Error cases:
- User is not the company creator
- Target user does not exist
- User already a subordinate
