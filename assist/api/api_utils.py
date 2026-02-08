#!/usr/bin/env python3
"""
Vikunja API utility functions
"""

import sys
import os

if sys.platform == 'win32':
    os.environ['PYTHONIOENCODING'] = 'utf-8'
    if sys.version_info >= (3, 7):
        sys.stdout.reconfigure(encoding='utf-8')
        sys.stderr.reconfigure(encoding='utf-8')

import requests
from typing import Dict, List, Optional, Any, Union


def login(base_url: str, username: str, password: str) -> str:
    """Login and get JWT token"""
    url = f"{base_url.rstrip('/')}/api/v1/login"
    response = requests.post(
        url,
        json={'username': username, 'password': password}
    )
    response.raise_for_status()
    token = response.json().get('token')
    if not token:
        raise Exception("Failed to obtain authentication token")
    return token


def create_user(base_url: str, username: str, email: str, password: str) -> Union[int, str]:
    """Create new user, returns user ID on success, failure reason on failure"""
    url = f"{base_url.rstrip('/')}/api/v1/register"
    data = {
        'username': username,
        'email': email,
        'password': password
    }
    try:
        response = requests.post(url, json=data)
        if response.status_code == 200:
            user_id = response.json().get('id')
            if not user_id:
                return "Failed to get user ID"
            return user_id
        elif response.status_code == 400:
            return response.json().get('message', 'Bad request')
        elif response.status_code == 409:
            return "Username or email already exists"
        else:
            return f"Failed with status code: {response.status_code}"
    except requests.exceptions.RequestException as e:
        raise Exception(f"Request failed: {e}")


def get_user(base_url: str, token: str) -> Union[dict, str]:
    """Get current user info, returns user dict on success, failure reason on failure"""
    headers = {
        'Authorization': f'Bearer {token}'
    }
    try:
        url = f"{base_url.rstrip('/')}/api/v1/user"
        response = requests.get(url, headers=headers)
        if response.status_code == 200:
            return response.json()
        elif response.status_code == 401:
            return "Unauthorized"
        elif response.status_code == 404:
            return "User not found"
        else:
            return f"Failed with status code: {response.status_code}"
    except requests.exceptions.RequestException as e:
        raise Exception(f"Request failed: {e}")


def update_user_settings(base_url: str, token: str, name: str) -> Union[int, str]:
    """Update user settings, returns True on success, failure reason on failure"""
    user_info = get_user(base_url, token)
    if isinstance(user_info, str):
        return user_info

    user_settings = user_info.get('settings', {})

    user_settings['name'] = name
    user_settings['discoverable_by_name'] = True
    user_settings['discoverable_by_email'] = True
    user_settings['language'] = 'zh-CN'

    headers = {
        'Authorization': f'Bearer {token}',
        'Content-Type': 'application/json'
    }
    try:
        url = f"{base_url.rstrip('/')}/api/v1/user/settings/general"
        response = requests.post(url, headers=headers, json=user_settings)
        if response.status_code == 200:
            return True
        elif response.status_code == 400:
            return response.json().get('message', 'Bad request')
        elif response.status_code == 401:
            return "Unauthorized"
        elif response.status_code == 404:
            return "User not found"
        else:
            return f"Failed with status code: {response.status_code}"
    except requests.exceptions.RequestException as e:
        raise Exception(f"Request failed: {e}")


def create_project(base_url: str, token: str, title: str) -> Union[int, str]:
    """Create new project, returns project ID on success, failure reason on failure"""
    headers = {
        'Authorization': f'Bearer {token}',
        'Content-Type': 'application/json'
    }
    data = {
        'title': title
    }
    try:
        url = f"{base_url.rstrip('/')}/api/v1/projects"
        response = requests.put(url, headers=headers, json=data)
        if response.status_code in (200, 201):
            project_id = response.json().get('id')
            if not project_id:
                return "Failed to get project ID"
            return project_id
        elif response.status_code == 400:
            return response.json().get('message', 'Bad request')
        elif response.status_code == 401:
            return "Unauthorized"
        else:
            return f"Failed with status code: {response.status_code}"
    except requests.exceptions.RequestException as e:
        raise Exception(f"Request failed: {e}")


