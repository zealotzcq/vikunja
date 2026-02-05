#!/usr/bin/env python3
"""
Vikunja API utility functions
"""

import requests
from typing import Dict, List, Optional, Any, Union

DEFAULT_PASSWORD = "12345678"


class VikunjaAPI:
    """Vikunja API client"""

    def __init__(self, base_url: str, username: str, password: str):
        self.base_url = base_url.rstrip('/')
        self.username = username
        self.password = password
        self.token = None
        self._authenticate()

    def _authenticate(self):
        """Authenticate and get JWT token"""
        url = f"{self.base_url}/api/v1/login"
        response = requests.post(
            url,
            json={'username': self.username, 'password': self.password}
        )
        response.raise_for_status()
        self.token = response.json().get('token')
        if not self.token:
            raise Exception("Failed to obtain authentication token")

    def _request(self, method: str, endpoint: str, data: Optional[dict] = None) -> Union[dict, List[dict]]:
        """Make authenticated API request"""
        url = f"{self.base_url}/api/v1{endpoint}"
        headers = {
            'Authorization': f'Bearer {self.token}',
            'Content-Type': 'application/json'
        }
        response = requests.request(method, url, headers=headers, json=data)
        response.raise_for_status()
        return response.json()

    def _request_dict(self, method: str, endpoint: str, data: Optional[dict] = None) -> dict:
        """Make authenticated API request and ensure dict return"""
        result = self._request(method, endpoint, data)
        if isinstance(result, dict):
            return result
        return {}

    def _request_list(self, method: str, endpoint: str, data: Optional[dict] = None) -> List[dict]:
        """Make authenticated API request and ensure list return"""
        result = self._request(method, endpoint, data)
        if isinstance(result, list):
            return result
        if isinstance(result, dict):
            return [result]
        return []

    def search_users(self, search: str = "") -> List[dict]:
        """Search for users"""
        try:
            return self._request_list('GET', f'/users?s={search}')
        except requests.exceptions.HTTPError as e:
            if e.response.status_code == 401:
                raise Exception("Authentication failed. Check credentials.")
            raise

    def create_user(self, username: str, email: str, password: str, name: Optional[str] = None) -> dict:
        """Create new user"""
        data = {
            'username': username,
            'email': email,
            'password': password
        }
        if name:
            data['name'] = name
        return self._request_dict('POST', '/register', data)

    def add_team_member(self, team_id: int, username: str) -> dict:
        """Add user to team"""
        return self._request_dict('PUT', f'/teams/{team_id}/members', {'username': username})

    def get_user(self, user_id: Optional[int] = None) -> dict:
        """Get user information"""
        if user_id:
            return self._request_dict('GET', f'/users/{user_id}')
        return self._request_dict('GET', '/user')

    def update_user_settings(self, data: dict) -> dict:
        """Update user settings"""
        headers = {
            'Authorization': f'Bearer {self.token}',
            'Content-Type': 'application/json'
        }
        url = f"{self.base_url}/api/v1/user/settings/general"
        response = requests.post(url, headers=headers, json=data)
        response.raise_for_status()
        return response.json()

    def get_current_user_with_settings(self) -> dict:
        """Get current user including settings structure"""
        try:
            # First get basic user info
            user = self.get_user()
            
            # If user has 'settings' field, return it directly
            if 'settings' in user:
                return user
            else:
                # Return user with empty settings structure matching format
                return {
                    **user,
                    'settings': {
                        'name': '',
                        'email_reminders_enabled': False,
                        'discoverable_by_name': False,
                        'discoverable_by_email': False,
                        'overdue_tasks_reminders_enabled': False,
                        'overdue_tasks_reminders_time': '9:00',
                        'default_project_id': 0,
                        'week_start': 0,
                        'language': '',
                        'timezone': 'GMT',
                        'frontend_settings': None,
                        'extra_settings_links': None
                    }
                }
        except requests.exceptions.HTTPError as e:
            if e.response.status_code == 401:
                raise Exception("Authentication failed. Check credentials.")
            raise

    def create_project(self, title: str, owner_id: Optional[int] = None, description: str = "") -> dict:
        """Create new project"""
        data: dict = {'title': title}
        if description:
            data['description'] = description
        if owner_id is not None:
            data['owner'] = {'id': owner_id}
        return self._request_dict('PUT', '/projects', data)

    def share_project_with_user(self, project_id: int, username: str, permission: int = 1) -> dict:
        """Share project with user by username. Permission: 0=Read, 1=Read & Write, 2=Admin"""
        return self._request_dict('PUT', f'/projects/{project_id}/users', {
            'username': username,
            'permission': permission
        })

    def share_project_with_user_id(self, project_id: int, user_id: int, permission: int = 1) -> dict:
        """Share project with user by ID. Permission: 0=Read, 1=Read & Write, 2=Admin"""
        return self._request_dict('PUT', f'/projects/{project_id}/users', {
            'user_id': str(user_id),
            'project_id': str(project_id),
            'permission': permission
        })

    def get_user_by_username(self, username: str) -> Optional[dict]:
        """Get user by username"""
        users = self.search_users(username)
        for user in users:
            if user.get('username') == username:
                return user
        return None


