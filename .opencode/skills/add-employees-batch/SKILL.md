---
name: add-employees-batch
description: Batch add multiple new employees to company team. This skill should be used when users explicitly request to add new employees.
---

# Batch Add Employees

This skill provides a workflow for batch adding multiple new employees to a Vikunja instance.

## Purpose

Streamline the employee onboarding process by enabling batch creation of multiple user accounts in a single operation.

## When to Use

Use this skill when users explicitly request to add new employees to the system.

## Workflow

### Step 1: Parse Employee Input

Parse the employee input from the user. Each line should contain three parts separated by `|`:
- First part: Employee's full name (e.g., "王小牛")
- Second part: Employee's nickname/calling name (e.g., "小王")
- Third part: Employee's email address (e.g., "wang@example.com")

Only process lines that match this pattern (three parts separated by `|`).

Example input:
```
王小牛|小王|wang@example.com
张三|老张|zhang@example.com
```

### Step 2: Validate Configuration

Ensure that configuration file exists and contains:
- `api.base_url`: Vikunja API base URL
- `admin.username`: Admin account username
- `admin.password`: Admin account password
- `team.id`: Default team ID to add new employees to (optional)

### Step 3: Load Configuration

Load the configuration from `assets/batch_employees_config.yml` or prompt for missing values.

### Step 4: Process Each Employee

For each employee in the batch:
1. Use the full name as the username
2. Use the provided email address
3. Set default password to "12345678"
4. Register a new account using the Vikunja API
5. If username or email already exists, skip this employee and continue with the next one
6. Add the new user to the specified team (if team_id is configured)
7. Login with the new user's account
8. Update user settings:
   - Set "My Name" to the nickname
   - Enable "Allow others to discover me"
9. Record the username
10. Using the configured admin account, create a project named "User xxx's Work List" (xxx is the recorded username)
11. Share the project with the new user with read/write permissions

### Step 5: Generate Summary

Provide a summary of batch operation:
- List of users successfully created
- List of users not created with reasons for failure
- Number of users skipped (already existed)
- Any errors encountered

## Scripts

- `scripts/batch_add_employees.py` - Main Python script that processes employee data
- `scripts/utility.py` - Vikunja API utility functions

### Using the Batch Script

The script reads employee data from a JSON file:

```bash
python .opencode/skills/add-employees-batch/scripts/batch_add_employees.py --input employees.json
```

Or parse employee data from text input:

```bash
python .opencode/skills/add-employees-batch/scripts/batch_add_employees.py --text "王小牛|小王|wang@example.com
张三|老张|zhang@example.com"
```

### Configuration File Format

`assets/batch_employees_config.yml`:

```yaml
team:
  id: 1  # Company team ID (optional)

admin:
  username: "admin"  # Admin account username
  password: "your_password_here"  # Admin account password

api:
  base_url: "http://localhost:8080"  # Vikunja instance base URL (without /api/v1)
```

A configuration template is provided in `assets/batch_employees_config.yml.example`. Copy it to `assets/batch_employees_config.yml` and update with your actual values.

### Input File Format

`employees.json` (array of objects with `name`, `nickname`, and `email` fields):

```json
[
  {
    "name": "王小牛",
    "nickname": "小王",
    "email": "wang@example.com"
  },
  {
    "name": "张三",
    "nickname": "老张",
    "email": "zhang@example.com"
  }
]
```

### Configuration File Format

`config/batch_employees_config.json`:

```json
{
  "base_url": "https://vikunja.example.com",
  "username": "admin",
  "password": "your_password",
  "team_id": 1
}
```

### Input File Format

`employees.json` (array of objects with `name`, `nickname`, and `email` fields):

```json
[
  {
    "name": "王小牛",
    "nickname": "小王",
    "email": "wang@example.com"
  },
  {
    "name": "张三",
    "nickname": "老张",
    "email": "zhang@example.com"
  }
]
```

## Error Handling

Handle common errors:
- Duplicate usernames or emails (skip existing users)
- API authentication failures (prompt for correct credentials)
- Network errors (retry with backoff)
- Invalid input format (report and continue with valid entries)

## Security Considerations

- The default password "12345678" should be changed by users on first login
- Store configuration files securely with proper file permissions
- Log operations for audit purposes
- Never log or store passwords in plain text
