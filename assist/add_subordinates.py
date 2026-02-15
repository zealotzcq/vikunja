#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
为用户添加下级的业务层接口
此文件封装了为用户添加下级的完整业务逻辑
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


def get_user_id_by_username(username, db_path):
    """通过用户名获取用户ID

    参数:
        username: 用户名
        db_path: 数据库路径

    返回:
        用户ID (int) 或 None (用户不存在)
    """
    import sqlite3
    conn = sqlite3.connect(db_path)
    conn.row_factory = sqlite3.Row
    cursor = conn.cursor()

    try:
        cursor.execute('''
            SELECT id FROM users WHERE username = ?
        ''', (username,))
        result = cursor.fetchone()
        return result['id'] if result else None
    finally:
        conn.close()


def check_user_in_company(user_id, company_id, db_path):
    """检查用户是否在公司中

    参数:
        user_id: 用户ID
        company_id: 公司ID
        db_path: 数据库路径

    返回:
        True: 用户在公司中
        False: 用户不在公司中
    """
    import sqlite3
    conn = sqlite3.connect(db_path)
    conn.row_factory = sqlite3.Row
    cursor = conn.cursor()

    try:
        cursor.execute('''
            SELECT id FROM company_staff WHERE company_id = ? AND user_id = ?
        ''', (company_id, user_id))
        result = cursor.fetchone()
        return result is not None
    finally:
        conn.close()


def check_relation_exists(company_id, superior_user_id, subordinate_user_id, db_path):
    """检查上下级关系是否已存在

    参数:
        company_id: 公司ID
        superior_user_id: 上级用户ID
        subordinate_user_id: 下级用户ID
        db_path: 数据库路径

    返回:
        (exists: bool, project_id: int) - 关系已存在和对应的项目ID
    """
    import sqlite3
    conn = sqlite3.connect(db_path)
    conn.row_factory = sqlite3.Row
    cursor = conn.cursor()

    try:
        cursor.execute('''
            SELECT project_id FROM company_relation 
            WHERE company_id = ? AND superior_user_id = ? AND subordinate_user_id = ?
        ''', (company_id, superior_user_id, subordinate_user_id))
        result = cursor.fetchone()
        if result:
            return True, result['project_id']
        return False, 0
    finally:
        conn.close()


