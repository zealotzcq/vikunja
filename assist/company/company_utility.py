#!/usr/bin/env python3
# -*- coding: utf-8 -*-

import sys
import os
import sqlite3
import argparse
import bcrypt
import secrets
import string
import time

# 设置输出编码为UTF-8，解决Windows Git Bash中文乱码问题
if sys.platform == 'win32':
    os.environ['PYTHONIOENCODING'] = 'utf-8'
    if sys.version_info >= (3, 7):
        sys.stdout.reconfigure(encoding='utf-8')
        sys.stderr.reconfigure(encoding='utf-8')

DEFAULT_DB_PATH = '../../vikunja.db'
DEFAULT_API_URL = 'http://127.0.0.1:80'

# Import api_utils for user registration
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..', 'api'))
try:
    import api_utils
except ImportError:
    print('Warning: Could not import api_utils, falling back to direct database operations')
    api_utils = None

# 注意：company, company_staff, company_relation 这三张表应该由 Go 服务自动创建
# 表结构定义在 pkg/company/db.go 中
# 当服务启动时会通过 migration.Migrate() 自动创建这些表
# 如果表不存在，请重启 Vikunja 服务让它自动创建
#
# 初始化表的功能已移除，改为直接使用数据库


def get_connection(db_path):
    """获取数据库连接"""
    conn = sqlite3.connect(db_path)
    conn.row_factory = sqlite3.Row
    return conn


def create_user(username, email, password, db_path=DEFAULT_DB_PATH, api_url=DEFAULT_API_URL):
    """创建用户，优先使用API，失败则回退到数据库操作"""
    # 优先使用 API 创建用户
    if api_utils:
        try:
            result = api_utils.create_user(api_url, username, email, password)
            if isinstance(result, int):
                print(f'用户创建成功（通过API）: id={result}, username={username}, email={email}')
                return result
            else:
                print(f'API创建用户失败: {result}，尝试直接操作数据库')
        except Exception as e:
            print(f'API调用失败: {e}，尝试直接操作数据库')
    
    # 回退到直接数据库操作
    conn = get_connection(db_path)
    cursor = conn.cursor()
    
    try:
        # 检查用户名是否已存在
        cursor.execute('''
            SELECT id FROM users WHERE username = ?
        ''', (username,))
        existing = cursor.fetchone()
        
        if existing:
            print(f'用户名已存在: username={username}')
            return None
        
        # 检查email是否已存在
        cursor.execute('''
            SELECT id FROM users WHERE email = ?
        ''', (email,))
        existing_email = cursor.fetchone()
        
        if existing_email:
            print(f'邮箱已存在: email={email}')
            return None
        
        # 加密密码
        password_hash = bcrypt.hashpw(password.encode('utf-8'), bcrypt.gensalt()).decode('utf-8')
        
        # 插入用户，包含created和updated字段
        cursor.execute('''
            INSERT INTO users (username, email, password, name, status, created, updated)
            VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
        ''', (username, email, password_hash, username, 0))
        
        conn.commit()
        user_id = cursor.lastrowid
        print(f'用户创建成功（通过数据库）: id={user_id}, username={username}, email={email}')
        return user_id
    except Exception as e:
        print(f'创建用户失败: {e}')
        return None
    finally:
        conn.close()


def generate_invite_code():
    """生成邀请码"""
    return secrets.token_urlsafe(8)


def generate_numeric_lowercase_invite_code(length=6):
    """生成指定长度的数字+小写字母组合的邀请码"""
    characters = string.digits + string.ascii_lowercase
    return ''.join(secrets.choice(characters) for _ in range(length))

