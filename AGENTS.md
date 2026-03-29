# AGENT Instructions

Vikunja is a comprehensive todo and task management application with a Vue.js frontend and Go backend.

## Project Structure
- `pkg/` – Go code for the API service
- `frontend/` – Vue.js based web client
- `desktop/` – Electron wrapper application

## Agent Environment

The AI agent has access to:
- **Date**: Current date
- **User ID**: Authenticated user's ID
- **Company ID**: Company context for request
- **Language**: User's preferred language
- **Current Task**: Currently selected task (ID and title)
- **Current Route**: Current frontend route name and params
- **Subordinate Staff**: List of subordinate staff members
- **Message History**: Recent conversation messages
- **Chat Session**: Persistent session with 1-hour TTL

## Build Environment

### MinGW64 (GCC)
- GCC: `F:\msys64\mingw64\bin\gcc.exe`
- G++: `F:\msys64\mingw64\bin\g++.exe`

### Build Commands
**Backend (Go)**:
```bash
export CC="/f/msys64/mingw64/bin/gcc.exe"
export CXX="/f/msys64/mingw64/bin/g++.exe"
export PATH="/f/msys64/mingw64/bin:$PATH"
mage build
```

**Frontend**:
```bash
cd frontend && pnpm build
```

### ADB Path
Android SDK: `D:\android_sdk\platform-tools\adb.exe`

## Development Commands

**Backend**:
- `mage build` - Build Go binary
- `mage lint:fix` - Run golangci-lint with auto-fix

**Frontend**:
- `pnpm build` - Production build
- `pnpm lint:fix && pnpm lint:styles:fix` - Fix linting errors

## Architecture

**Backend**: Layered architecture with Models, Services, Routes, Web layers
**Frontend**: Vue 3 Composition API with Pinia state management

## API Documentation

Reference: `pkg/swagger/swagger.json`
Common endpoints: `/login`, `/user`, `/teams/{id}`, `/users`, `/projects`

## Commit Messages

Use **Conventional Commits**: `feat:`, `fix:`, `refactor:`, etc.

## Code Style

- **Go**: golangci-lint, goimports, enforce permissions
- **Vue**: ESLint + TS, single quotes, no semicolons, tab indent
