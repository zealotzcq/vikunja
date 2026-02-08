#!/usr/bin/env python3
# -*- coding: utf-8 -*-

import sys
import sqlite3
import argparse

# 设置输出编码为UTF-8，解决Windows Git Bash中文乱码问题
if sys.platform == 'win32':
    sys.stdout.reconfigure(encoding='utf-8')
    sys.stderr.reconfigure(encoding='utf-8')

DEFAULT_DB_PATH = './vikunja.db'


def get_connection(db_path):
    """获取数据库连接"""
    conn = sqlite3.connect(db_path)
    conn.row_factory = sqlite3.Row
    return conn


def init_company_table(db_path=DEFAULT_DB_PATH):
    """初始化company表格，如果表格已经存在不重复创建"""
    conn = get_connection(db_path)
    cursor = conn.cursor()
    
    try:
        # 定义期望的表结构（列名列表）
        expected_columns = ['id', 'user_id', 'username', 'team_id', 'created', 'updated']
        
        # 检查表是否存在
        cursor.execute('''
            SELECT name FROM sqlite_master 
            WHERE type='table' AND name='company'
        ''')
        table_exists = cursor.fetchone()
        
        if table_exists:
            # 获取现有表的列信息
            cursor.execute('''
                PRAGMA table_info(company)
            ''')
            existing_columns = [col[1] for col in cursor.fetchall()]
            
            # 比较列数和列名
            if len(existing_columns) == len(expected_columns) and set(existing_columns) == set(expected_columns):
                print('company表格已存在且结构一致')
            else:
                # 结构不一致，删除旧表
                print('检测到company表结构不一致，正在重建...')
                cursor.execute('DROP TABLE company')
                
                # 创建新表
                cursor.execute('''
                    CREATE TABLE company (
                        id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
                        user_id INTEGER NOT NULL,
                        username TEXT NOT NULL,
                        team_id INTEGER NOT NULL,
                        created DATETIME DEFAULT CURRENT_TIMESTAMP,
                        updated DATETIME DEFAULT CURRENT_TIMESTAMP,
                        UNIQUE(team_id)
                    )
                ''')
                conn.commit()
                print('company表格重建成功')
        else:
            # 创建新表
            cursor.execute('''
                CREATE TABLE company (
                    id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
                    user_id INTEGER NOT NULL,
                    username TEXT NOT NULL,
                    team_id INTEGER NOT NULL,
                    created DATETIME DEFAULT CURRENT_TIMESTAMP,
                    updated DATETIME DEFAULT CURRENT_TIMESTAMP,
                    UNIQUE(team_id)
                )
            ''')
            conn.commit()
            print('company表格初始化成功')
    except Exception as e:
        print(f'初始化company表格失败: {e}')
        return False
    finally:
        conn.close()
    return True


def init_company_relation_table(db_path=DEFAULT_DB_PATH):
    """初始化company_relation表格，如果表格已经存在不重复创建"""
    conn = get_connection(db_path)
    cursor = conn.cursor()
    
    try:
        # 定义期望的表结构（列名列表）
        expected_columns = ['id', 'company_id', 'superior_user_id', 'subordinate_user_id', 'project_id', 'created', 'updated']
        
        # 检查表是否存在
        cursor.execute('''
            SELECT name FROM sqlite_master 
            WHERE type='table' AND name='company_relation'
        ''')
        table_exists = cursor.fetchone()
        
        if table_exists:
            # 获取现有表的列信息
            cursor.execute('''
                PRAGMA table_info(company_relation)
            ''')
            existing_columns = [col[1] for col in cursor.fetchall()]
            
            # 比较列数和列名
            if len(existing_columns) == len(expected_columns) and set(existing_columns) == set(expected_columns):
                print('company_relation表格已存在且结构一致')
            else:
                # 结构不一致，删除旧表
                print('检测到company_relation表结构不一致，正在重建...')
                cursor.execute('DROP TABLE company_relation')
                
                # 创建新表
                cursor.execute('''
                    CREATE TABLE company_relation (
                        id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
                        company_id INTEGER NOT NULL,
                        superior_user_id INTEGER NOT NULL,
                        subordinate_user_id INTEGER NOT NULL,
                        project_id INTEGER NOT NULL,
                        created DATETIME DEFAULT CURRENT_TIMESTAMP,
                        updated DATETIME DEFAULT CURRENT_TIMESTAMP,
                        FOREIGN KEY (company_id) REFERENCES company(team_id),
                        FOREIGN KEY (superior_user_id) REFERENCES users(id),
                        FOREIGN KEY (subordinate_user_id) REFERENCES users(id),
                        FOREIGN KEY (project_id) REFERENCES projects(id)
                    )
                ''')
                conn.commit()
                print('company_relation表格重建成功')
        else:
            # 创建新表
            cursor.execute('''
                CREATE TABLE company_relation (
                    id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
                    company_id INTEGER NOT NULL,
                    superior_user_id INTEGER NOT NULL,
                    subordinate_user_id INTEGER NOT NULL,
                    project_id INTEGER NOT NULL,
                    created DATETIME DEFAULT CURRENT_TIMESTAMP,
                    updated DATETIME DEFAULT CURRENT_TIMESTAMP,
                    FOREIGN KEY (company_id) REFERENCES company(team_id),
                    FOREIGN KEY (superior_user_id) REFERENCES users(id),
                    FOREIGN KEY (subordinate_user_id) REFERENCES users(id),
                    FOREIGN KEY (project_id) REFERENCES projects(id)
                )
            ''')
            conn.commit()
            print('company_relation表格初始化成功')
    except Exception as e:
        print(f'初始化company_relation表格失败: {e}')
        return False
    finally:
        conn.close()
    return True