def add_company(description, invite_code=None, db_path=DEFAULT_DB_PATH):
    """添加公司信息到company表，如果同名已存在则跳过"""
    conn = get_connection(db_path)
    cursor = conn.cursor()
    
    try:
        # 如果没有提供邀请码，则自动生成
        if invite_code is None:
            invite_code = generate_invite_code()
        
        # 检查是否已存在同名公司
        cursor.execute('''
            SELECT id FROM company WHERE description = ?
        ''', (description,))
        existing = cursor.fetchone()
        
        if existing:
            print(f'公司已存在，跳过创建: description={description}')
            return False
        
        # 创建公司，XORM 的 created 字段是 int64 类型的 Unix 时间戳
        cursor.execute('''
            INSERT INTO company (description, invite_code, created)
            VALUES (?, ?, ?)
        ''', (description, invite_code, int(time.time())))
        company_id = cursor.lastrowid
        
        conn.commit()
        print(f'公司信息添加成功: id={company_id}, description={description}, invite_code={invite_code}')
        return company_id
    except Exception as e:
        print(f'添加公司信息失败: {e}')
        return False
    finally:
        conn.close()


def add_company_staff(company_id, user_id, role, db_path=DEFAULT_DB_PATH):
    """添加员工到公司，role: creator/admin/staff"""
    conn = get_connection(db_path)
    cursor = conn.cursor()
    
    if role not in ['creator', 'admin', 'staff']:
        print(f'错误: 角色必须是 creator, admin 或 staff')
        return False
    
    try:
        # 验证company_id是否存在
        cursor.execute('''
            SELECT id FROM company WHERE id = ?
        ''', (company_id,))
        company_exists = cursor.fetchone()
        
        if not company_exists:
            print(f'错误: 公司ID {company_id} 不存在')
            return False
        
        # 检查员工是否已经在公司
        cursor.execute('''
            SELECT id FROM company_staff WHERE company_id = ? AND user_id = ?
        ''', (company_id, user_id))
        existing = cursor.fetchone()
        
        if existing:
            print(f'用户已经是该公司员工: user_id={user_id}')
            return False
        
        cursor.execute('''
            INSERT INTO company_staff (company_id, user_id, role, created)
            VALUES (?, ?, ?, ?)
        ''', (company_id, user_id, role, int(time.time())))
        conn.commit()
        print(f'员工添加成功: company_id={company_id}, user_id={user_id}, role={role}')
        return True
    except Exception as e:
        print(f'添加员工失败: {e}')
        return False
    finally:
        conn.close()


def update_user_role(company_id, user_id, role, db_path=DEFAULT_DB_PATH):
    """更新用户在公司中的角色
    
    参数:
        company_id: 公司ID
        user_id: 用户ID
        role: 新角色 (creator/admin/staff)
        db_path: 数据库路径
    
    返回:
        True: 成功
        False: 失败
    """
    if role not in ['creator', 'admin', 'staff']:
        print(f'错误: 角色必须是 creator, admin 或 staff')
        return False
    
    conn = get_connection(db_path)
    cursor = conn.cursor()
    
    try:
        # 验证company_id是否存在
        cursor.execute('''
            SELECT id FROM company WHERE id = ?
        ''', (company_id,))
        company_exists = cursor.fetchone()
        
        if not company_exists:
            print(f'错误: 公司ID {company_id} 不存在')
            return False
        
        # 检查用户是否已经在公司
        cursor.execute('''
            SELECT id FROM company_staff WHERE company_id = ? AND user_id = ?
        ''', (company_id, user_id))
        existing = cursor.fetchone()
        
        if not existing:
            print(f'用户不在该公司中: user_id={user_id}, company_id={company_id}')
            return False
        
        # 更新角色
        cursor.execute('''
            UPDATE company_staff
            SET role = ?
            WHERE company_id = ? AND user_id = ?
        ''', (role, company_id, user_id))
        conn.commit()
        print(f'用户角色更新成功: company_id={company_id}, user_id={user_id}, role={role}')
        return True
    except Exception as e:
        print(f'更新角色失败: {e}')
        return False
    finally:
        conn.close()


