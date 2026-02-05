#!/usr/bin/env python3
"""
Batch add employees to Vikunja
"""

import argparse
import json
import os
import sys
from datetime import datetime
from typing import Dict, List, Optional, Tuple

from utility import VikunjaAPI

try:
    import requests
except ImportError:
    print("Error: requests library is required. Install with: pip install requests")
    sys.exit(1)

try:
    import yaml
except ImportError:
    print("Error: PyYAML library is required. Install with: pip install pyyaml")
    sys.exit(1)


DEFAULT_PASSWORD = "12345678"
CONFIG_PATH = "../assets/batch_employees_config.yml"


def load_config(config_path: str) -> dict:
    """Load configuration from YAML file"""
    if os.path.exists(config_path):
        with open(config_path, 'r', encoding='utf-8') as f:
            config = yaml.safe_load(f)

        # Normalize config structure
        result = {}
        if config:
            if 'api' in config and 'base_url' in config['api']:
                result['base_url'] = config['api']['base_url']
            if 'admin' in config:
                result['username'] = config['admin'].get('username')
                result['password'] = config['admin'].get('password')
            if 'team' in config and 'id' in config['team']:
                result['team_id'] = config['team']['id']

        return result
    return {}


def load_employees(input_path: str) -> List[dict]:
    """Load employee data from JSON file"""
    with open(input_path, 'r', encoding='utf-8') as f:
        return json.load(f)


def parse_employee_input(input_text: str) -> List[dict]:
    """Parse employee input from text lines"""
    employees = []
    lines = input_text.strip().split('\n')

    for line in lines:
        line = line.strip()
        if not line:
            continue

        parts = line.split('|')
        if len(parts) >= 3:
            name = parts[0].strip()
            nickname = parts[1].strip()
            email = parts[2].strip()
            employees.append({
                'name': name,
                'nickname': nickname,
                'email': email
            })

    return employees


def validate_employee_data(employees: List[dict]) -> Tuple[List[dict], List[str]]:
    """Validate employee data"""
    valid_employees = []
    errors = []

    for idx, emp in enumerate(employees, 1):
        emp_errors = []

        if not emp.get('name'):
            emp_errors.append('missing name')

        if not emp.get('nickname'):
            emp_errors.append('missing nickname')

        if not emp.get('email'):
            emp_errors.append('missing email')

        if '@' not in emp.get('email', ''):
            emp_errors.append('invalid email format')

        if emp_errors:
            errors.append(f"Employee {idx}: {', '.join(emp_errors)}")
        else:
            valid_employees.append(emp)

    return valid_employees, errors


def user_exists(api: VikunjaAPI, username: str = None, email: str = None) -> bool:
    """Check if user already exists by username or email"""
    try:
        users = api.search_users("")
        for user in users:
            if username and user.get('username') == username:
                return True
            if email and user.get('email') == email:
                return True
        return False
    except Exception as e:
        print(f"Warning: Failed to check if user exists: {e}")
        return False


