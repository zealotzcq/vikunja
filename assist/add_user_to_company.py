#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
将用户加入公司的业务层接口
此文件封装了将用户加入公司的完整业务逻辑
"""

import sys
import os
import sqlite3
import requests

# 设置输出编码为UTF-8
if sys.platform == 'win32':
    os.environ['PYTHONIOENCODING'] = 'utf-8'
    if sys.version_info >= (3, 7):
        sys.stdout.reconfigure(encoding='utf-8')
        sys.stderr.reconfigure(encoding='utf-8')

# 导入底层工具模块
sys.path.insert(0, os.path.join(os.path.dirname(__file__), 'api'))
sys.path.insert(0, os.path.join(os.path.dirname(__file__), 'company'))

import api_utils
import company_utility

DEFAULT_API_URL = 'http://127.0.0.1:80'
DEFAULT_DB_PATH = os.path.join(os.path.dirname(__file__), '..', 'vikunja.db')
DEFAULT_PASSWORD = '12345678'


def update_user_settings_with_all_fields(base_url, token, settings_data):
    """更新用户设置（包含所有字段）"""
    headers = {
        'Authorization': f'Bearer {token}',
        'Content-Type': 'application/json'
    }
    try:
        url = f"{base_url.rstrip('/')}/api/v1/user/settings/general"
        response = requests.post(url, headers=headers, json=settings_data)
        if response.status_code == 200:
            return True
        else:
            print(f'更新设置失败: {response.status_code} - {response.text}')
            return False
    except requests.exceptions.RequestException as e:
        print(f'请求失败: {e}')
        return False


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


def add_user_to_company(username, email, company_id, role, api_url=DEFAULT_API_URL, db_path=DEFAULT_DB_PATH):
    """将用户加入公司（业务层接口）

    参数:
        username: 用户名
        email: 邮箱
        company_id: 公司ID
        role: 角色
        api_url: API URL (默认使用本地API)
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
    
    if user_id:
        # 用户已存在，返回错误
        result['message'] = f'用户已存在: {username}，无法重复创建'
        print(result['message'])
        return result
    
    # 用户不存在，需要先注册
    result['message'] = f'用户不存在，正在创建用户: {username}'
    print(result['message'])

    # 步骤2: 获取公司邀请码
    company_info = company_utility.get_company_info(company_id, db_path)
    if not company_info:
        result['message'] = f'获取公司信息失败，公司ID {company_id} 不存在'
        return result

    invite_code = company_info['invite_code']
    print(f'获取到邀请码: {invite_code}')

    # 步骤3: 用邀请码注册用户
    user_id = api_utils.create_user(api_url, username, email, DEFAULT_PASSWORD, invite_code)
    if isinstance(user_id, int):
        result['message'] = f'用户创建成功: {username}'
        print(result['message'])
    else:
        result['message'] = f'用户创建失败: {user_id}'
        return result

    result['user_id'] = user_id

    # 步骤4: 更新用户设置（语言和查找权限）
    try:
        print(f'正在更新用户设置...')
        # 以新用户身份登录
        token = api_utils.login(api_url, username, db_path)

        # 获取当前用户信息
        user_info = api_utils.get_user(api_url, token)

        # 构建设置数据
        if isinstance(user_info, dict):
            settings_data = user_info.get('settings', {})
        else:
            settings_data = {}

        # 更新设置
        settings_data['name'] = username
        settings_data['discoverable_by_name'] = True
        settings_data['discoverable_by_email'] = True
        settings_data['language'] = 'zh-CN'

        # 提交更新
        settings_result = update_user_settings_with_all_fields(api_url, token, settings_data)
        if settings_result:
            print(f'用户设置更新成功: 语言=zh-CN, 查找权限已启用')
        else:
            print(f'警告: 用户设置更新失败，但不影响加入公司')
    except Exception as e:
        print(f'警告: 更新用户设置时出错: {e}，但不影响加入公司')

    # 步骤5: 更新用户角色为指定角色（因为注册时默认是 staff）
    role_result = company_utility.update_user_role(company_id, user_id, role, db_path)
    if role_result:
        result['message'] += f', 已加入公司并设置为 {role}'
        result['success'] = True
    else:
        result['message'] += f', 设置角色失败'

    return result


def main():
    """命令行入口"""
    import argparse

    parser = argparse.ArgumentParser(description='将用户加入公司的业务层工具')
    parser.add_argument('username', type=str, help='用户名')
    parser.add_argument('email', type=str, help='邮箱')
    parser.add_argument('company_id', type=int, help='公司ID')
    parser.add_argument('role', type=str, help='角色 (creator/admin/staff)')
    parser.add_argument('-a', '--api', type=str, default=DEFAULT_API_URL,
                       help=f'API URL (默认: {DEFAULT_API_URL})')
    parser.add_argument('-d', '--db', type=str, default=DEFAULT_DB_PATH,
                       help=f'数据库文件路径 (默认: {DEFAULT_DB_PATH})')

    args = parser.parse_args()

    result = add_user_to_company(
        username=args.username,
        email=args.email,
        company_id=args.company_id,
        role=args.role,
        api_url=args.api,
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