def get_company_staff(company_id, db_path=DEFAULT_DB_PATH):
    """查询公司所有员工"""
    conn = get_connection(db_path)
    cursor = conn.cursor()
    
    try:
        company_id = int(company_id)
        
        # 查询公司所有员工
        cursor.execute('''
            SELECT cs.id, cs.company_id, cs.user_id, cs.role, cs.created, u.username
            FROM company_staff cs
            INNER JOIN users u ON cs.user_id = u.id
            WHERE cs.company_id = ?
            ORDER BY cs.role, cs.id
        ''', (company_id,))
        staff_list = cursor.fetchall()
        
        if not staff_list:
            print(f'公司ID为 {company_id} 没有员工')
            return []
        
        result = []
        print(f'公司员工列表 (公司ID: {company_id}):')
        for idx, staff in enumerate(staff_list, 1):
            staff_info = {
                'id': staff['id'],
                'company_id': staff['company_id'],
                'user_id': staff['user_id'],
                'username': staff['username'],
                'role': staff['role'],
                'created': staff['created']
            }
            result.append(staff_info)
            print(f'  {idx}. 用户ID: {staff["user_id"]}, 用户名: {staff["username"]}, 角色: {staff["role"]}')
        
        return result
    except ValueError:
        print('错误: company_id必须是整数')
        return []
    except Exception as e:
        print(f'查询公司员工失败: {e}')
        return []
    finally:
        conn.close()


def add_company_relation(company_id, superior_user_id, subordinate_user_id, project_id, db_path=DEFAULT_DB_PATH):
    """添加公司上下级关系到company_relation表"""
    conn = get_connection(db_path)
    cursor = conn.cursor()
    
    try:
        # 验证company_id是否存在
        cursor.execute('''
            SELECT id FROM company WHERE id = ?
        ''', (company_id,))
        company_exists = cursor.fetchone()
        
        if not company_exists:
            print(f'错误: 公司ID {company_id} 不存在')
            return False
        
        cursor.execute('''
            INSERT INTO company_relation (company_id, superior_user_id, subordinate_user_id, project_id, created, updated)
            VALUES (?, ?, ?, ?, ?, ?)
        ''', (company_id, superior_user_id, subordinate_user_id, project_id, int(time.time()), int(time.time())))
        conn.commit()
        print(f'公司关系添加成功: company_id={company_id}, superior_user_id={superior_user_id}, subordinate_user_id={subordinate_user_id}, project_id={project_id}')
        return True
    except Exception as e:
        print(f'添加公司关系失败: {e}')
        return False
    finally:
        conn.close()


def get_company_info(company_id, db_path=DEFAULT_DB_PATH):
    """查询公司信息，输入公司id，输出对应的company信息、创建者信息等"""
    conn = get_connection(db_path)
    cursor = conn.cursor()
    
    try:
        company_id = int(company_id)
        
        # 查询company表中的公司信息
        cursor.execute('''
            SELECT id, description, created, invite_code
            FROM company 
            WHERE id = ?
        ''', (company_id,))
        company_info = cursor.fetchone()
        
        if not company_info:
            print(f'未找到公司ID为 {company_id} 的公司信息')
            return None
        
        # 查询创建者信息
        cursor.execute('''
            SELECT u.id, u.username
            FROM users u
            INNER JOIN company_staff cs ON u.id = cs.user_id
            WHERE cs.company_id = ? AND cs.role = 'creator'
        ''', (company_id,))
        creator_info = cursor.fetchone()
        
        # 格式化输出
        result = {
            'id': company_info['id'],
            'description': company_info['description'],
            'created': company_info['created'],
            'invite_code': company_info['invite_code'],
            'creator_id': creator_info['id'] if creator_info else None,
            'creator_username': creator_info['username'] if creator_info else None
        }
        
        print(f'公司信息:')
        print(f'  公司ID: {result["id"]}')
        print(f'  公司名称: {result["description"]}')
        print(f'  创始者ID: {result["creator_id"]}')
        print(f'  创始者名称: {result["creator_username"]}')
        print(f'  创建时间: {result["created"]}')
        print(f'  邀请码: {result["invite_code"]}')
        
        return result
    except ValueError:
        print('错误: company_id必须是整数')
        return None
    except Exception as e:
        print(f'查询公司信息失败: {e}')
        return None
    finally:
        conn.close()


