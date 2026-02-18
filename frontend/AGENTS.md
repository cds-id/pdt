# AGENTS.md - AI Agent Guidelines

This document provides guidelines for AI agents working on this codebase.

## Project Architecture Overview

```
src/
├── application/          # Application layer - Redux store setup
│   └── store/           # Store configuration and middleware
├── components/          # Shared UI components
│   └── ui/              # shadcn/ui primitives
├── config/              # Application configuration
├── domain/              # Domain layer - Business entities & interfaces
│   └── auth/            # Auth domain models
├── infrastructure/      # Infrastructure layer - External services
│   ├── api/             # API client configuration
│   ├── services/        # Service implementations
│   └── slices/          # Redux slices with RTK Query
├── lib/                 # Utility functions
└── presentation/        # Presentation layer - UI components
    ├── components/      # Feature-specific components
    ├── layouts/         # Layout components
    ├── pages/           # Page components (route endpoints)
    └── routes/          # Route definitions
```

### Layer Responsibilities

| Layer | Responsibility | Example |
|-------|---------------|---------|
| Domain | Business entities, interfaces | `auth.interface.ts` |
| Application | Store setup, middleware | `store/index.ts` |
| Infrastructure | API calls, external services | `auth.service.ts`, `auth.slice.ts` |
| Presentation | UI components, pages, layouts | `LoginPage.tsx`, `DashboardLayout.tsx` |

---

## Naming Conventions

### Files
- **Components**: `PascalCase.tsx` (e.g., `StatsCard.tsx`)
- **Hooks**: `camelCase.ts` with `use` prefix (e.g., `useAuth.ts`)
- **Types**: `kebab-case.types.ts` (e.g., `stats-card.types.ts`)
- **Utils**: `camelCase.ts` (e.g., `formatDate.ts`)
- **Constants**: `camelCase.ts` or `SCREAMING_SNAKE_CASE` for values

### Components
- Component names: `PascalCase` (e.g., `StatsCard`)
- Props interfaces: `ComponentNameProps` suffix (e.g., `StatsCardProps`)
- Context: `ComponentNameContext` (e.g., `SidebarContext`)

### Variables & Functions
- Functions/variables: `camelCase`
- Constants: `SCREAMING_SNAKE_CASE` (e.g., `MAX_RETRIES`)
- Boolean props: `is`, `has`, `should` prefix (e.g., `isLoading`, `hasError`)

---

## File Organization Rules

### Component Folder Structure
```
ComponentName/
├── ComponentName.tsx       # Main component
├── SubComponent.tsx        # Sub-components (if needed)
├── component-name.types.ts # TypeScript interfaces
├── useComponentName.ts     # Custom hooks (if needed)
└── index.ts                # Barrel export
```

### Barrel Exports
Every component folder MUST have an `index.ts`:
```typescript
export { ComponentName } from './ComponentName'
export type { ComponentNameProps } from './component-name.types'
```

### Import Order
1. React imports
2. Third-party libraries
3. Internal absolute imports (`@/`)
4. Relative imports
5. Type imports
6. CSS/style imports

```typescript
import * as React from 'react'

import { motion } from 'framer-motion'

import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'

import { SidebarNav } from './SidebarNav'

import type { SidebarProps } from './sidebar.types'
```

---

## Component Patterns

### When to Use shadcn/ui vs Custom Components
- **shadcn/ui**: Base primitives (Button, Input, Card, etc.)
- **Custom**: Feature-specific components that compose primitives

### shadcn/ui Component Template
```typescript
import * as React from 'react'
import { cva, type VariantProps } from 'class-variance-authority'

import { cn } from '@/lib/utils'

const componentVariants = cva(
  'base-classes',
  {
    variants: {
      variant: {
        default: 'variant-classes',
      },
      size: {
        default: 'size-classes',
      },
    },
    defaultVariants: {
      variant: 'default',
      size: 'default',
    },
  }
)

export interface ComponentProps
  extends React.HTMLAttributes<HTMLDivElement>,
    VariantProps<typeof componentVariants> {}

const Component = React.forwardRef<HTMLDivElement, ComponentProps>(
  ({ className, variant, size, ...props }, ref) => (
    <div
      ref={ref}
      className={cn(componentVariants({ variant, size, className }))}
      {...props}
    />
  )
)
Component.displayName = 'Component'

export { Component, componentVariants }
```

### Feature Component Template
```typescript
import * as React from 'react'

import { cn } from '@/lib/utils'
import { Card } from '@/components/ui/card'

import type { FeatureComponentProps } from './feature-component.types'

export function FeatureComponent({
  className,
  children,
  ...props
}: FeatureComponentProps) {
  return (
    <Card className={cn('feature-styles', className)} {...props}>
      {children}
    </Card>
  )
}
```

### forwardRef Usage
Use `forwardRef` for:
- Components that wrap DOM elements
- Components that need ref access for focus management
- shadcn/ui primitives

---

## State Management Guidelines

