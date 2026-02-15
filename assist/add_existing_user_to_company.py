#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
将已存在的用户加入公司的业务层接口
此文件封装了将已存在的用户加入公司的完整业务逻辑
"""

import sys
import os
import sqlite3

# 设置输出编码为UTF-8
if sys.platform == 'win32':
    os.environ['PYTHONIOENCODING'] = 'utf-8'
    if sys.version_info >= (3, 7):
        sys.stdout.reconfigure(encoding='utf-8')
        sys.stderr.reconfigure(encoding='utf-8')

# 导入底层工具模块
sys.path.insert(0, os.path.join(os.path.dirname(__file__), 'company'))

import company_utility

DEFAULT_DB_PATH = os.path.join(os.path.dirname(__file__), '..', 'vikunja.db')


def get_connection(db_path):
    """获取数据库连接"""
    conn = sqlite3.connect(db_path)
    conn.row_factory = sqlite3.Row
    return conn


def check_user_exists(username, db_path=DEFAULT_DB_PATH):
    """检查用户是否存在，返回用户ID或None"""
    conn = get_connection(db_path)
    cursor = conn.cursor()
    
    try:
        cursor.execute('''
            SELECT id FROM users WHERE username = ?
        ''', (username,))
        result = cursor.fetchone()
        return result['id'] if result else None
    except Exception as e:
        print(f'检查用户是否存在失败: {e}')
        return None
    finally:
        conn.close()


def add_existing_user_to_company(username, company_id, role, db_path=DEFAULT_DB_PATH):
    """将已存在的用户加入公司（业务层接口）

    参数:
        username: 用户名
        company_id: 公司ID
        role: 角色 (creator/admin/staff)
        db_path: 数据库路径

    返回:
        {
            'success': True/False,
            'user_id': 用户ID (成功时),
            'company_id': 公司ID,
            'role': 角色,
            'message': 成功/失败信息
        }
    """
    result = {
        'success': False,
        'user_id': None,
        'company_id': company_id,
        'role': role,
        'message': ''
    }

    # 验证角色
    if role not in ['creator', 'admin', 'staff']:
        result['message'] = f'错误: 角色必须是 creator, admin 或 staff'
        return result

    # 步骤1: 检查用户是否存在
    user_id = check_user_exists(username, db_path)
    
    if not user_id:
        result['message'] = f'用户不存在: {username}'
        print(result['message'])
        return result
    
    print(f'用户存在: {username}')
    
    # 步骤2: 尝试添加员工到公司
    add_result = company_utility.add_company_staff(company_id, user_id, role, db_path)
    if add_result:
        result['user_id'] = user_id
        result['message'] = f'已加入公司并设置为 {role}'
        result['success'] = True
    else:
        # 添加失败，可能是用户已经在公司，尝试更新角色
        print(f'添加员工失败，尝试更新角色...')
        update_result = company_utility.update_user_role(company_id, user_id, role, db_path)
        if update_result:
            result['user_id'] = user_id
            result['message'] = f'已更新角色为 {role}'
            result['success'] = True
        else:
            result['message'] = f'加入公司失败'

    return result


def main():
    """命令行入口"""
    import argparse

    parser = argparse.ArgumentParser(description='将已存在的用户加入公司的业务层工具')
    parser.add_argument('username', type=str, help='用户名')
    parser.add_argument('company_id', type=int, help='公司ID')
    parser.add_argument('role', type=str, help='角色 (creator/admin/staff)')
    parser.add_argument('-d', '--db', type=str, default=DEFAULT_DB_PATH,
                       help=f'数据库文件路径 (默认: {DEFAULT_DB_PATH})')

    args = parser.parse_args()

    result = add_existing_user_to_company(
        username=args.username,
        company_id=args.company_id,
        role=args.role,
        db_path=args.db
    )

    if result['success']:
        print(f'✓ {result["message"]}')
        print(f'  用户ID: {result["user_id"]}')
        print(f'  公司ID: {result["company_id"]}')
        print(f'  角色: {result["role"]}')
    else:
        print(f'✗ {result["message"]}')
        sys.exit(1)


if __name__ == '__main__':
    main()