def update_invite_code(company_id, db_path=DEFAULT_DB_PATH):
    """更新公司邀请码，生成新的6位数字+小写字母组合的邀请码"""
    conn = get_connection(db_path)
    cursor = conn.cursor()
    
    try:
        company_id = int(company_id)
        
        # 检查公司是否存在
        cursor.execute('''
            SELECT id, description FROM company WHERE id = ?
        ''', (company_id,))
        company_info = cursor.fetchone()
        
        if not company_info:
            print(f'未找到公司ID为 {company_id} 的公司信息')
            return None
        
        # 生成新的6位邀请码
        new_code = generate_numeric_lowercase_invite_code(6)
        
        # 更新邀请码
        cursor.execute('''
            UPDATE company
            SET invite_code = ?
            WHERE id = ?
        ''', (new_code, company_id))
        
        conn.commit()
        
        print(f'邀请码更新成功:')
        print(f'  公司ID: {company_id}')
        print(f'  公司名称: {company_info["description"]}')
        print(f'  新邀请码: {new_code}')
        
        return new_code
    except ValueError:
        print('错误: company_id必须是整数')
        return None
    except Exception as e:
        print(f'更新邀请码失败: {e}')
        return None
    finally:
        conn.close()


def delete_company_relation(company_id, superior_user_id, subordinate_user_id, db_path=DEFAULT_DB_PATH):
    """删除公司上下级关系"""
    conn = get_connection(db_path)
    cursor = conn.cursor()
    
    try:
        cursor.execute('''
            DELETE FROM company_relation
            WHERE company_id = ? AND superior_user_id = ? AND subordinate_user_id = ?
        ''', (company_id, superior_user_id, subordinate_user_id))
        
        if cursor.rowcount == 0:
            print(f'未找到company_id为 {company_id}、上级ID为 {superior_user_id} 和下级ID为 {subordinate_user_id} 的关系')
            return False
        
        conn.commit()
        print(f'公司关系删除成功: company_id={company_id}, superior_user_id={superior_user_id}, subordinate_user_id={subordinate_user_id}')
        return True
    except Exception as e:
        print(f'删除公司关系失败: {e}')
        return False
    finally:
        conn.close()


def get_relation_project(company_id, superior_user_id, subordinate_user_id, db_path=DEFAULT_DB_PATH):
    """查询公司上下级关系对应的项目ID"""
    conn = get_connection(db_path)
    cursor = conn.cursor()
    
    try:
        cursor.execute('''
            SELECT project_id
            FROM company_relation
            WHERE company_id = ? AND superior_user_id = ? AND subordinate_user_id = ?
        ''', (company_id, superior_user_id, subordinate_user_id))
        result = cursor.fetchone()
        
        if not result:
            print(f'未找到company_id为 {company_id}、上级ID为 {superior_user_id} 和下级ID为 {subordinate_user_id} 的关系')
            return None
        
        project_id = result['project_id']
        print(f'关系查询成功: company_id={company_id}, superior_user_id={superior_user_id}, subordinate_user_id={subordinate_user_id}, project_id={project_id}')
        return project_id
    except Exception as e:
        print(f'查询关系失败: {e}')
        return None
    finally:
        conn.close()


