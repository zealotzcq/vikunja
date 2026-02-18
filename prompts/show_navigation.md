Display a navigation button for user to navigate to various locations.

Use this tool when you want to provide a button that allows users to navigate to:
- Tasks (route: task.detail with param id)
- Projects (route: project.index with param projectId)
- Project list (route: projects.index)
- Home (route: home) - all task list, higher priority than upcoming page
- Favourite task list (route: project.index with param projectId = -1)
- Upcomming task list (route: tasks.range with param showNulls=true)

The button will display a label and optionally a title showing the target (e.g., task title, project title).

Parameters:
- route_name (required)
- params (optional)
- label (required)
- title (optional)

Example usage:
- Show task button: route_name="task.detail", params={"id": 1}, label="查看任务1", title="任务1:报告"
- Navigate to project: route_name="project.index", params={"projectId": 4}, label="查看项目4", title="项目4:盘古"