def add_subordinates(company_id, superior_username, subordinate_usernames, project_id=None, api_url=DEFAULT_API_URL, db_path=DEFAULT_DB_PATH):
    """为用户添加下级（业务层接口）

    参数:
        company_id: 公司ID
        superior_username: 上级用户名
        subordinate_usernames: 下级用户名列表
        project_id: 对应项目ID (可选，如果未提供则自动创建新项目)
        api_url: API URL (默认使用本地API)
        db_path: 数据库路径

    返回:
        {
            'success': True/False,
            'company_id': 公司ID,
            'superior': {'id': 用户ID, 'username': 用户名},
            'results': [
                {
                    'subordinate': {'id': 用户ID, 'username': 用户名},
                    'success': True/False,
                    'reason': 原因描述,
                    'project_id': 项目ID
                },
                ...
            ]
        }
    """
    result = {
        'success': False,
        'company_id': company_id,
        'superior': None,
        'results': []
    }

    # 步骤1: 获取上级用户ID
    superior_user_id = get_user_id_by_username(superior_username, db_path)
    if not superior_user_id:
        result['superior'] = {'id': None, 'username': superior_username}
        for sub_username in subordinate_usernames:
            result['results'].append({
                'subordinate': {'id': None, 'username': sub_username},
                'success': False,
                'reason': f'上级用户不存在',
                'project_id': None
            })
        return result
    result['superior'] = {'id': superior_user_id, 'username': superior_username}

    # 步骤2: 检查上级是否在公司中
    if not check_user_in_company(superior_user_id, company_id, db_path):
        for sub_username in subordinate_usernames:
            result['results'].append({
                'subordinate': {'id': None, 'username': sub_username},
                'success': False,
                'reason': f'上级用户不在公司中',
                'project_id': None
            })
        return result

    # 步骤3: 用上级用户登录API
    try:
        superior_token = api_utils.login(api_url, superior_username, db_path)
    except Exception as e:
        for sub_username in subordinate_usernames:
            result['results'].append({
                'subordinate': {'id': None, 'username': sub_username},
                'success': False,
                'reason': f'上级用户登录失败: {str(e)}',
                'project_id': None
            })
        return result

    # 步骤4: 处理每个下级用户
    all_success = True
    for sub_username in subordinate_usernames:
        sub_result = {
            'subordinate': {'id': None, 'username': sub_username},
            'success': False,
            'reason': '',
            'project_id': None
        }

        # 获取下级用户ID
        sub_user_id = get_user_id_by_username(sub_username, db_path)
        if not sub_user_id:
            sub_result['reason'] = '用户不存在'
            all_success = False
            result['results'].append(sub_result)
            continue
        sub_result['subordinate']['id'] = sub_user_id

        # 检查下级是否在公司中
        if not check_user_in_company(sub_user_id, company_id, db_path):
            sub_result['reason'] = '用户不在公司中'
            all_success = False
            result['results'].append(sub_result)
            continue

        # 检查关系是否已存在
        relation_exists, existing_project_id = check_relation_exists(company_id, superior_user_id, sub_user_id, db_path)
        if relation_exists:
            sub_result['success'] = True
            sub_result['reason'] = '上下级关系已存在'
            sub_result['project_id'] = existing_project_id
            result['results'].append(sub_result)
            continue

        # 关系不存在，创建新项目
        try:
            project_title = f"to {sub_username}"
            new_project_id = api_utils.create_project(api_url, superior_token, project_title)

            if isinstance(new_project_id, int):
                # 共享项目给下级用户（权限1=读写）
                share_result = api_utils.share_project_with_user(api_url, superior_token, new_project_id, sub_username, 1)

                if share_result is True:
                    # 添加上下级关系到数据库
                    add_result = company_utility.add_company_relation(company_id, superior_user_id, sub_user_id, new_project_id, db_path)
                    if add_result:
                        sub_result['success'] = True
                        sub_result['reason'] = '添加成功'
                        sub_result['project_id'] = new_project_id
                    else:
                        sub_result['reason'] = '添加关系失败（数据库错误）'
                        all_success = False
                else:
                    sub_result['reason'] = f'共享项目失败: {share_result}'
                    all_success = False
            else:
                sub_result['reason'] = f'创建项目失败: {new_project_id}'
                all_success = False
        except Exception as e:
            sub_result['reason'] = f'创建/共享项目时出错: {str(e)}'
            all_success = False
        result['results'].append(sub_result)

    result['success'] = all_success
    return result


def main():
    """命令行入口"""
    import argparse

    parser = argparse.ArgumentParser(description='为用户添加下级的业务层工具')
    parser.add_argument('company_id', type=int, help='公司ID')
    parser.add_argument('superior_username', type=str, help='上级用户名')
    parser.add_argument('subordinate_usernames', type=str, help='下级用户名列表，用逗号分隔，例如: user1,user2,user3')
    parser.add_argument('-p', '--project_id', type=int, default=None, help='对应项目ID (可选，未提供则自动创建新项目)')
    parser.add_argument('-a', '--api', type=str, default=DEFAULT_API_URL,
                       help=f'API URL (默认: {DEFAULT_API_URL})')
    parser.add_argument('-d', '--db', type=str, default=DEFAULT_DB_PATH,
                       help=f'数据库文件路径 (默认: {DEFAULT_DB_PATH})')

    args = parser.parse_args()

    # 解析下级用户名列表
    subordinate_usernames = [name.strip() for name in args.subordinate_usernames.split(',')]

    result = add_subordinates(
        company_id=args.company_id,
        superior_username=args.superior_username,
        subordinate_usernames=subordinate_usernames,
        project_id=args.project_id,
        api_url=args.api,
        db_path=args.db
    )

    # 输出结果
    print(f'公司ID: {result["company_id"]}')
    print(f'上级用户: {result["superior"]["username"]} (ID: {result["superior"]["id"]})')
    print()
    print('下级添加结果:')
    print('-' * 60)

    for idx, res in enumerate(result['results'], 1):
        status = '✓' if res['success'] else '✗'
        print(f'{idx}. {status} {res["subordinate"]["username"]} (ID: {res["subordinate"]["id"]})')
        print(f'   原因: {res["reason"]}')
        if res['project_id']:
            print(f'   项目ID: {res["project_id"]}')
        print()

    print('-' * 60)
    success_count = sum(1 for r in result['results'] if r['success'])
    fail_count = len(result['results']) - success_count
    print(f'总计: {len(result["results"])} 个, 成功: {success_count}, 失败: {fail_count}')

    if not result['success']:
        sys.exit(1)


if __name__ == '__main__':
    main()
