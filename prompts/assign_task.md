Assign a task to a staff member or the user himself. Use this when user wants to assign work or a task to someone.

The system context contains information of subordinate staffs and the user, including:
- User ID, Username, Name, and Project ID for each subordinate

Priority determination (based on user's tone/phrasing):
- HIGH priority: When user says "马上", "立即", "尽快", "urgent", "immediately", etc.
- MEDIUM priority (default): Normal tone without urgency indicators
- LOW priority: When user says "有空", "有时间", "不急", "when convenient", "no rush", etc.

Due date calculation:
- If user specifies a time expression (e.g., "today", "tomorrow", "下周五", "2024-12-25"), use that expression
- If NO time expression is specified, use priority-based calculation:
  * HIGH priority: 1 day from now
  * MEDIUM priority: 3 days from now
  * LOW priority: 7 days from now

Task properties:
- Start date: Now (current time)
- IsFavorite: true (favorited by default)
- Subscription: Subscribed to task notifications
注意:填充time_expression时,指定语言为英语

Example usage:
- "让小王马上写报告" -> HIGH priority, no time_expr, due in 1 day
- "叫李四有空的时候整理文档" -> LOW priority, no time_expr, due in 7 days
- "给张三安排个任务，明天截止" -> MEDIUM priority, time_expr="tomorrow"
- "让小王下周五提交报告" -> MEDIUM priority, time_expr="Friday in next week"
- "给李四分配任务,2小时后完成" -> HIGH priority, time_expr="two hours later"