def create_team(base_url: str, token: str, name: str) -> Union[int, str]:
    """Create new team, returns team ID on success, failure reason on failure"""
    headers = {
        'Authorization': f'Bearer {token}',
        'Content-Type': 'application/json'
    }
    data = {
        'name': name
    }
    try:
        url = f"{base_url.rstrip('/')}/api/v1/teams"
        response = requests.put(url, headers=headers, json=data)
        if response.status_code == 201:
            team_id = response.json().get('id')
            if not team_id:
                return "Failed to get team ID"
            return team_id
        elif response.status_code == 400:
            return response.json().get('message', 'Bad request')
        elif response.status_code == 401:
            return "Unauthorized"
        else:
            return f"Failed with status code: {response.status_code}"
    except requests.exceptions.RequestException as e:
        raise Exception(f"Request failed: {e}")


def share_project_with_user(base_url: str, token: str, project_id: int, username: str, permission: int = 1) -> Union[int, str]:
    """Share project with user by username. Permission: 0=Read, 1=Read & Write, 2=Admin"""
    headers = {
        'Authorization': f'Bearer {token}',
        'Content-Type': 'application/json'
    }
    data = {
        'username': username,
        'permission': permission
    }
    try:
        url = f"{base_url.rstrip('/')}/api/v1/projects/{project_id}/users"
        response = requests.put(url, headers=headers, json=data)
        if response.status_code in (200, 201):
            return True
        elif response.status_code == 400:
            return response.json().get('message', 'Bad request')
        elif response.status_code == 401:
            return "Unauthorized"
        elif response.status_code == 404:
            return "User or project not found"
        else:
            return f"Failed with status code: {response.status_code}"
    except requests.exceptions.RequestException as e:
        raise Exception(f"Request failed: {e}")


def add_team_member(base_url: str, token: str, team_id: int, username: str) -> Union[int, str]:
    """Add user to team"""
    headers = {
        'Authorization': f'Bearer {token}',
        'Content-Type': 'application/json'
    }
    data = {
        'username': username
    }
    try:
        url = f"{base_url.rstrip('/')}/api/v1/teams/{team_id}/members"
        response = requests.put(url, headers=headers, json=data)
        if response.status_code in (200, 201):
            return True
        elif response.status_code == 400:
            return response.json().get('message', 'Bad request')
        elif response.status_code == 401:
            return "Unauthorized"
        elif response.status_code == 404:
            return "User or team not found"
        else:
            return f"Failed with status code: {response.status_code}"
    except requests.exceptions.RequestException as e:
        raise Exception(f"Request failed: {e}")