def process_employee(api: VikunjaAPI, employee: dict, team_id: int = None, verbose: bool = False) -> dict:
    """Process a single employee"""
    name = employee['name']
    nickname = employee['nickname']
    email = employee['email']

    result = {
        'name': name,
        'nickname': nickname,
        'username': name,
        'email': email,
        'success': False,
        'skipped': False,
        'errors': []
    }

    try:
        # Check if user already exists by username or email
        if user_exists(api, username=name, email=email):
            if verbose:
                print(f"  User '{name}' or email '{email}' already exists, skipping")
            result['skipped'] = True
            result['success'] = True
            return result

        # Create user with default password
        if verbose:
            print(f"  Creating user: {name} ({email}, password: {DEFAULT_PASSWORD})")

        user = api.create_user(
            username=name,
            email=email,
            password=DEFAULT_PASSWORD,
            name=name
        )
        result['user_id'] = user['id']

        # Login as the new user to update settings
        try:
            if verbose:
                print(f"  Logging in as {name} to update settings")

            new_user_api = VikunjaAPI(api.base_url, name, DEFAULT_PASSWORD)

            # Update user settings: set name to nickname, language to zh-CN, and enable discoverability
            if verbose:
                print(f"  Updating user settings: name='{nickname}', language='zh-CN', discoverable=true")

            # Get current user with settings structure, copy it, and modify only 4 fields
            current_user = new_user_api.get_current_user_with_settings()
            
            # Copy settings structure and modify only the 4 fields we want to change
            if 'settings' in current_user:
                settings_data = current_user['settings'].copy()
                settings_data['name'] = nickname
                settings_data['discoverable_by_name'] = True
                settings_data['discoverable_by_email'] = True
                settings_data['language'] = 'zh-CN'
            else:
                # Fallback: User doesn't have settings field, create one
                settings_data = {
                    'name': nickname,
                    'discoverable_by_name': True,
                    'discoverable_by_email': True,
                    'language': 'zh-CN'
                }

            try:
                new_user_api.update_user_settings(settings_data)
                result['nickname_updated'] = True
            except requests.exceptions.HTTPError as e:
                if e.response.status_code == 400:
                    result['errors'].append(f"Failed to update user settings (validation error)")
                    if verbose:
                        print(f"  Validation error: {e.response.text}")
                else:
                    raise
        except Exception as e:
            result['errors'].append(f"Failed to update user settings: {str(e)}")

        # Add to team if team_id is specified
        if team_id:
            try:
                if verbose:
                    print(f"  Adding to team: {team_id}")
                api.add_team_member(team_id, name)
                result['team_id'] = team_id
            except Exception as e:
                result['errors'].append(f"Failed to add to team: {str(e)}")

        result['success'] = True

    except requests.exceptions.HTTPError as e:
        if e.response.status_code == 409:
            result['skipped'] = True
            result['success'] = True
            if verbose:
                print(f"  User '{name}' or email '{email}' already exists (conflict), skipping")
        else:
            error_msg = e.response.json().get('message', str(e)) if e.response.json() else str(e)
            result['errors'].append(f"HTTP {e.response.status_code}: {error_msg}")
    except Exception as e:
        result['errors'].append(str(e))

    return result


def print_summary(results: List[dict], start_time: datetime):
    """Print operation summary"""
    end_time = datetime.now()
    duration = (end_time - start_time).total_seconds()

    successful = sum(1 for r in results if r['success'])
    created = sum(1 for r in results if r['success'] and not r['skipped'])
    skipped = sum(1 for r in results if r['skipped'])
    failed = len(results) - successful

    print("\n" + "=" * 60)
    print("BATCH EMPLOYEE ONBOARDING SUMMARY")
    print("=" * 60)
    print(f"Total processed: {len(results)}")
    print(f"Created: {created}")
    print(f"Skipped (already exists): {skipped}")
    print(f"Failed: {failed}")
    print(f"Duration: {duration:.2f} seconds")
    print("=" * 60)

    if created > 0:
        print("\nSuccessfully Created Users:")
        for result in results:
            if result['success'] and not result['skipped']:
                print(f"  - {result['name']} ({result['username']})")
                print(f"    Nickname: {result['nickname']}")
                print(f"    Email: {result['email']}")
                print(f"    Password: {DEFAULT_PASSWORD}")
                if 'team_id' in result:
                    print(f"    Team ID: {result['team_id']}")
                if 'project_id' in result:
                    print(f"    Project: {result.get('project_title', 'N/A')} (ID: {result['project_id']})")
                if result.get('errors'):
                    print(f"    Warnings: {', '.join(result['errors'])}")

    if skipped > 0:
        print("\nSkipped Users (Already Exist):")
        for result in results:
            if result['skipped']:
                print(f"  - {result['name']} ({result['username']})")

    if failed > 0:
        print("\nFailed Users:")
        for result in results:
            if not result['success']:
                print(f"  - {result['name']} ({result['username']})")
                print(f"    Email: {result.get('email', 'N/A')}")
                print(f"    Reason: {', '.join(result['errors'])}")


