# Vikunja API Utils

Vikunja API工具函数集，用于快速调用服务器基础API接口。

## API函数

### login(base_url, username, password)
登录并获取JWT token，返回token字符串。

### create_user(base_url, username, email, password)
创建新用户，成功返回用户ID，失败返回错误信息。

### get_user(base_url, token)
获取当前用户信息，成功返回用户字典，失败返回错误信息。

### update_user_settings(base_url, token, name)
更新用户设置，设置名称、可发现性和语言，成功返回True。

### create_project(base_url, token, title)
创建新项目，成功返回项目ID，失败返回错误信息。

### create_team(base_url, token, name)
创建新团队，成功返回团队ID，失败返回错误信息。

### share_project_with_user(base_url, token, project_id, username, permission)
共享项目给用户，权限0=只读,1=读写,2=管理员。

### add_team_member(base_url, token, team_id, username)
添加用户到团队，成功返回True，失败返回错误信息。

## 使用示例

```python
from api_utils import login, create_user, create_project

# 登录
token = login('http://127.0.0.1:3456', 'username', 'password')

# 创建用户
user_id = create_user('http://127.0.0.1:3456', 'newuser', 'user@example.com', 'password')

# 创建项目
project_id = create_project('http://127.0.0.1:3456', token, 'My Project')
```

## 测试

```bash
python api_utils.py <test_name>
```

可用测试: login, get_user, create_user, update_user_settings, create_project, create_team, share_project, add_team_member