def main():
    parser = argparse.ArgumentParser(description='公司信息管理工具')
    parser.add_argument('-d', '--db', type=str, default=DEFAULT_DB_PATH, 
                       help=f'数据库文件路径 (默认: {DEFAULT_DB_PATH})')
    parser.add_argument('-a', '--api', type=str, default=DEFAULT_API_URL,
                       help=f'API URL (默认: {DEFAULT_API_URL})')
    subparsers = parser.add_subparsers(dest='command', help='可用命令')
    
   
    # 创建用户命令
    create_user_parser = subparsers.add_parser('create_user', help='创建用户')
    create_user_parser.add_argument('username', type=str, help='用户名')
    create_user_parser.add_argument('email', type=str, help='邮箱')
    create_user_parser.add_argument('password', type=str, help='密码')
    
   
    # 添加公司信息命令
    add_company_parser = subparsers.add_parser('add_company', help='添加公司信息')
    add_company_parser.add_argument('description', type=str, help='公司名称')
    add_company_parser.add_argument('--invite_code', type=str, help='邀请码（可选，不指定则自动生成）')
    
    # 添加公司员工命令
    add_staff_parser = subparsers.add_parser('add_staff', help='添加员工到公司')
    add_staff_parser.add_argument('company_id', type=int, help='公司ID')
    add_staff_parser.add_argument('user_id', type=int, help='用户ID')
    add_staff_parser.add_argument('role', type=str, help='角色：creator/admin/staff')
    
    # 查询公司员工命令
    get_staff_parser = subparsers.add_parser('get_staff', help='查询公司所有员工')
    get_staff_parser.add_argument('company_id', type=str, help='公司ID')
    
    # 添加公司关系命令
    add_relation_parser = subparsers.add_parser('add_relation', help='添加公司上下级关系')
    add_relation_parser.add_argument('company_id', type=int, help='公司ID')
    add_relation_parser.add_argument('superior_user_id', type=int, help='上级用户ID')
    add_relation_parser.add_argument('subordinate_user_id', type=int, help='下级用户ID')
    add_relation_parser.add_argument('project_id', type=int, help='对应项目ID')
    
    # 删除公司关系命令
    delete_relation_parser = subparsers.add_parser('delete_relation', help='删除公司上下级关系')
    delete_relation_parser.add_argument('company_id', type=int, help='公司ID')
    delete_relation_parser.add_argument('superior_user_id', type=int, help='上级用户ID')
    delete_relation_parser.add_argument('subordinate_user_id', type=int, help='下级用户ID')
    
    # 查询关系命令
    get_relation_parser = subparsers.add_parser('get_relation', help='查询公司上下级关系的项目ID')
    get_relation_parser.add_argument('company_id', type=int, help='公司ID')
    get_relation_parser.add_argument('superior_user_id', type=int, help='上级用户ID')
    get_relation_parser.add_argument('subordinate_user_id', type=int, help='下级用户ID')
    
    # 查询公司信息命令
    get_info_parser = subparsers.add_parser('get_company_info', help='查询公司信息')
    get_info_parser.add_argument('company_id', type=str, help='公司ID')
    
    # 更新邀请码命令
    update_code_parser = subparsers.add_parser('update_invite_code', help='更新公司邀请码')
    update_code_parser.add_argument('company_id', type=str, help='公司ID')
    
    args = parser.parse_args()
    
    if not args.command:
        parser.print_help()
        sys.exit(1)
    
    if args.command == 'create_user':
        create_user(args.username, args.email, args.password, args.db, args.api)
    elif args.command == 'add_company':
        add_company(args.description, args.invite_code, args.db)
    elif args.command == 'add_staff':
        add_company_staff(args.company_id, args.user_id, args.role, args.db)
    elif args.command == 'get_staff':
        get_company_staff(args.company_id, args.db)
    elif args.command == 'add_relation':
        add_company_relation(args.company_id, args.superior_user_id, args.subordinate_user_id, args.project_id, args.db)
    elif args.command == 'delete_relation':
        delete_company_relation(args.company_id, args.superior_user_id, args.subordinate_user_id, args.db)
    elif args.command == 'get_relation':
        get_relation_project(args.company_id, args.superior_user_id, args.subordinate_user_id, args.db)
    elif args.command == 'get_company_info':
        get_company_info(args.company_id, args.db)
    elif args.command == 'update_invite_code':
        update_invite_code(args.company_id, args.db)


if __name__ == '__main__':
    main()
