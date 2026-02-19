# PDT Dashboard Restyle + Register Page — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Restyle all dashboard pages and layout to match the landing page's PDT branding, create reusable shared components, and add a Register/Signup page.

**Architecture:** Component-first approach — build 6 shared PDT-styled building blocks first, then restyle the layout (sidebar, header, main area), then restyle each page using the shared components, and finally add the register page. Each task is independently testable.

**Tech Stack:** React 18, TypeScript, Tailwind CSS with PDT tokens (`pdt-primary`, `pdt-background`, `pdt-neutral`), shadcn/ui, react-hook-form, RTK Query, Lucide icons.

**Color tokens (from tailwind.config.mjs):**
- `pdt-primary` = `#1B1B1E` (dark), `pdt-primary-light` = `#2d2d32`, `pdt-primary-dark` = `#000000`
- `pdt-background` = `#F8C630` (yellow)
- `pdt-neutral` = `#FBFFFE` (off-white)
- `pdt-accent` = `#96031A` (red — NOT used for hover, only for destructive actions)

**Button variants (from `src/components/ui/button.tsx`):**
- `pdt`: dark bg, yellow text, border → hover: yellow bg, dark text, dark border
- `pdtOutline`: transparent bg, dark border, dark text → hover: dark bg, yellow text

---

## Task 1: Create PageHeader Component

**Files:**
- Create: `src/presentation/components/common/PageHeader.tsx`

**Step 1: Create the component**

```tsx
import type { ReactNode } from 'react'

interface PageHeaderProps {
  title: string
  description?: string
  action?: ReactNode
}

export function PageHeader({ title, description, action }: PageHeaderProps) {
  return (
    <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
      <div>
        <h1 className="text-2xl font-bold tracking-tight text-pdt-neutral md:text-3xl">
          {title}
        </h1>
        {description && (
          <p className="text-sm text-pdt-neutral/60 md:text-base">{description}</p>
        )}
      </div>
      {action && <div className="flex-shrink-0">{action}</div>}
    </div>
  )
}
```

**Step 2: Verify build**

Run: `cd /home/nst/GolandProjects/pdt/frontend && npx vite build 2>&1 | tail -5`
Expected: Build succeeds

**Step 3: Commit**

```bash
git add src/presentation/components/common/PageHeader.tsx
git commit -m "feat: add PageHeader component with PDT styling"
```

---

## Task 2: Create DataCard Component

**Files:**
- Create: `src/presentation/components/common/DataCard.tsx`

**Step 1: Create the component**

```tsx
import type { ReactNode } from 'react'
import { cn } from '@/lib/utils'

interface DataCardProps {
  title?: string
  action?: ReactNode
  children: ReactNode
  className?: string
}

export function DataCard({ title, action, children, className }: DataCardProps) {
  return (
    <div
      className={cn(
        'rounded-lg border border-pdt-background/20 bg-pdt-primary-light p-4',
        className
      )}
    >
      {(title || action) && (
        <div className="mb-4 flex items-center justify-between">
          {title && (
            <h2 className="text-lg font-semibold text-pdt-neutral">{title}</h2>
          )}
          {action && <div>{action}</div>}
        </div>
      )}
      {children}
    </div>
  )
}
```

**Step 2: Verify build**

Run: `cd /home/nst/GolandProjects/pdt/frontend && npx vite build 2>&1 | tail -5`
Expected: Build succeeds

**Step 3: Commit**

```bash
git add src/presentation/components/common/DataCard.tsx
git commit -m "feat: add DataCard component with PDT styling"
```

---

## Task 3: Create StatusBadge Component

**Files:**
- Create: `src/presentation/components/common/StatusBadge.tsx`

**Step 1: Create the component**

```tsx
import { cn } from '@/lib/utils'

type BadgeVariant = 'success' | 'warning' | 'info' | 'neutral' | 'danger'

interface StatusBadgeProps {
  children: React.ReactNode
  variant?: BadgeVariant
  className?: string
}

const variantStyles: Record<BadgeVariant, string> = {
  success: 'bg-green-500/20 text-green-400',
  warning: 'bg-pdt-background/20 text-pdt-background',
  info: 'bg-blue-500/20 text-blue-400',
  neutral: 'bg-gray-500/20 text-gray-400',
  danger: 'bg-red-500/20 text-red-400'
}

export function StatusBadge({ children, variant = 'neutral', className }: StatusBadgeProps) {
  return (
    <span
      className={cn(
        'inline-flex items-center rounded px-2 py-0.5 text-xs font-medium',
        variantStyles[variant],
        className
      )}
    >
      {children}
    </span>
  )
}
```

**Step 2: Verify build**

Run: `cd /home/nst/GolandProjects/pdt/frontend && npx vite build 2>&1 | tail -5`
Expected: Build succeeds

**Step 3: Commit**

```bash
git add src/presentation/components/common/StatusBadge.tsx
git commit -m "feat: add StatusBadge component with PDT styling"
```

---

## Task 4: Create FilterBar Component

**Files:**
- Create: `src/presentation/components/common/FilterBar.tsx`

**Step 1: Create the component**

```tsx
import type { ReactNode } from 'react'
import { cn } from '@/lib/utils'

interface FilterBarProps {
  children: ReactNode
  className?: string
}

export function FilterBar({ children, className }: FilterBarProps) {
  return (
    <div className={cn('flex flex-wrap items-center gap-3', className)}>
      {children}
    </div>
  )
}
```