def main():
    parser = argparse.ArgumentParser(description='Batch add employees to Vikunja')
    parser.add_argument('--config', default=CONFIG_PATH, help='Path to config file')
    parser.add_argument('--input', help='Input JSON file path')
    parser.add_argument('--text', help='Input text (one employee per line: name|nickname|email)')
    parser.add_argument('--verbose', action='store_true', help='Verbose output')

    args = parser.parse_args()

    # Load configuration
    config = load_config(args.config)
    print(f"Loaded config from: {args.config}")

    if not config.get('base_url'):
        config['base_url'] = input("Enter Vikunja base URL (e.g., https://vikunja.example.com): ").strip()

    if not config.get('username'):
        config['username'] = input("Enter username: ").strip()

    if not config.get('password'):
        import getpass
        config['password'] = getpass.getpass("Enter password: ")

    team_id = config.get('team_id')
    if team_id:
        team_id = int(team_id)
    else:
        team_input = input("Enter team ID (press Enter to skip): ").strip()
        if team_input:
            team_id = int(team_input)
        else:
            team_id = None

    # Load employee data
    if args.input:
        employees = load_employees(args.input)
    elif args.text:
        employees = parse_employee_input(args.text)
    else:
        print("\nPaste employee data (one per line: name|nickname|email). Press Enter on empty line to finish:\n")
        lines = []
        while True:
            line = sys.stdin.readline().strip()
            if not line:
                break
            lines.append(line)
        employees = parse_employee_input('\n'.join(lines))

    if not employees:
        print("No employees to process.")
        sys.exit(0)

    print(f"\nLoaded {len(employees)} employee(s)")

    # Validate data
    employees, validation_errors = validate_employee_data(employees)
    if validation_errors:
        print("\nValidation errors:")
        for error in validation_errors:
            print(f"  - {error}")

    if not employees:
        print("\nNo valid employee data to process.")
        sys.exit(1)

    # Initialize API client
    try:
        api = VikunjaAPI(
            config['base_url'],
            config['username'],
            config['password']
        )
    except Exception as e:
        print(f"Error initializing API client: {e}")
        sys.exit(1)

    # Process employees
    start_time = datetime.now()
    results = []

    for idx, employee in enumerate(employees, 1):
        print(f"\nProcessing {idx}/{len(employees)}: {employee['name']}")
        result = process_employee(api, employee, team_id, args.verbose)
        results.append(result)

        if result['success']:
            if result['skipped']:
                print(f"  Skipped (already exists)")
            else:
                print(f"  Success")
        else:
            print(f"  Failed: {', '.join(result['errors'])}")

    # Create projects for successfully created users using the admin account
    successful_results = [r for r in results if r['success'] and not r['skipped']]
    if successful_results:
        print("\n" + "=" * 60)
        print("CREATING PROJECTS FOR NEW USERS")
        print("=" * 60)

        for result in successful_results:
            username = result['username']
            project_title = f"User {username}'s Work List"

            try:
                print(f"\nCreating project '{project_title}' and sharing with {username}")

                # Create project
                project = api.create_project(
                    title=project_title,
                    description=f"Work list for {result['name']}"
                )
                project_id = project['id']

                # Share project with the new user with write permission (permission=1)
                api.share_project_with_user(project_id, username, permission=1)

                result['project_id'] = project_id
                result['project_title'] = project_title
                print(f"  Project created and shared successfully")

            except Exception as e:
                error_msg = str(e)
                result['errors'].append(f"Failed to create/share project: {error_msg}")
                print(f"  Failed: {error_msg}")

    # Print summary
    print_summary(results, start_time)


if __name__ == '__main__':
    main()
