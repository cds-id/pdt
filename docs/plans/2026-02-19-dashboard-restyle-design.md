# PDT Dashboard Restyle + Register Page Design

## Overview

Restyle all dashboard pages and layout to match the landing page's PDT branding (dark background, yellow accents, off-white text). Create reusable shared components. Add a dedicated Register/Signup page.

## Design System — Shared Components

All components in `src/presentation/components/common/`.

| Component | Purpose | Key Props |
|-----------|---------|-----------|
| `PageHeader` | Page title + subtitle + optional action | `title`, `description`, `action` (ReactNode) |
| `StatCard` | Metric display with icon | `title`, `value`, `icon`, `description`, `trend` |
| `DataCard` | Dark card wrapper for sections | `title`, `children`, `action` |
| `FilterBar` | Horizontal filter row | `children` (filter inputs) |
| `StatusBadge` | Colored status pill | `status`, `variant` (success/warning/info/neutral) |
| `EmptyState` | No-data placeholder | `icon`, `title`, `description`, `action` |

### Color Rules

- Page backgrounds: `bg-pdt-primary` (#1B1B1E)
- Cards: `bg-pdt-primary-light` (#2d2d32) with `border border-pdt-background/20`
- Primary text: `text-pdt-neutral` (#FBFFFE)
- Secondary text: `text-pdt-neutral/60`
- Accents/highlights: `text-pdt-background` (#F8C630)
- Buttons: `pdt` and `pdtOutline` variants from button.tsx

## Dashboard Layout Restyle

### Sidebar
- Background: `bg-pdt-primary`
- Nav items: `text-pdt-neutral/70`, active: `text-pdt-background` with left yellow border
- Hover: `bg-pdt-primary-light`

### Header
- Background: `bg-pdt-primary-light` with bottom border `border-pdt-background/20`
- User dropdown: Dark theme

### Main content area
- Background: `bg-pdt-primary`

## Register/Signup Page

New page at `/register`:
- Yellow background (`bg-pdt-background`)
- Centered dark card (`bg-pdt-primary`)
- Fields: Name, Email, Password
- Form validation: react-hook-form + yup
- On success: auto-login, redirect to `/dashboard/home`
- Link to login page

## Page-by-Page Restyle

### DashboardHomePage (`/dashboard/home`)
- PageHeader: "Welcome back, {name}" + sync status
- 3x StatCard: Total Commits, Jira Linked %, Active Sprint
- DataCard: Recent Commits table
- Sync Now button (pdt variant)

### ReposPage (`/dashboard/repos`)
- PageHeader: "Repositories" + "Add Repository" action
- Add repo form in DataCard
- Repo list with provider icons and StatusBadge (valid/invalid)

### CommitsPage (`/dashboard/commits`)
- PageHeader: "Commits"
- FilterBar: Repository dropdown, Jira key search, unlinked toggle
- Commits table in DataCard with inline Jira linking

### JiraPage (`/dashboard/jira`)
- PageHeader: "Jira Integration"
- Active sprint DataCard with cards grid
- StatusBadge for card statuses
- All sprints list

### ReportsPage (`/dashboard/reports`)
- PageHeader: "Reports" + "Generate Report" action
- Report generator form in DataCard
- Past reports list

### SettingsPage (`/dashboard/settings`)
- PageHeader: "Settings"
- Integration sections in DataCard blocks
- StatusBadge for configured/not configured
- Save + Validate buttons

## Route Changes

- Remove DashboardPage, `/dashboard` redirects to `/dashboard/home`
- `/register` renders new RegisterPage

## Files

### New
- `src/presentation/components/common/PageHeader.tsx`
- `src/presentation/components/common/StatCard.tsx`
- `src/presentation/components/common/DataCard.tsx`
- `src/presentation/components/common/FilterBar.tsx`
- `src/presentation/components/common/StatusBadge.tsx`
- `src/presentation/components/common/EmptyState.tsx`
- `src/presentation/pages/RegisterPage.tsx`

### Modified
- `src/presentation/routes/index.tsx`
- `src/presentation/layouts/DashboardLayout.tsx`
- `src/presentation/layouts/components/Header.tsx`
- `src/presentation/components/dashboard/Sidebar/Sidebar.tsx`
- `src/presentation/pages/DashboardHomePage.tsx`
- `src/presentation/pages/ReposPage.tsx`
- `src/presentation/pages/CommitsPage.tsx`
- `src/presentation/pages/JiraPage.tsx`
- `src/presentation/pages/ReportsPage.tsx`
- `src/presentation/pages/SettingsPage.tsx`
- `src/presentation/pages/LoginPage.tsx`