**Step 2: Verify build**

Run: `cd /home/nst/GolandProjects/pdt/frontend && npx vite build 2>&1 | tail -5`
Expected: Build succeeds

**Step 3: Commit**

```bash
git add src/presentation/components/common/FilterBar.tsx
git commit -m "feat: add FilterBar component"
```

---

## Task 5: Create EmptyState Component

**Files:**
- Create: `src/presentation/components/common/EmptyState.tsx`

**Step 1: Create the component**

Note: There is an existing `EmptyState` exported from `src/presentation/components/dashboard/EmptyState/` (via the dashboard index). Check if it exists first. If it does, skip this task and use it. If not, create this new one.

```tsx
import type { ReactNode } from 'react'
import type { LucideIcon } from 'lucide-react'
import { cn } from '@/lib/utils'

interface EmptyStateProps {
  icon?: LucideIcon
  title: string
  description?: string
  action?: ReactNode
  className?: string
}

export function EmptyState({ icon: Icon, title, description, action, className }: EmptyStateProps) {
  return (
    <div
      className={cn(
        'rounded-lg border border-pdt-background/20 bg-pdt-primary-light p-8 text-center',
        className
      )}
    >
      {Icon && (
        <div className="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-pdt-background/10">
          <Icon className="h-6 w-6 text-pdt-background" />
        </div>
      )}
      <p className="text-pdt-neutral/60">{title}</p>
      {description && (
        <p className="mt-2 text-sm text-pdt-neutral/40">{description}</p>
      )}
      {action && <div className="mt-4">{action}</div>}
    </div>
  )
}
```

**Step 2: Verify build**

Run: `cd /home/nst/GolandProjects/pdt/frontend && npx vite build 2>&1 | tail -5`
Expected: Build succeeds

**Step 3: Commit**

```bash
git add src/presentation/components/common/EmptyState.tsx
git commit -m "feat: add EmptyState component with PDT styling"
```

---

## Task 6: Create common components barrel export

**Files:**
- Create: `src/presentation/components/common/index.ts`

**Step 1: Create the barrel export**

```ts
export { PageHeader } from './PageHeader'
export { DataCard } from './DataCard'
export { StatusBadge } from './StatusBadge'
export { FilterBar } from './FilterBar'
export { EmptyState } from './EmptyState'
```

**Step 2: Verify build**

Run: `cd /home/nst/GolandProjects/pdt/frontend && npx vite build 2>&1 | tail -5`
Expected: Build succeeds

**Step 3: Commit**

```bash
git add src/presentation/components/common/index.ts
git commit -m "feat: add barrel export for common components"
```

---

## Task 7: Restyle StatsCard to PDT Theme

**Files:**
- Modify: `src/presentation/components/dashboard/StatsCard/StatsCard.tsx`
- Modify: `src/presentation/components/dashboard/StatsCard/StatsCardSkeleton.tsx`

**Step 1: Update StatsCard.tsx**

Replace the entire Card in StatsCard to use PDT colors:

```tsx
import * as React from 'react'

import { cn } from '@/lib/utils'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'

import { TrendIndicator } from './TrendIndicator'
import type { StatsCardProps } from './stats-card.types'

export function StatsCard({
  title,
  value,
  description,
  icon: Icon,
  trend,
  className
}: StatsCardProps) {
  return (
    <Card className={cn('border-pdt-background/20 bg-pdt-primary-light', className)}>
      <CardHeader className="flex flex-row items-center justify-between space-y-0 p-3 pb-1 sm:p-6 sm:pb-2">
        <CardTitle className="text-xs font-medium text-pdt-neutral/70 sm:text-sm">
          {title}
        </CardTitle>
        {Icon && (
          <div className="flex h-8 w-8 items-center justify-center rounded-lg border border-pdt-background/30">
            <Icon className="size-4 text-pdt-background" />
          </div>
        )}
      </CardHeader>
      <CardContent className="p-3 pt-0 sm:p-6 sm:pt-0">
        <div className="text-lg font-bold text-pdt-neutral sm:text-2xl">{value}</div>
        <div className="flex flex-col gap-1 sm:flex-row sm:items-center sm:justify-between">
          {description && (
            <p className="truncate text-[10px] text-pdt-neutral/50 sm:text-xs">
              {description}
            </p>
          )}
          {trend && (
            <TrendIndicator value={trend.value} isPositive={trend.isPositive} />
          )}
        </div>
      </CardContent>
    </Card>
  )
}
```

**Step 2: Update StatsCardSkeleton.tsx**

```tsx
import * as React from 'react'

import { cn } from '@/lib/utils'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'

import type { StatsCardSkeletonProps } from './stats-card.types'

export function StatsCardSkeleton({ className }: StatsCardSkeletonProps) {
  return (
    <Card className={cn('border-pdt-background/20 bg-pdt-primary-light', className)}>
      <CardHeader className="flex flex-row items-center justify-between space-y-0 p-3 pb-1 sm:p-6 sm:pb-2">
        <Skeleton className="h-3 w-16 bg-pdt-neutral/10 sm:h-4 sm:w-24" />
        <Skeleton className="size-8 rounded-lg bg-pdt-neutral/10" />
      </CardHeader>
      <CardContent className="p-3 pt-0 sm:p-6 sm:pt-0">
        <Skeleton className="mb-2 h-5 w-16 bg-pdt-neutral/10 sm:h-8 sm:w-20" />
        <Skeleton className="h-2 w-20 bg-pdt-neutral/10 sm:h-3 sm:w-32" />
      </CardContent>
    </Card>
  )
}
```

