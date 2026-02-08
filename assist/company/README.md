# Company Utility

Company Utility 是一个用于管理Vikunja系统中公司信息的Python工具。它通过操作SQLite数据库来管理公司和团队的相关信息。

## 功能

1. **初始化company表格** - 创建company表用于存储公司基本信息
2. **初始化company_relation表格** - 创建company_relation表用于存储公司上下级关系
3. **添加公司信息** - 向company表添加公司信息
4. **添加公司关系** - 向company_relation表添加上下级关系
5. **删除公司关系** - 从company_relation表删除上下级关系
6. **查询公司关系** - 查询公司上下级关系对应的项目ID
7. **查询公司信息** - 查询指定团队的公司信息和所有成员

## 数据库表结构

### company表
- `id` - 主键，自增
- `user_id` - 公司用户ID
- `username` - 登录用户名
- `team_id` - 对应的团队ID（唯一）
- `created` - 创建时间
- `updated` - 更新时间

### company_relation表
- `id` - 主键，自增
- `company_id` - 公司ID（外键关联company表）
- `superior_user_id` - 上级用户ID（外键关联users表）
- `subordinate_user_id` - 下级用户ID（外键关联users表）
- `project_id` - 对应的项目ID（外键关联projects表）
- `created` - 创建时间
- `updated` - 更新时间

## 使用方法

### 帮助信息
```bash
python company/company_utility.py --help
python company/company_utility.py <command> --help
```

### 全局参数

所有命令都支持以下可选参数：

- `-d DB_PATH` 或 `--db DB_PATH` - 指定数据库文件路径（默认: `./vikunja.db`）

### 初始化表

#### 初始化company表
```bash
# 使用默认数据库路径
python company/company_utility.py init_company

# 指定数据库路径
python company/company_utility.py -d /path/to/database.db init_company
python company/company_utility.py --db /path/to/database.db init_company
```

#### 初始化company_relation表
```bash
# 使用默认数据库路径
python company/company_utility.py init_relation

# 指定数据库路径
python company/company_utility.py -d /path/to/database.db init_relation
```

### 添加数据

#### 添加公司信息
```bash
# 使用默认数据库路径
python company/company_utility.py add_company <user_id> <username> <team_id>

# 指定数据库路径
python company/company_utility.py -d /path/to/database.db add_company <user_id> <username> <team_id>
```

示例：
```bash
python company/company_utility.py add_company 1 "company_user" 1
python company/company_utility.py -d ./vikunja.db add_company 1 "company_user" 1
```

#### 添加公司上下级关系
```bash
# 使用默认数据库路径
python company/company_utility.py add_relation <company_id> <superior_user_id> <subordinate_user_id> <project_id>

# 指定数据库路径
python company/company_utility.py -d /path/to/database.db add_relation <company_id> <superior_user_id> <subordinate_user_id> <project_id>
```

示例：
```bash
python company/company_utility.py add_relation 1 5 6 1
python company/company_utility.py --db ./vikunja.db add_relation 1 5 6 1
```

#### 删除公司上下级关系
```bash
# 使用默认数据库路径
python company/company_utility.py delete_relation <company_id> <superior_user_id> <subordinate_user_id>

# 指定数据库路径
python company/company_utility.py -d /path/to/database.db delete_relation <company_id> <superior_user_id> <subordinate_user_id>
```

示例：
```bash
python company/company_utility.py delete_relation 1 5 6
python company/company_utility.py --db ./vikunja.db delete_relation 1 5 6
```

#### 查询公司上下级关系的项目ID
```bash
# 使用默认数据库路径
python company/company_utility.py get_relation <company_id> <superior_user_id> <subordinate_user_id>

# 指定数据库路径
python company/company_utility.py -d /path/to/database.db get_relation <company_id> <superior_user_id> <subordinate_user_id>
```

示例：
```bash
python company/company_utility.py get_relation 1 5 6
python company/company_utility.py --db ./vikunja.db get_relation 1 5 6
```

### 查询信息

#### 查询公司信息
```bash
# 使用默认数据库路径
python company/company_utility.py get_company_info <team_id>

# 指定数据库路径
python company/company_utility.py -d /path/to/database.db get_company_info <team_id>
```

示例：
```bash
python company/company_utility.py get_company_info 1
python company/company_utility.py --db ./vikunja.db get_company_info 1
```

输出示例：
```
公司信息:
  Company用户ID: 1
  Company用户名: company_user
  团队ID: 1
  团队成员数: 16
  团队成员列表:
    1. 用户ID: 1, 用户名: leader, 姓名: 
    2. 用户ID: 5, 用户名: 小李, 姓名: 小李
    ...
```

## 函数API

所有函数都支持 `db_path` 参数：

```python
from company.company_utility import (
    init_company_table,
    init_company_relation_table,
    add_company,
    add_company_relation,
    delete_company_relation,
    get_relation_project,
    get_company_info
)

# 使用默认数据库路径
init_company_table()
init_company_relation_table()
add_company(1, "company_user", 1)
add_company_relation(1, 5, 6, 1)
delete_company_relation(1, 5, 6)
project_id = get_relation_project(1, 5, 6)
get_company_info(1)

# 指定数据库路径
db_path = '/path/to/database.db'
init_company_table(db_path)
init_company_relation_table(db_path)
add_company(1, "company_user", 1, db_path)
add_company_relation(1, 5, 6, 1, db_path)
delete_company_relation(1, 5, 6, db_path)
project_id = get_relation_project(1, 5, 6, db_path)
get_company_info(1, db_path)
```

## 注意事项

1. company表的team_id字段是唯一的，一个团队只能对应一个公司信息
2. company_relation表中的company_id、用户ID和项目ID必须在对应的company、users和projects表中存在
3. 工具只能读写company和company_relation表，对其他表只读操作
4. 所有操作都使用参数化查询，防止SQL注入
5. 默认数据库路径为 `./vikunja.db`，可通过 `-d` 或 `--db` 参数指定

## 依赖

- Python 3.x
- sqlite3（Python标准库，无需额外安装）
