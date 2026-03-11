Display a navigation button to help users check progress or status.

Use this tool when users ask about status or progress of:
- A specific person's status → use that person's project ID
- Company-wide status → use projectId = -1
- User's own status → use user's own project ID

The button will display a label and optionally a title showing the target (e.g., person's name, company name).

Parameters:
- projectId (required)
- label (required)
- title (optional)

Example usage:
- Check a person's status: projectId=5, label="查看张三的状态", title="张三"
- Check company status: projectId=-1, label="查看公司整体状态", title="公司"
- Check own status: projectId=2, label="查看我的状态"