def add_company(user_id, username, team_id, db_path=DEFAULT_DB_PATH):
    """添加公司信息到company表"""
    conn = get_connection(db_path)
    cursor = conn.cursor()
    
    try:
        cursor.execute('''
            INSERT OR REPLACE INTO company (user_id, username, team_id)
            VALUES (?, ?, ?)
        ''', (user_id, username, team_id))
        conn.commit()
        print(f'公司信息添加成功: user_id={user_id}, username={username}, team_id={team_id}')
        return True
    except Exception as e:
        print(f'添加公司信息失败: {e}')
        return False
    finally:
        conn.close()


def add_company_relation(company_id, superior_user_id, subordinate_user_id, project_id, db_path=DEFAULT_DB_PATH):
    """添加公司上下级关系到company_relation表"""
    conn = get_connection(db_path)
    cursor = conn.cursor()
    
    try:
        cursor.execute('''
            INSERT INTO company_relation (company_id, superior_user_id, subordinate_user_id, project_id)
            VALUES (?, ?, ?, ?)
        ''', (company_id, superior_user_id, subordinate_user_id, project_id))
        conn.commit()
        print(f'公司关系添加成功: company_id={company_id}, superior_user_id={superior_user_id}, subordinate_user_id={subordinate_user_id}, project_id={project_id}')
        return True
    except Exception as e:
        print(f'添加公司关系失败: {e}')
        return False
    finally:
        conn.close()


def get_company_info(team_id, db_path=DEFAULT_DB_PATH):
    """查询公司信息，输入团队id，输出对应的company用户id，所有团队成员信息数组"""
    conn = get_connection(db_path)
    cursor = conn.cursor()
    
    try:
        team_id = int(team_id)
        
        # 查询company表中的公司信息
        cursor.execute('''
            SELECT user_id, username, team_id 
            FROM company 
            WHERE team_id = ?
        ''', (team_id,))
        company_info = cursor.fetchone()
        
        if not company_info:
            print(f'未找到团队ID为 {team_id} 的公司信息')
            return None
        
        # 查询该团队的所有成员
        cursor.execute('''
            SELECT u.id, u.username, u.name
            FROM users u
            INNER JOIN team_members tm ON u.id = tm.user_id
            WHERE tm.team_id = ?
        ''', (team_id,))
        team_members = cursor.fetchall()
        
        # 格式化输出
        result = {
            'company_user_id': company_info['user_id'],
            'company_username': company_info['username'],
            'team_id': company_info['team_id'],
            'team_members': []
        }
        
        for member in team_members:
            result['team_members'].append({
                'user_id': member['id'],
                'username': member['username'],
                'name': member['name'] if member['name'] else ''
            })
        
        print(f'公司信息:')
        print(f'  Company用户ID: {result["company_user_id"]}')
        print(f'  Company用户名: {result["company_username"]}')
        print(f'  团队ID: {result["team_id"]}')
        print(f'  团队成员数: {len(result["team_members"])}')
        print(f'  团队成员列表:')
        for idx, member in enumerate(result['team_members'], 1):
            print(f'    {idx}. 用户ID: {member["user_id"]}, 用户名: {member["username"]}, 姓名: {member["name"]}')
        
        return result
    except ValueError:
        print('错误: team_id必须是整数')
        return None
    except Exception as e:
        print(f'查询公司信息失败: {e}')
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
    subparsers = parser.add_subparsers(dest='command', help='可用命令')
    
    # 初始化company表命令
    init_company_parser = subparsers.add_parser('init_company', help='初始化company表')
    
    # 初始化company_relation表命令
    init_relation_parser = subparsers.add_parser('init_relation', help='初始化company_relation表')
    
    # 添加公司信息命令
    add_company_parser = subparsers.add_parser('add_company', help='添加公司信息')
    add_company_parser.add_argument('user_id', type=int, help='公司用户ID')
    add_company_parser.add_argument('username', type=str, help='登录用户名')
    add_company_parser.add_argument('team_id', type=int, help='对应团队ID')
    
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
    get_info_parser.add_argument('team_id', type=str, help='团队ID')
    
    args = parser.parse_args()
    
    if not args.command:
        parser.print_help()
        sys.exit(1)
    
    if args.command == 'init_company':
        init_company_table(args.db)
    elif args.command == 'init_relation':
        init_company_relation_table(args.db)
    elif args.command == 'add_company':
        add_company(args.user_id, args.username, args.team_id, args.db)
    elif args.command == 'add_relation':
        add_company_relation(args.company_id, args.superior_user_id, args.subordinate_user_id, args.project_id, args.db)
    elif args.command == 'delete_relation':
        delete_company_relation(args.company_id, args.superior_user_id, args.subordinate_user_id, args.db)
    elif args.command == 'get_relation':
        get_relation_project(args.company_id, args.superior_user_id, args.subordinate_user_id, args.db)
    elif args.command == 'get_company_info':
        get_company_info(args.team_id, args.db)


if __name__ == '__main__':
    main()
