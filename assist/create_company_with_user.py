#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
创建用户并创建公司的业务层接口
此文件封装了创建用户和创建公司的完整业务逻辑
"""

import sys
import os

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


def create_company_with_user(username, email, password, company_name, api_url=DEFAULT_API_URL, db_path=DEFAULT_DB_PATH):
    """创建用户并创建公司（业务层接口）

    参数:
        username: 用户名
        email: 邮箱
        password: 密码
        company_name: 公司名称
        api_url: API URL (默认使用本地API)
        db_path: 数据库路径

    返回:
        {
            'success': True/False,
            'user_id': 用户ID (成功时),
            'company_id': 公司ID (成功时),
            'invite_code': 公司邀请码 (成功时),
            'message': 成功/失败信息
        }
    """
    result = {
        'success': False,
        'user_id': None,
        'company_id': None,
        'invite_code': None,
        'message': ''
    }

    # 步骤1: 创建用户（通过 API）
    user_id = api_utils.create_user(api_url, username, email, password)
    if isinstance(user_id, int):
        result['user_id'] = user_id
        result['message'] = f'用户创建成功: {username}'
    else:
        result['message'] = f'用户创建失败: {user_id}'
        return result

    # 步骤2: 生成邀请码
    invite_code = company_utility.generate_numeric_lowercase_invite_code(6)

    # 步骤3: 创建公司
    company_id = company_utility.add_company(company_name, invite_code, db_path)
    if company_id:
        result['company_id'] = company_id
        result['invite_code'] = invite_code
        result['message'] += f', 公司创建成功: {company_name}'
    else:
        result['message'] += f', 公司创建失败'
        return result

    # 步骤4: 添加用户为创建者
    staff_result = company_utility.add_company_staff(company_id, user_id, 'creator', db_path)
    if staff_result:
        result['message'] += f', 用户已添加为创建者'
        result['success'] = True
    else:
        result['message'] += f', 添加创建者失败'

    return result


def main():
    """命令行入口"""
    import argparse

    parser = argparse.ArgumentParser(description='创建用户并创建公司的业务层工具')
    parser.add_argument('username', type=str, help='用户名')
    parser.add_argument('email', type=str, help='邮箱')
    parser.add_argument('password', type=str, help='密码')
    parser.add_argument('company_name', type=str, help='公司名称')
    parser.add_argument('-a', '--api', type=str, default=DEFAULT_API_URL,
                       help=f'API URL (默认: {DEFAULT_API_URL})')
    parser.add_argument('-d', '--db', type=str, default=DEFAULT_DB_PATH,
                       help=f'数据库文件路径 (默认: {DEFAULT_DB_PATH})')

    args = parser.parse_args()

    result = create_company_with_user(
        username=args.username,
        email=args.email,
        password=args.password,
        company_name=args.company_name,
        api_url=args.api,
        db_path=args.db
    )

    if result['success']:
        print(f'✓ {result["message"]}')
        print(f'  用户ID: {result["user_id"]}')
        print(f'  公司ID: {result["company_id"]}')
        print(f'  邀请码: {result["invite_code"]}')
    else:
        print(f'✗ {result["message"]}')
        sys.exit(1)


if __name__ == '__main__':
    main()