if __name__ == '__main__':
    import sys

    BASE_URL = "http://127.0.0.1:3456"
    ADMIN_USERNAME = "王小牛"
    ADMIN_PASSWORD = "12345678"

    if len(sys.argv) > 1:
        test_name = sys.argv[1]

        if test_name == 'test1':
            print("=== Test 1: Login and Search Users ===")
            try:
                api = VikunjaAPI(BASE_URL, ADMIN_USERNAME, ADMIN_PASSWORD)
                print(f"[OK] Login successful")

                users = api.search_users("")
                print(f"[OK] Found {len(users)} users")
                for user in users[:3]:
                    print(f"  - {user.get('username')} ({user.get('name')})")
            except Exception as e:
                print(f"[FAIL] Error: {e}")

        elif test_name == 'test2':
            print("=== Test 2: Create New User ===")
            test_username = "testuser999"
            test_email = "testuser999@example.com"
            try:
                api = VikunjaAPI(BASE_URL, ADMIN_USERNAME, ADMIN_PASSWORD)
                print(f"[OK] Login successful")

                user = api.create_user(
                    username=test_username,
                    email=test_email,
                    password=DEFAULT_PASSWORD,
                    name="Test User"
                )
                print(f"[OK] Created user with ID: {user.get('id')}")
            except Exception as e:
                print(f"[FAIL] Error: {e}")

        elif test_name == 'test3':
            print("=== Test 3: Get User Info ===")
            try:
                api = VikunjaAPI(BASE_URL, ADMIN_USERNAME, ADMIN_PASSWORD)
                print(f"[OK] Login successful")

                user = api.get_user()
                print(f"[OK] User info:")
                print(f"  - Username: {user.get('username')}")
                print(f"  - Name: {user.get('name')}")
                print(f"  - Email: {user.get('email')}")
            except Exception as e:
                print(f"[FAIL] Error: {e}")

        elif test_name == 'test4':
            print("=== Test 4: Update User Settings ===")
            try:
                api = VikunjaAPI(BASE_URL, ADMIN_USERNAME, ADMIN_PASSWORD)
                print(f"[OK] Login successful")

                current = api.get_current_user_with_settings()
                print(f"[OK] Got current user with settings")

                settings_data = {
                    'name': '小牛',
                    'discoverable_by_name': True,
                    'discoverable_by_email': True,
                    'language': 'zh-CN'
                }

                result = api.update_user_settings(settings_data)
                print(f"[OK] Updated settings")
            except Exception as e:
                print(f"[FAIL] Error: {e}")

        elif test_name == 'test5':
            print("=== Test 5: Create Project ===")
            try:
                api = VikunjaAPI(BASE_URL, ADMIN_USERNAME, ADMIN_PASSWORD)
                print(f"[OK] Login successful")

                project = api.create_project(
                    title="Test Project from utility.py",
                    description="Test project"
                )
                print(f"[OK] Created project with ID: {project.get('id')}")
                print(f"  Title: {project.get('title')}")
            except Exception as e:
                print(f"[FAIL] Error: {e}")

        elif test_name == 'test6':
            print("=== Test 6: Share Project with User ===")
            try:
                api = VikunjaAPI(BASE_URL, ADMIN_USERNAME, ADMIN_PASSWORD)
                print(f"[OK] Login successful")

                user_info = api.get_user_by_username(ADMIN_USERNAME)
                if not user_info:
                    print(f"[FAIL] User '{ADMIN_USERNAME}' not found")
                    sys.exit(1)
                print(f"[OK] Found user ID: {user_info.get('id')}")

                project = api.create_project(
                    title="Shared Test Project",
                    description="Test project for sharing"
                )
                print(f"[OK] Created project ID: {project.get('id')}")

                share_result = api.share_project_with_user(
                    project_id=project['id'],
                    username=ADMIN_USERNAME,
                    permission=1
                )
                print(f"[OK] Shared project with user")
            except Exception as e:
                print(f"[FAIL] Error: {e}")

        elif test_name == 'test7':
            print("=== Test 7: Get User by Username ===")
            try:
                api = VikunjaAPI(BASE_URL, ADMIN_USERNAME, ADMIN_PASSWORD)
                print(f"[OK] Login successful")

                user = api.get_user_by_username(ADMIN_USERNAME)
                if user:
                    print(f"[OK] Found user:")
                    print(f"  - ID: {user.get('id')}")
                    print(f"  - Username: {user.get('username')}")
                    print(f"  - Name: {user.get('name')}")
                    print(f"  - Email: {user.get('email')}")
                else:
                    print(f"[FAIL] User '{ADMIN_USERNAME}' not found")
            except Exception as e:
                print(f"[FAIL] Error: {e}")

        elif test_name == 'all':
            print("=== Running All Tests ===\n")
            for i in range(1, 8):
                print(f"\n{'='*50}")
                sys.argv[1] = f'test{i}'
                exec(open(__file__).read())
            print(f"\n{'='*50}")
            print("=== All Tests Completed ===")

        else:
            print(f"Unknown test: {test_name}")
            print("Available tests: test1, test2, test3, test4, test5, test6, test7, all")
            print("\nTest descriptions:")
            print("  test1 - Login and search users")
            print("  test2 - Create new user")
            print("  test3 - Get user info")
            print("  test4 - Update user settings")
            print("  test5 - Create project")
            print("  test6 - Share project with user")
            print("  test7 - Get user by username")
            print("  all   - Run all tests")
    else:
        print("Usage: python utility.py <test_name>")
        print("\nAvailable tests: test1, test2, test3, test4, test5, test6, test7, all")
        print("\nTest descriptions:")
        print("  test1 - Login and search users")
        print("  test2 - Create new user")
        print("  test3 - Get user info")
        print("  test4 - Update user settings")
        print("  test5 - Create project")
        print("  test6 - Share project with user")
        print("  test7 - Get user by username")
        print("  all   - Run all tests")