**Step 3: Verify build**

Run: `cd /home/nst/GolandProjects/pdt/frontend && npx vite build 2>&1 | tail -5`
Expected: Build succeeds

**Step 4: Commit**

```bash
git add src/presentation/components/dashboard/StatsCard/
git commit -m "style: restyle StatsCard and skeleton to PDT theme"
```

---

## Task 8: Restyle Dashboard Layout (Sidebar + Header + Main)

**Files:**
- Modify: `src/presentation/layouts/DashboardLayout.tsx`
- Modify: `src/presentation/layouts/components/Header.tsx`
- Modify: `src/presentation/layouts/components/MobileNav.tsx`
- Modify: `src/presentation/components/dashboard/Sidebar/Sidebar.tsx`
- Modify: `src/presentation/components/dashboard/Sidebar/SidebarNavItem.tsx`
- Modify: `src/presentation/components/dashboard/Sidebar/SidebarNavGroup.tsx`

**Step 1: Update DashboardLayout.tsx**

Change the main container background from `bg-muted/40` to `bg-pdt-primary`:

In `DashboardLayoutContent`, change:
```tsx
<div className="flex h-screen w-full overflow-hidden bg-muted/40">
```
to:
```tsx
<div className="flex h-screen w-full overflow-hidden bg-pdt-primary">
```

Also change the `<main>` tag to ensure dark background:
```tsx
<main className="flex-1 overflow-y-auto overflow-x-hidden bg-pdt-primary p-3 sm:p-4 lg:p-6">
```

Update BreadcrumbPage and BreadcrumbLink text colors. Add `className` to BreadcrumbPage:
- BreadcrumbPage: add `className="text-pdt-neutral"`
- BreadcrumbLink: add `className="text-pdt-neutral/60 hover:text-pdt-background"`

**Step 2: Update Sidebar.tsx**

Change background and border from `bg-background` to PDT dark:
```tsx
<aside
  className={cn(
    'flex flex-col border-r border-pdt-background/20 bg-pdt-primary transition-all duration-300',
    isCollapsed ? 'w-16' : 'w-64',
    className
  )}
>
```

Update the header section border and text:
```tsx
<div
  className={cn(
    'flex h-16 items-center border-b border-pdt-background/20 px-3',
    isCollapsed && 'justify-center px-2'
  )}
>
  <div className="flex items-center gap-2">
    <img
      src="/logo.svg"
      alt="Logo"
      className={cn('size-8', isCollapsed && 'size-7')}
    />
    {!isCollapsed && (
      <span className="text-base font-semibold text-pdt-neutral">Dashboard</span>
    )}
  </div>
  <Button
    variant="ghost"
    size="icon"
    onClick={toggle}
    className={cn('ml-auto size-8 text-pdt-background hover:bg-pdt-primary-light hover:text-pdt-background', isCollapsed && 'ml-0 hidden')}
  >
    {isCollapsed ? (
      <PanelLeft className="size-4" />
    ) : (
      <PanelLeftClose className="size-4" />
    )}
    <span className="sr-only">Toggle sidebar</span>
  </Button>
</div>
```

**Step 3: Update SidebarNavItem.tsx**

Replace the styling for nav items to use PDT colors:

Change the Link className from:
```tsx
'flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition-colors',
'hover:bg-accent hover:text-accent-foreground active:bg-accent/80',
isActive && 'bg-accent text-accent-foreground',
```
to:
```tsx
'flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition-colors',
'text-pdt-neutral/70 hover:bg-pdt-primary-light hover:text-pdt-neutral active:bg-pdt-primary-light/80',
isActive && 'border-l-2 border-pdt-background bg-pdt-primary-light text-pdt-background',
```

**Step 4: Update SidebarNavGroup.tsx**

Change the group title color from `text-muted-foreground` to `text-pdt-neutral/40`:
```tsx
<h4
  className={cn(
    'mb-1.5 px-3 text-[11px] font-semibold uppercase tracking-wider text-pdt-neutral/40'
  )}
>
```

**Step 5: Update Header.tsx**

Change the header background and border:
```tsx
<header
  className={cn(
    'flex h-16 items-center justify-between border-b border-pdt-background/20 bg-pdt-primary-light px-4 lg:px-6',
    className
  )}
>
```

Update the brand text:
```tsx
<span className="hidden text-base font-semibold text-pdt-neutral sm:inline-block md:text-lg">
  Dashboard
</span>
```

Update ghost button colors:
- Menu button: add `text-pdt-neutral hover:bg-pdt-primary hover:text-pdt-neutral`
- Bell button: add `text-pdt-neutral hover:bg-pdt-primary hover:text-pdt-neutral`
- Bell red dot: change `bg-destructive` to `bg-pdt-background`
- Avatar trigger: add `hover:bg-pdt-primary`
- AvatarFallback: add `bg-pdt-background text-pdt-primary`

Update DropdownMenuContent:
```tsx
<DropdownMenuContent align="end" className="w-56 border-pdt-background/20 bg-pdt-primary-light">
```

