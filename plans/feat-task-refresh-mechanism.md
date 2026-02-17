# Implement Task Refresh Mechanism

## Overview
Add a periodic task refresh mechanism that automatically resets tasks to incomplete based on project title patterns:
- `#0#` - Always refresh (every access)
- `#1#` - Daily refresh (different local calendar day)
- `#2#` - Weekly refresh
- `#3#` - Monthly refresh

## Requirements

### Frontend
- Local persistence of project ID -> last refresh time map
- On project access, check if refresh is needed based on:
  - Current time vs last refresh time
  - Project title ending pattern (#0#, #1#, #2#, #3#)
- Refresh rules:
  - #0# - Always refresh (every time the project is accessed)
    - Stores yesterday's time in localStorage
    - Sends tomorrow's start time to backend
    - Effectively resets all tasks every access
  - #1# - Daily refresh (different local calendar day)
  - #2# - Weekly refresh
  - #3# - Monthly refresh
- Call refresh API with:
  - Project ID
  - Start of day/week/month/tomorrow timestamp (in database-compatible format)
- Update local last refresh time after successful API call

### Backend
- Add API endpoint: `POST /api/v1/projects/{id}/refresh-tasks`
- Accept parameters:
  - `sinceTime` - timestamp string
- Logic:
  - Find all tasks in the project
  - Identify tasks with `updated_at` < `sinceTime`
  - Mark those tasks as incomplete

## Implementation Steps

### 1. Backend API Development
- [x] Create API endpoint in `pkg/routes/api/v1/`
- [x] Add route handler for `POST /projects/{id}/refresh-tasks`
- [x] Implement task refresh logic in `pkg/services/` or model
- [x] Add Swagger annotations
- [x] Test the endpoint

### 2. Frontend Data Structure
- [x] Create composable for refresh time persistence (localStorage)
- [x] Implement time comparison logic for daily/weekly/monthly rules
- [x] Add special handling for #0# (always refresh)
- [x] Add API service call for refresh endpoint

### 3. Frontend Integration
- [x] Integrate refresh check into project view loading
- [x] Parse project title for refresh pattern (#0#, #1#, #2#, #3#)
- [x] Calculate start of day/week/month/tomorrow timestamps
- [x] Handle success/error responses
- [x] Update local storage on success
- [x] Store yesterday's time for #0# pattern

### 4. Testing
- [ ] Backend tests for refresh endpoint
- [ ] Frontend tests for time calculation logic
- [ ] Integration tests
- [ ] Manual testing scenarios

### 5. Documentation
- [ ] Update API documentation
- [ ] Add feature description to user docs
- [x] Add debugging logs for troubleshooting

## Debugging

### Frontend Logs
When opening a project with a refresh pattern, check the browser console for logs prefixed with `[frontend]`:

1. **Initialization**: `useProjectTaskRefresh initialized with project: {...}`
2. **Pattern Detection**: `Extracted refresh pattern: #0# from title: ...`
3. **Last Refresh Time**: `Last refresh time for project X: never` or `...format()`
4. **Should Refresh**: `shouldRefreshBasedOnPattern: true pattern: #0#`
5. **Refresh Trigger**: `Project loaded, checking for task refresh...`
6. **API Request**: `Sending refresh request: {...}`
7. **Response**: `Refresh response status: 200 OK`
8. **Complete**: `Tasks refreshed for project X`

### Common Issues

**No logs appearing:**
- Check that the project title ends with exactly one of: `#0#`, `#1#`, `#2#`, `#3#`
- Ensure the project is loaded successfully
- Check browser console for any errors

**Should Refresh is false:**
- Check if lastRefreshTime is set in localStorage (key: `taskRefreshTimestamps`)
- For `#0#`, shouldRefresh should always be true
- For `#1#`, check if the current day is different from last refresh day

**API Request Failing:**
- Check response status and error text in console
- Verify backend is running and accessible at `/api/v1/projects/{id}/refresh-tasks`
- Check network tab for actual request/response

## Technical Notes

### Time Handling
- Frontend uses local time for comparisons
- Backend `updated_at` field format needs to match (check database structure)
- Time format conversion must be consistent
- For #0# pattern: stores yesterday, sends tomorrow (ensures always refresh)

### Task Status Change
- Need to identify the correct field for "completed" status
- Check if task model has is_done, done_at, or similar field

## Usage Examples

### Always Refresh (#0#)
Project title: `Daily Reset Tasks#0#`
- Every time the project is accessed, all tasks are reset to incomplete
- Good for tasks that need to be re-done daily

### Daily Refresh (#1#)
Project title: `My Daily Tasks#1#`
- Tasks are reset to incomplete when the day changes
- Keeps progress within the same day

### Weekly Refresh (#2#)
Project title: `Weekly Goals#2#`
- Tasks are reset at the start of each week

### Monthly Refresh (#3#)
Project title: `Monthly Review#3#`
- Tasks are reset at the start of each month
