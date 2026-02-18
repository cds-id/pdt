# Vite Tailwind Boilerplate - Project Guide

This document provides an overview of the project architecture, flow, and key components to help AI assistants understand the project structure without needing to scan all files.

## Project Architecture

The project follows a simplified architecture with these key layers:

### 1. Infrastructure Layer
- **Location**: `src/infrastructure/`
- **Purpose**: Contains API services, Redux slices, and API constants
- **Key Components**:
  - `services/`: RTK Query API services
  - `slices/`: Redux slices for state management
  - `constants/`: API endpoints and configuration

### 2. Domain Layer
- **Location**: `src/domain/`
- **Purpose**: Contains interfaces and types only (no use cases)
- **Key Components**:
  - `auth/interfaces/`: Authentication-related interfaces
  - `user/interfaces/`: User-related interfaces

### 3. Presentation Layer
- **Location**: `src/presentation/`
- **Purpose**: Contains React components, pages, layouts, and routes
- **Key Components**:
  - `pages/`: Page components
  - `layouts/`: Layout components for page structure
  - `routes/`: Routing configuration
  - `components/`: Custom UI components (legacy, being replaced by shadcn/ui)

### 4. UI Components
- **Location**: `src/components/ui/`
- **Purpose**: Contains shadcn/ui components
- **Key Components**: Button, Card, Input, Label, Avatar, etc.

### 5. Utils
- **Location**: `src/utils/`
- **Purpose**: Utility functions
- **Key Components**:
  - `auth.ts`: Authentication utility functions
  - `project-setup/`: Tools for project initialization

## Authentication Flow

1. **Login Process**:
   - User enters credentials in `LoginPage.tsx`
   - Form is handled by `react-hook-form`
   - On submit, `useLoginMutation` hook from RTK Query is called
   - Upon successful login, token is stored in local storage
   - User info is extracted and stored in Redux state
   - User is redirected to the dashboard

2. **Token Validation**:
   - `utils/auth.ts` provides `isAuthenticated()` function to check token validity
   - Token is checked for existence and expiration

3. **Layout-based Protection**:
   - `PublicLayout.tsx`: For unauthenticated pages (e.g., login)
     - Redirects to dashboard if user is authenticated
   - `ProtectedLayout.tsx`: For authenticated pages (e.g., dashboard)
     - Redirects to login if user is not authenticated
     - Provides user info and logout functionality

## Routing

- **Location**: `src/presentation/routes/index.tsx`
- Uses React Router v6 with layout-based routing
- Routes are wrapped in appropriate layouts (Public or Protected)

## State Management

- **Redux Toolkit**: Used for global state management
- **RTK Query**: Used for API calls
  - API services are defined in `infrastructure/services/`
  - Hooks are exported directly from service files (e.g., `useLoginMutation`)

## UI Components

- **shadcn/ui**: A component library built on Radix UI and styled with Tailwind CSS
- Components are imported from `@/components/ui/`
- Styling follows a consistent design system with theme variables

## Key Features

1. **Dashboard**: Displays user statistics and information
2. **Authentication**: Token-based with expiration check
3. **Project Setup Utilities**: Tools for icon generation and web manifest creation

## Development Workflow

1. **Scripts**:
   - `npm run dev`: Start development server
   - `npm run build`: Build for production
   - `npm run lint`: Run ESLint
   - `npm run lint:fix`: Run ESLint with auto-fix
   - `npm run commit`: Use commitizen for standardized commits
   - `npm run push`: Push to the remote repository

2. **Dependencies**:
   - React v18
   - TypeScript v5.3.3
   - Redux Toolkit
   - Tailwind CSS
   - shadcn/ui
   - React Router v6