Update DropdownMenuLabel text:
```tsx
<p className="text-sm font-medium leading-none text-pdt-neutral">
  {user?.name || 'User'}
</p>
<p className="text-xs leading-none text-pdt-neutral/60">
  {user?.email || 'user@example.com'}
</p>
```

Update DropdownMenuItem:
```tsx
<DropdownMenuItem asChild className="text-pdt-neutral/70 hover:text-pdt-neutral focus:bg-pdt-primary focus:text-pdt-neutral">
```

**Step 6: Update MobileNav.tsx**

Change backdrop:
```tsx
'fixed inset-0 z-40 bg-pdt-primary/80 backdrop-blur-sm transition-opacity md:hidden',
```

Change drawer:
```tsx
'fixed inset-y-0 left-0 z-50 flex w-[280px] max-w-[85vw] flex-col border-r border-pdt-background/20 bg-pdt-primary transition-transform duration-300 md:hidden',
```

Update header:
```tsx
<div className="flex h-14 shrink-0 items-center justify-between border-b border-pdt-background/20 px-3">
  <div className="flex items-center gap-2">
    <img src="/logo.svg" alt="Logo" className="size-7" />
    <span className="text-base font-semibold text-pdt-neutral">Menu</span>
  </div>
  <Button variant="ghost" size="icon" onClick={onClose} className="text-pdt-neutral hover:bg-pdt-primary-light hover:text-pdt-neutral">
    <X className="size-5" />
    <span className="sr-only">Close menu</span>
  </Button>
</div>
```

**Step 7: Verify build**

Run: `cd /home/nst/GolandProjects/pdt/frontend && npx vite build 2>&1 | tail -5`
Expected: Build succeeds

**Step 8: Commit**

```bash
git add src/presentation/layouts/ src/presentation/components/dashboard/Sidebar/
git commit -m "style: restyle dashboard layout, sidebar, and header to PDT dark theme"
```

---

## Task 9: Update Routes — Remove Demo Dashboard, Add Register

**Files:**
- Modify: `src/presentation/routes/index.tsx`
- Modify: `src/config/navigation.ts`

**Step 1: Update routes**

Replace the full routes file. Key changes:
- Remove `DashboardPage` import
- Add `RegisterPage` import
- `/dashboard` redirects to `/dashboard/home`
- `/register` renders `RegisterPage` (not LoginPage)

```tsx
import { createBrowserRouter, Navigate } from 'react-router-dom'
import PublicLayout from '../layouts/PublicLayout'
import { DashboardLayout } from '../layouts/DashboardLayout'
import { LoginPage } from '../pages/auth/LoginPage'
import { NotFoundPage } from '../pages/NotFoundPage'
import { LandingPage } from '../pages/LandingPage'
import { RegisterPage } from '../pages/auth/RegisterPage'

import { DashboardHomePage } from '../pages/DashboardHomePage'
import { ReposPage } from '../pages/ReposPage'
import { CommitsPage } from '../pages/CommitsPage'
import { JiraPage } from '../pages/JiraPage'
import { ReportsPage } from '../pages/ReportsPage'
import { SettingsPage } from '../pages/SettingsPage'

export const router = createBrowserRouter([
  {
    path: '/',
    element: <PublicLayout />,
    errorElement: <NotFoundPage />,
    children: [
      { path: '/', element: <LandingPage /> },
      { path: 'login', element: <LoginPage /> },
      { path: 'register', element: <RegisterPage /> }
    ]
  },
  {
    path: '/',
    element: <DashboardLayout />,
    children: [
      { path: 'dashboard', element: <Navigate to="/dashboard/home" replace /> },
      { path: 'dashboard/home', element: <DashboardHomePage /> },
      { path: 'dashboard/repos', element: <ReposPage /> },
      { path: 'dashboard/commits', element: <CommitsPage /> },
      { path: 'dashboard/jira', element: <JiraPage /> },
      { path: 'dashboard/reports', element: <ReportsPage /> },
      { path: 'dashboard/settings', element: <SettingsPage /> }
    ]
  },
  { path: '*', element: <NotFoundPage /> }
])
```

**Step 2: Update navigation config**

In `src/config/navigation.ts`, change the Dashboard href from `/dashboard` to `/dashboard/home`:

```ts
{ title: 'Dashboard', href: '/dashboard/home', icon: LayoutDashboard },
```

