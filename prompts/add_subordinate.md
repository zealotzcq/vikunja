为员工增加下属，当用户明确要求时使用

Parameters:
- username (required)
- nickname (optional)

Example usage:
- "添加张三为下属" -> username="张三", nickname=""
- "把李四加为下属，昵称叫小李" -> username="李四", nickname="小李"

Error cases:
- User is not the company creator
- Target user does not exist
- User already a subordinate