if __name__ == '__main__':
    import sys

    BASE_URL = "http://127.0.0.1:3456"
    ADMIN_USERNAME = "王大牛"
    ADMIN_PASSWORD = "50095152"

    if len(sys.argv) > 1:
        test_name = sys.argv[1]

        if test_name == 'login':
            print("=== Test: Login ===")
            try:
                token = login(BASE_URL, ADMIN_USERNAME, ADMIN_PASSWORD)
                print(f"[OK] Login successful")
                print(f"Token: {token}")
            except Exception as e:
                print(f"[FAIL] Error: {e}")

        elif test_name == 'get_user':
            print("=== Test: Get User ===")
            try:
                token = login(BASE_URL, ADMIN_USERNAME, ADMIN_PASSWORD)
                print(f"[OK] Login successful")

                user_info = get_user(BASE_URL, token)
                if isinstance(user_info, dict):
                    print(f"[OK] Get user info successful")
                    print(f"Username: {user_info.get('username')}")
                    print(f"Name: {user_info.get('name')}")
                    print(f"Email: {user_info.get('email')}")
                    print(f"ID: {user_info.get('id')}")
                else:
                    print(f"[FAIL] Get user info failed: {user_info}")
            except Exception as e:
                print(f"[FAIL] Error: {e}")

        elif test_name == 'create_user':
            import time

            print("=== Test: Create User ===")

            print("\nTest 1: Create new user")
            timestamp = str(int(time.time()))
            test_username = f"testuser{timestamp}"
            test_email = f"testuser{timestamp}@example.com"
            test_password = "12345678"
            try:
                result = create_user(
                    base_url=BASE_URL,
                    username=test_username,
                    email=test_email,
                    password=test_password
                )
                if isinstance(result, int):
                    print(f"[OK] User created successfully, User ID: {result}")
                else:
                    print(f"[FAIL] Failed: {result}")
            except Exception as e:
                print(f"[ERROR] Exception: {e}")

            print("\nTest 2: Create duplicate user")
            try:
                result = create_user(
                    base_url=BASE_URL,
                    username=test_username,
                    email=test_email,
                    password=test_password
                )
                if isinstance(result, int):
                    print(f"[OK] User created successfully, User ID: {result}")
                else:
                    print(f"[OK] Expected failure: {result}")
            except Exception as e:
                print(f"[ERROR] Exception: {e}")

            print("\nTest 3: Create user with duplicate email")
            try:
                result = create_user(
                    base_url=BASE_URL,
                    username="testuser891",
                    email=test_email,
                    password=test_password
                )
                if isinstance(result, int):
                    print(f"[OK] User created successfully, User ID: {result}")
                else:
                    print(f"[OK] Expected failure: {result}")
            except Exception as e:
                print(f"[ERROR] Exception: {e}")

        elif test_name == 'update_user_settings':
            print("=== Test: Update User Settings ===")
            import time

            timestamp = str(int(time.time()))
            test_username = f"testuser{timestamp}"
            test_email = f"testuser{timestamp}@example.com"
            test_password = "12345678"
            test_name_alias = "测试员工"

            print("\nStep 1: Create user")
            result = create_user(
                base_url=BASE_URL,
                username=test_username,
                email=test_email,
                password=test_password
            )
            if isinstance(result, int):
                print(f"[OK] User created, ID: {result}")
            else:
                print(f"[FAIL] Create user failed: {result}")
                sys.exit(1)

            print(f"\nStep 2: Login")
            try:
                token = login(BASE_URL, test_username, test_password)
                print(f"[OK] Login successful")
            except Exception as e:
                print(f"[FAIL] Login failed: {e}")
                sys.exit(1)

            print(f"\nStep 3: Update user settings")
            result = update_user_settings(BASE_URL, token, test_name_alias)
            if result is True:
                print(f"[OK] User settings updated successfully")
            else:
                print(f"[FAIL] Update failed: {result}")

        elif test_name == 'create_project':
            print("=== Test: Create Project ===")

            print("\nStep 1: Login")
            try:
                token = login(BASE_URL, ADMIN_USERNAME, ADMIN_PASSWORD)
                print(f"[OK] Login successful")
            except Exception as e:
                print(f"[FAIL] Login failed: {e}")
                sys.exit(1)

            print(f"\nStep 2: Create project")
            import time
            project_title = f"测试项目_{int(time.time())}"
            result = create_project(BASE_URL, token, project_title)
            if isinstance(result, int):
                print(f"[OK] Project created successfully, Project ID: {result}")
                print(f"Title: {project_title}")
            else:
                print(f"[FAIL] Create project failed: {result}")

        elif test_name == 'create_team':
            print("=== Test: Create Team ===")

            print("\nStep 1: Login")
            try:
                token = login(BASE_URL, ADMIN_USERNAME, ADMIN_PASSWORD)
                print(f"[OK] Login successful")
            except Exception as e:
                print(f"[FAIL] Login failed: {e}")
                sys.exit(1)

            print(f"\nStep 2: Create team")
            import time
            team_name = f"测试团队_{int(time.time())}"
            result = create_team(BASE_URL, token, team_name)
            if isinstance(result, int):
                print(f"[OK] Team created successfully, Team ID: {result}")
                print(f"Name: {team_name}")
            else:
                print(f"[FAIL] Create team failed: {result}")

        elif test_name == 'share_project':
            print("=== Test: Share Project with User ===")
            import time

            timestamp = str(int(time.time()))

            print("\nStep 1: Create test user")
            test_username = f"testuser{timestamp}"
            test_email = f"testuser{timestamp}@example.com"
            test_password = "12345678"
            result = create_user(
                base_url=BASE_URL,
                username=test_username,
                email=test_email,
                password=test_password
            )
            if isinstance(result, int):
                test_user_id = result
                print(f"[OK] Test user created, ID: {test_user_id}")
            else:
                print(f"[FAIL] Create user failed: {result}")
                sys.exit(1)

            print(f"\nStep 2: Update test user settings (make discoverable)")
            test_user_token = login(BASE_URL, test_username, test_password)
            result = update_user_settings(BASE_URL, test_user_token, "测试用户")
            if result is True:
                print(f"[OK] User settings updated")
            else:
                print(f"[FAIL] Update settings failed: {result}")
                sys.exit(1)

            print(f"\nStep 3: Login as admin and create project")
            admin_token = login(BASE_URL, ADMIN_USERNAME, ADMIN_PASSWORD)
            project_title = f"共享测试项目_{timestamp}"
            result = create_project(BASE_URL, admin_token, project_title)
            if isinstance(result, int):
                project_id = result
                print(f"[OK] Project created, ID: {project_id}")
            else:
                print(f"[FAIL] Create project failed: {result}")
                sys.exit(1)

            print(f"\nStep 4: Share project with test user")
            result = share_project_with_user(BASE_URL, admin_token, project_id, test_username, 1)
            if result is True:
                print(f"[OK] Project shared successfully")
            else:
                print(f"[FAIL] Share project failed: {result}")

        elif test_name == 'add_team_member':
            print("=== Test: Add Team Member ===")
            import time

            timestamp = str(int(time.time()))

            print("\nStep 1: Create test user")
            test_username = f"testuser{timestamp}"
            test_email = f"testuser{timestamp}@example.com"
            test_password = "12345678"
            result = create_user(
                base_url=BASE_URL,
                username=test_username,
                email=test_email,
                password=test_password
            )
            if isinstance(result, int):
                print(f"[OK] Test user created, ID: {result}")
            else:
                print(f"[FAIL] Create user failed: {result}")
                sys.exit(1)

            print(f"\nStep 2: Update test user settings (make discoverable)")
            test_user_token = login(BASE_URL, test_username, test_password)
            result = update_user_settings(BASE_URL, test_user_token, "测试用户")
            if result is True:
                print(f"[OK] User settings updated")
            else:
                print(f"[FAIL] Update settings failed: {result}")
                sys.exit(1)

            print(f"\nStep 3: Login as admin and create team")
            admin_token = login(BASE_URL, ADMIN_USERNAME, ADMIN_PASSWORD)
            team_name = f"测试团队_{timestamp}"
            result = create_team(BASE_URL, admin_token, team_name)
            if isinstance(result, int):
                team_id = result
                print(f"[OK] Team created, ID: {team_id}")
            else:
                print(f"[FAIL] Create team failed: {result}")
                sys.exit(1)

            print(f"\nStep 4: Add test user to team")
            result = add_team_member(BASE_URL, admin_token, team_id, test_username)
            if result is True:
                print(f"[OK] Team member added successfully")
            else:
                print(f"[FAIL] Add team member failed: {result}")

        else:
            print(f"Unknown test: {test_name}")
            print("Available tests: login, get_user, create_user, update_user_settings, create_project, create_team, share_project, add_team_member")
    else:
        print("Usage: python api_utils.py <test_name>")
        print("Available tests: login, get_user, create_user, update_user_settings, create_project, create_team, share_project, add_team_member")