**Step 3: Verify build** (will fail until RegisterPage exists — that's OK, note it)

Skip build verification for now. RegisterPage will be created in Task 10.

**Step 4: Commit** (defer to after Task 10)

---

## Task 10: Create Register Page

**Files:**
- Create: `src/presentation/pages/auth/RegisterPage.tsx`

**Step 1: Create the component**

```tsx
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useNavigate, Link } from 'react-router-dom'
import { Eye, EyeOff } from 'lucide-react'

import { useRegisterMutation } from '@/infrastructure/services/auth.service'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

interface RegisterFormData {
  name: string
  email: string
  password: string
}

export function RegisterPage() {
  const navigate = useNavigate()
  const [registerMutation, { isLoading, error }] = useRegisterMutation()
  const [showPassword, setShowPassword] = useState(false)

  const {
    register,
    handleSubmit,
    formState: { errors }
  } = useForm<RegisterFormData>()

  const onSubmit = async (data: RegisterFormData) => {
    try {
      await registerMutation(data).unwrap()
      navigate('/dashboard/home')
    } catch {
      // Error handled by RTK Query onQueryStarted
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-pdt-background px-4">
      <div className="w-full max-w-md">
        {/* Header */}
        <div className="mb-8 text-center">
          <Link to="/" className="inline-flex items-center gap-2">
            <svg width="40" height="40" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
              <circle cx="12" cy="12" r="10" stroke="#1B1B1E" strokeWidth="2"/>
              <path d="M7 14L10 11L13 13L17 8" stroke="#1B1B1E" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round"/>
              <circle cx="17" cy="8" r="1.5" fill="#1B1B1E"/>
            </svg>
            <span className="text-3xl font-bold text-pdt-primary">PDT</span>
          </Link>
          <p className="mt-2 text-pdt-primary/70">Personal Development Tracker</p>
        </div>

        {/* Form Card */}
        <div className="rounded-xl border-2 border-pdt-primary/10 bg-pdt-primary p-6 shadow-xl sm:p-8">
          <h2 className="mb-1 text-center text-2xl font-bold text-pdt-neutral">
            Create your account
          </h2>
          <p className="mb-6 text-center text-sm text-pdt-neutral/60">
            Start tracking your development progress
          </p>

          {error && (
            <div className="mb-4 rounded-lg bg-red-500/10 p-3 text-sm text-red-400">
              Registration failed. Please try again.
            </div>
          )}

          <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="name" className="text-pdt-neutral/70">
                Name
              </Label>
              <Input
                {...register('name', { required: 'Name is required' })}
                id="name"
                type="text"
                placeholder="John Doe"
                autoComplete="name"
                className="border-pdt-neutral/20 bg-pdt-primary-light text-pdt-neutral placeholder:text-pdt-neutral/30 focus:border-pdt-background focus:ring-pdt-background"
              />
              {errors.name && (
                <p className="text-sm text-red-400">{errors.name.message}</p>
              )}
            </div>

            <div className="space-y-2">
              <Label htmlFor="email" className="text-pdt-neutral/70">
                Email
              </Label>
              <Input
                {...register('email', {
                  required: 'Email is required',
                  pattern: {
                    value: /^[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}$/i,
                    message: 'Invalid email address'
                  }
                })}
                id="email"
                type="email"
                placeholder="you@example.com"
                autoComplete="email"
                className="border-pdt-neutral/20 bg-pdt-primary-light text-pdt-neutral placeholder:text-pdt-neutral/30 focus:border-pdt-background focus:ring-pdt-background"
              />
              {errors.email && (
                <p className="text-sm text-red-400">{errors.email.message}</p>
              )}
            </div>

            <div className="space-y-2">
              <Label htmlFor="password" className="text-pdt-neutral/70">
                Password
              </Label>
              <div className="relative">
                <Input
                  {...register('password', {
                    required: 'Password is required',
                    minLength: {
                      value: 8,
                      message: 'Password must be at least 8 characters'
                    }
                  })}
                  id="password"
                  type={showPassword ? 'text' : 'password'}
                  placeholder="Min. 8 characters"
                  autoComplete="new-password"
                  className="border-pdt-neutral/20 bg-pdt-primary-light pr-10 text-pdt-neutral placeholder:text-pdt-neutral/30 focus:border-pdt-background focus:ring-pdt-background"
                />
                <button
                  type="button"
                  onClick={() => setShowPassword(!showPassword)}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-pdt-neutral/40 hover:text-pdt-neutral/70"
                >
                  {showPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                </button>
              </div>
              {errors.password && (
                <p className="text-sm text-red-400">{errors.password.message}</p>
              )}
            </div>

            <Button
              type="submit"
              disabled={isLoading}
              className="w-full bg-pdt-background text-pdt-primary font-semibold hover:bg-pdt-background/90"
            >
              {isLoading ? (
                <>
                  <svg
                    className="mr-2 h-4 w-4 animate-spin"
                    xmlns="http://www.w3.org/2000/svg"
                    fill="none"
                    viewBox="0 0 24 24"
                  >
                    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                    <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
                  </svg>
                  Creating account...
                </>
              ) : (
                'Create Account'
              )}
            </Button>
          </form>

          <p className="mt-6 text-center text-sm text-pdt-neutral/60">
            Already have an account?{' '}
            <Link
              to="/login"
              className="font-medium text-pdt-background underline-offset-4 hover:underline"
            >
              Sign in
            </Link>
          </p>
        </div>
      </div>
    </div>
  )
}
```

**Step 2: Verify build**

Run: `cd /home/nst/GolandProjects/pdt/frontend && npx vite build 2>&1 | tail -5`
Expected: Build succeeds

**Step 3: Commit (includes Task 9 route changes)**

```bash
git add src/presentation/pages/auth/RegisterPage.tsx src/presentation/routes/index.tsx src/config/navigation.ts
git commit -m "feat: add register page and update routes (redirect /dashboard to /dashboard/home)"
```

---

## Task 11: Restyle LoginPage to Match PDT Theme

**Files:**
- Modify: `src/presentation/pages/auth/LoginPage.tsx`

**Step 1: Restyle LoginPage**

Rewrite LoginPage to match the RegisterPage style — yellow bg, dark card, PDT colors:

```tsx
import { useForm } from 'react-hook-form'
import { useNavigate, Link } from 'react-router-dom'
import { ILoginCredentials } from '@/domain/auth/interfaces/auth.interface'
import { useLoginMutation } from '@/infrastructure/services/auth.service'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

export const LoginPage = () => {
  const navigate = useNavigate()
  const [loginMutation, { isLoading, error }] = useLoginMutation()
  const {
    register,
    handleSubmit,
    formState: { errors }
  } = useForm<ILoginCredentials>()

  const onSubmit = async (data: ILoginCredentials) => {
    try {
      await loginMutation(data).unwrap()
      navigate('/dashboard/home')
    } catch {
      // Error handled by RTK Query onQueryStarted
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-pdt-background px-4">
      <div className="w-full max-w-md">
        {/* Header */}
        <div className="mb-8 text-center">
          <Link to="/" className="inline-flex items-center gap-2">
            <svg width="40" height="40" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
              <circle cx="12" cy="12" r="10" stroke="#1B1B1E" strokeWidth="2"/>
              <path d="M7 14L10 11L13 13L17 8" stroke="#1B1B1E" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round"/>
              <circle cx="17" cy="8" r="1.5" fill="#1B1B1E"/>
            </svg>
            <span className="text-3xl font-bold text-pdt-primary">PDT</span>
          </Link>
          <p className="mt-2 text-pdt-primary/70">Personal Development Tracker</p>
        </div>

        {/* Form Card */}
        <div className="rounded-xl border-2 border-pdt-primary/10 bg-pdt-primary p-6 shadow-xl sm:p-8">
          <h2 className="mb-1 text-center text-2xl font-bold text-pdt-neutral">
            Welcome Back
          </h2>
          <p className="mb-6 text-center text-sm text-pdt-neutral/60">
            Enter your credentials to access your account
          </p>

          {error && (
            <div className="mb-4 rounded-lg bg-red-500/10 p-3 text-sm text-red-400">
              Login failed. Please check your credentials.
            </div>
          )}

          <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="email" className="text-pdt-neutral/70">
                Email
              </Label>
              <Input
                {...register('email', {
                  required: 'Email is required',
                  pattern: {
                    value: /^[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}$/i,
                    message: 'Invalid email address'
                  }
                })}
                id="email"
                type="email"
                placeholder="you@example.com"
                autoComplete="email"
                className="border-pdt-neutral/20 bg-pdt-primary-light text-pdt-neutral placeholder:text-pdt-neutral/30 focus:border-pdt-background focus:ring-pdt-background"
              />
              {errors.email && (
                <p className="text-sm text-red-400">{errors.email.message}</p>
              )}
            </div>

            <div className="space-y-2">
              <Label htmlFor="password" className="text-pdt-neutral/70">
                Password
              </Label>
              <Input
                {...register('password', {
                  required: 'Password is required'
                })}
                id="password"
                type="password"
                autoComplete="current-password"
                className="border-pdt-neutral/20 bg-pdt-primary-light text-pdt-neutral placeholder:text-pdt-neutral/30 focus:border-pdt-background focus:ring-pdt-background"
              />
              {errors.password && (
                <p className="text-sm text-red-400">{errors.password.message}</p>
              )}
            </div>

            <Button
              type="submit"
              disabled={isLoading}
              className="w-full bg-pdt-background text-pdt-primary font-semibold hover:bg-pdt-background/90"
            >
              {isLoading ? (
                <>
                  <svg
                    className="mr-2 h-4 w-4 animate-spin"
                    xmlns="http://www.w3.org/2000/svg"
                    fill="none"
                    viewBox="0 0 24 24"
                  >
                    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                    <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
                  </svg>
                  Signing in...
                </>
              ) : (
                'Sign in'
              )}
            </Button>
          </form>

          <p className="mt-6 text-center text-sm text-pdt-neutral/60">
            Don&apos;t have an account?{' '}
            <Link
              to="/register"
              className="font-medium text-pdt-background underline-offset-4 hover:underline"
            >
              Sign up
            </Link>
          </p>
        </div>
      </div>
    </div>
  )
}
```

**Step 2: Verify build**

Run: `cd /home/nst/GolandProjects/pdt/frontend && npx vite build 2>&1 | tail -5`
Expected: Build succeeds

**Step 3: Commit**

```bash
git add src/presentation/pages/auth/LoginPage.tsx
git commit -m "style: restyle LoginPage to match PDT dark theme"
```

---

## Task 12: Restyle DashboardHomePage

**Files:**
- Modify: `src/presentation/pages/DashboardHomePage.tsx`

**Step 1: Restyle using shared components**

Replace the entire file content. Key changes:
- Use `PageHeader` for the welcome section
- Use `DataCard` for the recent commits section
- Use `StatusBadge` for Jira card keys
- Use `pdt` button variant for Sync Now
- Remove all hardcoded hex colors

```tsx
import { GitCommit, Link2, Trello, RefreshCw } from 'lucide-react'

import { useListCommitsQuery } from '@/infrastructure/services/commit.service'
import { useGetActiveSprintQuery } from '@/infrastructure/services/jira.service'
import { useGetSyncStatusQuery, useTriggerSyncMutation } from '@/infrastructure/services/sync.service'
import { useGetProfileQuery } from '@/infrastructure/services/user.service'
import { Button } from '@/components/ui/button'
import { StatsCard, StatsCardSkeleton } from '@/presentation/components/dashboard'
import { PageHeader, DataCard, StatusBadge } from '@/presentation/components/common'

export function DashboardHomePage() {
  const { data: profile, isLoading: profileLoading } = useGetProfileQuery()
  const { data: commitsData, isLoading: commitsLoading } = useListCommitsQuery()
  const { data: activeSprint, isLoading: sprintLoading } = useGetActiveSprintQuery()
  const { data: syncStatus } = useGetSyncStatusQuery()
  const [triggerSync, { isLoading: isSyncing }] = useTriggerSyncMutation()

  const totalCommits = commitsData?.total || 0
  const commits = commitsData?.commits || []
  const linkedCommits = commits.filter((c) => (c as any).hasLink || (c as any).jiraCardKey).length
  const linkedPercent = totalCommits > 0 ? Math.round((linkedCommits / totalCommits) * 100) : 0
  const activeSprintCards = activeSprint?.cards?.length || 0

  const isLoading = profileLoading || commitsLoading || sprintLoading

  const stats = [
    {
      title: 'Total Commits (30d)',
      value: totalCommits,
      description: syncStatus?.lastSyncAt
        ? `Last sync: ${new Date(syncStatus.lastSyncAt).toLocaleString()}`
        : 'No sync yet',
      icon: GitCommit
    },
    {
      title: 'Linked to Jira',
      value: linkedCommits,
      description: `${linkedPercent}% linked`,
      icon: Link2
    },
    {
      title: 'Active Sprint',
      value: activeSprintCards,
      description: activeSprint?.name || 'No active sprint',
      icon: Trello
    }
  ]

  return (
    <div className="min-w-0 space-y-4 md:space-y-6">
      <PageHeader
        title={`Welcome back${profile?.email ? '' : ''}`}
        description={profile?.email || 'Loading...'}
        action={
          <Button
            onClick={() => triggerSync()}
            disabled={isSyncing}
            variant="pdt"
          >
            <RefreshCw className={`mr-2 h-4 w-4 ${isSyncing ? 'animate-spin' : ''}`} />
            {isSyncing ? 'Syncing...' : 'Sync Now'}
          </Button>
        }
      />

      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
        {isLoading
          ? Array.from({ length: 3 }).map((_, i) => <StatsCardSkeleton key={i} />)
          : stats.map((stat) => (
              <StatsCard
                key={stat.title}
                title={stat.title}
                value={stat.value}
                description={stat.description}
                icon={stat.icon}
              />
            ))}
      </div>

      <DataCard title="Recent Commits">
        {commitsLoading ? (
          <p className="text-pdt-neutral/60">Loading...</p>
        ) : commits.length === 0 ? (
          <p className="text-pdt-neutral/60">No commits yet. Add a repository to get started.</p>
        ) : (
          <div className="space-y-0">
            {commits.slice(0, 5).map((commit) => (
              <div
                key={commit.id}
                className="flex items-center justify-between border-b border-pdt-neutral/10 py-3 last:border-0"
              >
                <div className="min-w-0 flex-1">
                  <p className="truncate text-pdt-neutral">{commit.message}</p>
                  <p className="text-sm text-pdt-neutral/50">
                    {commit.sha.slice(0, 7)} &middot;{' '}
                    {new Date(commit.date).toLocaleDateString()}
                  </p>
                </div>
                {(commit as any).jiraCardKey && (
                  <StatusBadge variant="warning" className="ml-2">
                    {(commit as any).jiraCardKey}
                  </StatusBadge>
                )}
              </div>
            ))}
          </div>
        )}
      </DataCard>
    </div>
  )
}

export default DashboardHomePage
```

**Step 2: Verify build**

Run: `cd /home/nst/GolandProjects/pdt/frontend && npx vite build 2>&1 | tail -5`
Expected: Build succeeds

**Step 3: Commit**

```bash
git add src/presentation/pages/DashboardHomePage.tsx
git commit -m "style: restyle DashboardHomePage with shared components"
```

---

## Task 13: Restyle ReposPage

**Files:**
- Modify: `src/presentation/pages/ReposPage.tsx`

**Step 1: Restyle using shared components**

Key changes:
- Use `PageHeader` with add repo action
- Use `DataCard` for the add form
- Use `StatusBadge` for valid/invalid
- Use `EmptyState` for empty list
- Use `pdt` button variant
- Use PDT token classes instead of hardcoded hex

Replace the entire file. Use `PageHeader`, `DataCard`, `StatusBadge`, `EmptyState` from `@/presentation/components/common`. Keep all existing API logic and state unchanged. Replace all `bg-[#1B1B1E]` with `bg-pdt-primary-light`, `text-[#FBFFFE]` with `text-pdt-neutral`, `text-[#FBFFFE]/60` with `text-pdt-neutral/60`, `border-[#F8C630]/20` with `border-pdt-background/20`, `text-[#F8C630]` with `text-pdt-background`. Use `variant="pdt"` for the "Add Repository" button. Use `StatusBadge variant="success"` for Valid and `variant="danger"` for Invalid.

**Step 2: Verify build**

Run: `cd /home/nst/GolandProjects/pdt/frontend && npx vite build 2>&1 | tail -5`
Expected: Build succeeds

**Step 3: Commit**

```bash
git add src/presentation/pages/ReposPage.tsx
git commit -m "style: restyle ReposPage with shared components"
```

---

## Task 14: Restyle CommitsPage

**Files:**
- Modify: `src/presentation/pages/CommitsPage.tsx`

**Step 1: Restyle using shared components**

Key changes:
- Use `PageHeader` for title
- Use `FilterBar` wrapper for the filters row
- Use `DataCard` for the commits table
- Use `StatusBadge` for Jira badges
- Use PDT token classes instead of hardcoded hex
- Style the `<select>` with PDT colors
- Use `variant="pdt"` / `variant="pdtOutline"` for buttons

Keep all existing filter state, API calls, and link-to-Jira logic unchanged.

**Step 2: Verify build**

Run: `cd /home/nst/GolandProjects/pdt/frontend && npx vite build 2>&1 | tail -5`
Expected: Build succeeds

**Step 3: Commit**

```bash
git add src/presentation/pages/CommitsPage.tsx
git commit -m "style: restyle CommitsPage with shared components"
```

---

## Task 15: Restyle JiraPage

**Files:**
- Modify: `src/presentation/pages/JiraPage.tsx`

**Step 1: Restyle using shared components**

Key changes:
- Use `PageHeader` for title
- Use `DataCard` for active sprint and all sprints sections
- Use `StatusBadge` for sprint states and card statuses (Done=success, In Progress=info, other=neutral, active=success, closed=neutral, future=info)
- Use `EmptyState` for no active sprint / no sprints
- Replace Card/CardHeader/CardContent with DataCard or plain divs with PDT classes
- Use PDT token classes instead of hardcoded hex

Keep all existing API calls unchanged.

**Step 2: Verify build**

Run: `cd /home/nst/GolandProjects/pdt/frontend && npx vite build 2>&1 | tail -5`
Expected: Build succeeds

**Step 3: Commit**

```bash
git add src/presentation/pages/JiraPage.tsx
git commit -m "style: restyle JiraPage with shared components"
```

---

## Task 16: Restyle ReportsPage

**Files:**
- Modify: `src/presentation/pages/ReportsPage.tsx`

**Step 1: Restyle using shared components**

Key changes:
- Use `PageHeader` for title
- Use `DataCard` for generate report form and reports list
- Use `EmptyState` for no reports
- Use `pdt` button variant for Generate
- Use PDT token classes instead of hardcoded hex
- Style the date input with PDT colors

Keep all existing API calls and state unchanged.

**Step 2: Verify build**

Run: `cd /home/nst/GolandProjects/pdt/frontend && npx vite build 2>&1 | tail -5`
Expected: Build succeeds

**Step 3: Commit**

```bash
git add src/presentation/pages/ReportsPage.tsx
git commit -m "style: restyle ReportsPage with shared components"
```

---

## Task 17: Restyle SettingsPage

**Files:**
- Modify: `src/presentation/pages/SettingsPage.tsx`

**Step 1: Restyle using shared components**

Key changes:
- Use `PageHeader` for title
- Use `DataCard` for Profile section and Integrations form
- Use `StatusBadge` for configured (variant="success") / not configured (variant="danger")
- Use `pdt` button variant for Save, `pdtOutline` for Validate
- Use PDT token classes for all inputs
- Replace inline CheckCircle/XCircle with StatusBadge

Keep all existing form state and API calls unchanged.

**Step 2: Verify build**

Run: `cd /home/nst/GolandProjects/pdt/frontend && npx vite build 2>&1 | tail -5`
Expected: Build succeeds

**Step 3: Commit**

```bash
git add src/presentation/pages/SettingsPage.tsx
git commit -m "style: restyle SettingsPage with shared components"
```

---

## Task 18: Restyle NotFoundPage

**Files:**
- Modify: `src/presentation/pages/NotFoundPage.tsx`

**Step 1: Restyle with PDT theme**

```tsx
import { Link } from 'react-router-dom'
import { Button } from '@/components/ui/button'

export const NotFoundPage = () => {
  return (
    <div className="flex min-h-screen items-center justify-center bg-pdt-primary px-4">
      <div className="text-center">
        <h1 className="mb-2 text-6xl font-bold text-pdt-background">404</h1>
        <p className="mb-6 text-lg text-pdt-neutral/60">Page not found</p>
        <Button asChild variant="pdt">
          <Link to="/">Go back home</Link>
        </Button>
      </div>
    </div>
  )
}
```

**Step 2: Verify build**

Run: `cd /home/nst/GolandProjects/pdt/frontend && npx vite build 2>&1 | tail -5`
Expected: Build succeeds

**Step 3: Commit**

```bash
git add src/presentation/pages/NotFoundPage.tsx
git commit -m "style: restyle NotFoundPage with PDT theme"
```

---

## Task 19: Final Build Verification

**Step 1: Run full build**

Run: `cd /home/nst/GolandProjects/pdt/frontend && npx vite build 2>&1`
Expected: Build succeeds with no errors

**Step 2: Check for any remaining hardcoded hex colors in modified files**

Run: `grep -rn '#1B1B1E\|#F8C630\|#FBFFFE\|#96031A' src/presentation/pages/ src/presentation/layouts/ src/presentation/components/common/`

Expected: No matches in dashboard pages (landing page still uses hex, that's OK).

**Step 3: Final commit if any cleanup needed**

```bash
git add -A
git commit -m "style: final cleanup for PDT dashboard restyle"
```