### RTK Query (Server State)
Use for ALL API calls:
```typescript
// In infrastructure/slices/
export const apiSlice = createApi({
  reducerPath: 'api',
  baseQuery: fetchBaseQuery({ baseUrl: '/api' }),
  endpoints: (builder) => ({
    getUsers: builder.query<User[], void>({
      query: () => 'users',
    }),
  }),
})

export const { useGetUsersQuery } = apiSlice
```

### Redux Slices (Client State)
Use for:
- UI state (sidebar collapsed, theme)
- Auth state (current user)
- Form drafts

```typescript
// In infrastructure/slices/
const uiSlice = createSlice({
  name: 'ui',
  initialState: { sidebarCollapsed: false },
  reducers: {
    toggleSidebar: (state) => {
      state.sidebarCollapsed = !state.sidebarCollapsed
    },
  },
})
```

### Local State (useState)
Use for:
- Component-specific UI state
- Form inputs before submission
- Temporary values

---

## DO's

1. **Use `cn()` for className merging** - Always use the utility from `@/lib/utils`
2. **Use `forwardRef` for DOM wrappers** - Enables ref forwarding
3. **Use absolute imports (`@/`)** - Configured in tsconfig
4. **Create barrel exports** - Every folder needs `index.ts`
5. **Define interfaces in `.types.ts` files** - Separate types from implementation
6. **Use RTK Query for API calls** - Handles caching, loading, errors
7. **Use Lucide icons** - Import from `lucide-react`
8. **Use CVA for variants** - `class-variance-authority` for component variants
9. **Follow responsive-first design** - Mobile-first with Tailwind breakpoints
10. **Add `displayName` to forwardRef components** - For debugging

---

## DON'Ts

1. **Don't modify shadcn patterns** - Keep primitives standard
2. **Don't use inline styles** - Use Tailwind classes
3. **Don't use `any` type** - Use proper TypeScript types
4. **Don't use default exports** - Except for pages and slices
5. **Don't import from `react-redux` directly** - Use typed hooks from store
6. **Don't mix business logic in presentation** - Keep layers separate
7. **Don't create 200+ line components** - Split into smaller components
8. **Don't use `index` as React key** - Use unique identifiers
9. **Don't hardcode colors** - Use Tailwind CSS variables
10. **Don't skip loading states** - Always handle loading UI

---

## Common Patterns

### Loading States
```typescript
function Component() {
  const { data, isLoading, error } = useGetDataQuery()

  if (isLoading) return <Skeleton />
  if (error) return <ErrorState error={error} />
  if (!data) return <EmptyState />

  return <DataDisplay data={data} />
}
```

### Error Handling
```typescript
import { ErrorBoundary } from '@/presentation/components/common/ErrorBoundary'

function App() {
  return (
    <ErrorBoundary>
      <MainContent />
    </ErrorBoundary>
  )
}
```

### Form Pattern (React Hook Form)
```typescript
import { useForm } from 'react-hook-form'
import { yupResolver } from '@hookform/resolvers/yup'
import * as yup from 'yup'

const schema = yup.object({
  email: yup.string().email().required(),
})

function FormComponent() {
  const { register, handleSubmit, formState: { errors } } = useForm({
    resolver: yupResolver(schema),
  })

  const onSubmit = (data) => {
    // Handle submission
  }

  return (
    <form onSubmit={handleSubmit(onSubmit)}>
      <Input {...register('email')} />
      {errors.email && <span>{errors.email.message}</span>}
      <Button type="submit">Submit</Button>
    </form>
  )
}
```

### Responsive Sidebar Pattern
```typescript
const { isCollapsed, isMobile, toggle } = useSidebar()

return (
  <>
    {isMobile ? (
      <Sheet open={isOpen} onOpenChange={setIsOpen}>
        <SheetContent side="left">
          <SidebarNav />
        </SheetContent>
      </Sheet>
    ) : (
      <aside className={cn('w-64', isCollapsed && 'w-16')}>
        <SidebarNav />
      </aside>
    )}
  </>
)
```

---

## Testing Guidelines

- Unit tests: `*.test.ts` or `*.test.tsx`
- Use Vitest with React Testing Library
- Test user interactions, not implementation details
- Mock API calls with MSW or RTK Query utilities

```typescript
import { render, screen } from '@testing-library/react'
import { StatsCard } from './StatsCard'

describe('StatsCard', () => {
  it('renders title and value', () => {
    render(<StatsCard title="Users" value={100} />)
    expect(screen.getByText('Users')).toBeInTheDocument()
    expect(screen.getByText('100')).toBeInTheDocument()
  })
})
```

---

## File Templates Reference

When creating new files, use these as starting points:

- **Page**: See `src/presentation/pages/dashboard/DashboardPage.tsx`
- **Layout**: See `src/presentation/layouts/ProtectedLayout.tsx`
- **shadcn Component**: See `src/components/ui/button.tsx`
- **Feature Component**: See `src/presentation/components/dashboard/`
- **Redux Slice**: See `src/infrastructure/slices/auth/auth.slice.ts`
