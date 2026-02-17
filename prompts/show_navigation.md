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
- route_name (required): The name of route to navigate to
- params (optional): Route parameters (e.g., id for task, projectId for project)
- label (required): The button label text (e.g., taskid,projectid)
- title (optional): The title of the target entity to display after the label (e.g., taskid: task title, projectid: project title)

Example usage:
- Show task button: route_name="task.detail", params={"id": 123}, label="查看任务123", title="任务123:写报告"
- Navigate to project: route_name="project.index", params={"projectId": 456}, label="查看项目456", title="项目456:盘古计划"
- Navigate to home: route_name="home", label="回到主页"
