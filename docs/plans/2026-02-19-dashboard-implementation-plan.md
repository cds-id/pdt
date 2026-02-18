# PDT Dashboard Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task.

**Goal:** Create dashboard UI pages that use the already-mapped API services with PDT branding

**Architecture:** Sidebar navigation with PDT branding, main content area with dark theme, responsive design, uses existing RTK Query services

**Tech Stack:** React, TypeScript, Tailwind CSS, shadcn/ui, RTK Query

---

## Task 1: Create Dashboard Layout with Sidebar and TopBar

**Files:**
- Create: `frontend/src/presentation/layouts/DashboardLayout.tsx`
- Create: `frontend/src/components/layout/Sidebar.tsx`
- Create: `frontend/src/components/layout/TopBar.tsx`
- Modify: `frontend/src/presentation/router/AppRouter.tsx` - add dashboard routes

**Step 1: Create DashboardLayout component**

```tsx
import { Outlet } from 'react-router-dom'
import { Sidebar } from '@/components/layout/Sidebar'
import { TopBar } from '@/components/layout/TopBar'

export function DashboardLayout() {
  return (
    <div className="flex h-screen bg-[#1B1B1E]">
      <Sidebar />
      <div className="flex-1 flex flex-col overflow-hidden">
        <TopBar />
        <main className="flex-1 overflow-y-auto p-4 md:p-6">
          <Outlet />
        </main>
      </div>
    </div>
  )
}
```

**Step 2: Create Sidebar component**

```tsx
import { NavLink } from 'react-router-dom'
import { cn } from '@/lib/utils'

const navItems = [
  { to: '/dashboard', label: 'Dashboard', icon: 'M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6' },
  { to: '/dashboard/repos', label: 'Repositories', icon: 'M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z' },
  { to: '/dashboard/commits', label: 'Commits', icon: 'M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-3 7h3m-3 4h3m-6-4h.01M9 16h.01' },
  { to: '/dashboard/jira', label: 'Jira', icon: 'M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2' },
  { to: '/dashboard/reports', label: 'Reports', icon: 'M9 17v-2m3 2v-4m3 4v-6m2 10H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z' },
  { to: '/dashboard/settings', label: 'Settings', icon: 'M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z' },
]

export function Sidebar() {
  return (
    <aside className="w-64 bg-[#F8C630] flex flex-col">
      {/* Logo */}
      <div className="h-16 flex items-center px-6 border-b border-[#1B1B1E]/10">
        <svg width="32" height="32" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
          <circle cx="12" cy="12" r="10" stroke="#1B1B1E" strokeWidth="2"/>
          <path d="M7 14L10 11L13 13L17 8" stroke="#1B1B1E" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round"/>
          <circle cx="17" cy="8" r="1.5" fill="#1B1B1E"/>
        </svg>
        <span className="ml-3 text-xl font-bold text-[#1B1B1E]">PDT</span>
      </div>

      {/* Navigation */}
      <nav className="flex-1 py-6 px-3">
        <ul className="space-y-1">
          {navItems.map((item) => (
            <li key={item.to}>
              <NavLink
                to={item.to}
                className={({ isActive }) =>
                  cn(
                    'flex items-center gap-3 px-4 py-3 rounded-lg text-sm font-medium transition-colors',
                    isActive
                      ? 'bg-[#1B1B1E] text-[#F8C630]'
                      : 'text-[#1B1B1E] hover:bg-[#1B1B1E]/10'
                  )
                }
              >
                <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d={item.icon} />
                </svg>
                {item.label}
              </NavLink>
            </li>
          ))}
        </ul>
      </nav>

      {/* Logout */}
      <div className="p-3 border-t border-[#1B1B1E]/10">
        <button className="flex items-center gap-3 w-full px-4 py-3 rounded-lg text-sm font-medium text-[#1B1B1E] hover:bg-[#1B1B1E]/10 transition-colors">
          <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1" />
          </svg>
          Logout
        </button>
      </div>
    </aside>
  )
}
```

**Step 3: Create TopBar component**

```tsx
import { useGetProfileQuery, useLogoutMutation } from '@/infrastructure/services/user.service'
import { useNavigate } from 'react-router-dom'

export function TopBar() {
  const { data: profile } = useGetProfileQuery()
  const [logout] = useLogoutMutation()
  const navigate = useNavigate()

  const handleLogout = async () => {
    await logout()
    navigate('/login')
  }

  return (
    <header className="h-16 bg-[#1B1B1E] border-b border-[#F8C630]/20 flex items-center justify-between px-4 md:px-6">
      <div className="text-[#FBFFFE] text-lg font-semibold">
        {/* Page title would be set by each page */}
      </div>

      <div className="flex items-center gap-4">
        <div className="flex items-center gap-3">
          <div className="w-8 h-8 bg-[#F8C630] rounded-full flex items-center justify-center">
            <span className="text-[#1B1B1E] font-semibold text-sm">
              {profile?.email?.[0]?.toUpperCase() || 'U'}
            </span>
          </div>
          <span className="text-[#FBFFFE] text-sm hidden md:block">{profile?.email}</span>
        </div>

        <button
          onClick={handleLogout}
          className="text-[#FBFFFE]/70 hover:text-[#F8C630] transition-colors"
        >
          <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1" />
          </svg>
        </button>
      </div>
    </header>
  )
}
```

**Step 4: Add dashboard routes**

Modify `AppRouter.tsx` to add dashboard routes with DashboardLayout

**Step 5: Commit**

```bash
cd frontend && git add src/presentation/layouts/DashboardLayout.tsx src/components/layout/ && git commit -m "feat: add dashboard layout with sidebar and topbar"
```

---

## Task 2: Create PDT-Styled Reusable Button Component

**Files:**
- Modify: `frontend/src/components/ui/button.tsx` - extend with PDT variants

**Step 1: Add PDT button variants**

```tsx
// Add these variants to the buttonVariants function:
pdt: "bg-[#F8C630] text-[#1B1B1E] hover:bg-[#F8C630]/90 font-semibold",
pdtOutline: "border-2 border-[#F8C630] text-[#F8C630] hover:bg-[#F8C630] hover:text-[#1B1B1E]",
pdtGhost: "text-[#F8C630] hover:bg-[#F8C630]/10",
```

**Step 2: Commit**

```bash
cd frontend && git add src/components/ui/button.tsx && git commit -m "feat: add PDT-styled button variants"
```

---

## Task 3: Create Dashboard Home Page

**Files:**
- Create: `frontend/src/presentation/pages/DashboardHomePage.tsx`

**Step 1: Create DashboardHomePage component**

```tsx
import { useListCommitsQuery } from '@/infrastructure/services/commit.service'
import { useGetActiveSprintQuery } from '@/infrastructure/services/jira.service'
import { useGetSyncStatusQuery, useTriggerSyncMutation } from '@/infrastructure/services/sync.service'
import { useGetProfileQuery } from '@/infrastructure/services/user.service'
import { Button } from '@/components/ui/button'

export function DashboardHomePage() {
  const { data: profile } = useGetProfileQuery()
  const { data: commits } = useListCommitsQuery()
  const { data: activeSprint } = useGetActiveSprintQuery()
  const { data: syncStatus } = useGetSyncStatusQuery()
  const [triggerSync, { isLoading: isSyncing }] = useTriggerSyncMutation()

  const totalCommits = commits?.total || 0
  const linkedCommits = commits?.commits.filter(c => c.hasLink).length || 0

  return (
    <div className="space-y-6">
      {/* Welcome */}
      <div>
        <h1 className="text-2xl md:text-3xl font-bold text-[#FBFFFE]">
          Welcome back
        </h1>
        <p className="text-[#FBFFFE]/60">{profile?.email}</p>
      </div>

      {/* Stats Grid */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <StatsCard
          title="Total Commits (30d)"
          value={totalCommits}
          icon={
            <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
            </svg>
          }
        />
        <StatsCard
          title="Linked to Jira"
          value={linkedCommits}
          subtitle={`${totalCommits > 0 ? Math.round((linkedCommits / totalCommits) * 100) : 0}%`}
          icon={
            <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" />
            </svg>
          }
        />
        <StatsCard
          title="Active Sprint"
          value={activeSprint?.cards?.length || 0}
          subtitle={activeSprint?.name}
          icon={
            <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
            </svg>
          }
        />
      </div>

      {/* Quick Actions */}
      <div className="flex gap-4">
        <Button
          onClick={() => triggerSync()}
          disabled={isSyncing}
          className="bg-[#F8C630] text-[#1B1B1E] hover:bg-[#F8C630]/90"
        >
          {isSyncing ? 'Syncing...' : 'Sync Now'}
        </Button>
      </div>

      {/* Recent Commits */}
      <div className="bg-[#1B1B1E] border border-[#F8C630]/20 rounded-lg p-4">
        <h2 className="text-lg font-semibold text-[#FBFFFE] mb-4">Recent Commits</h2>
        {commits?.commits.slice(0, 5).map((commit) => (
          <div key={commit.id} className="flex items-center justify-between py-3 border-b border-[#FBFFFE]/10 last:border-0">
            <div className="flex-1 min-w-0">
              <p className="text-[#FBFFFE] truncate">{commit.message}</p>
              <p className="text-[#FBFFFE]/50 text-sm">
                {commit.sha.slice(0, 7)} · {new Date(commit.date).toLocaleDateString()}
              </p>
            </div>
            {commit.jiraCardKey && (
              <span className="ml-2 px-2 py-1 bg-[#F8C630]/20 text-[#F8C630] text-xs rounded">
                {commit.jiraCardKey}
              </span>
            )}
          </div>
        ))}
      </div>
    </div>
  )
}

// StatsCard subcomponent
function StatsCard({ title, value, subtitle, icon }: { title: string; value: number; subtitle?: string; icon: React.ReactNode }) {
  return (
    <div className="bg-[#1B1B1E] border border-[#F8C630]/20 rounded-lg p-4">
      <div className="flex items-center justify-between mb-2">
        <span className="text-[#F8C630]">{icon}</span>
      </div>
      <p className="text-2xl font-bold text-[#FBFFFE]">{value}</p>
      <p className="text-sm text-[#FBFFFE]/60">{title}</p>
      {subtitle && <p className="text-xs text-[#FBFFFE]/40">{subtitle}</p>}
    </div>
  )
}
```

**Step 2: Commit**

```bash
cd frontend && git add src/presentation/pages/DashboardHomePage.tsx && git commit -m "feat: add dashboard home page with stats"
```

---

## Task 4: Create Repositories Page

**Files:**
- Create: `frontend/src/presentation/pages/ReposPage.tsx`

**Step 1: Create ReposPage component**

```tsx
import { useState } from 'react'
import { useListReposQuery, useAddRepoMutation, useDeleteRepoMutation } from '@/infrastructure/services/repo.service'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'

export function ReposPage() {
  const { data: repos, isLoading } = useListReposQuery()
  const [addRepo] = useAddRepoMutation()
  const [deleteRepo] = useDeleteRepoMutation()
  const [newRepoUrl, setNewRepoUrl] = useState('')

  const handleAddRepo = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!newRepoUrl.trim()) return
    await addRepo({ url: newRepoUrl })
    setNewRepoUrl('')
  }

  const handleDelete = async (id: string) => {
    if (confirm('Are you sure you want to remove this repository?')) {
      await deleteRepo(id)
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold text-[#FBFFFE]">Repositories</h1>
      </div>

      {/* Add Repo Form */}
      <form onSubmit={handleAddRepo} className="flex gap-2">
        <Input
          type="url"
          placeholder="https://github.com/owner/repo"
          value={newRepoUrl}
          onChange={(e) => setNewRepoUrl(e.target.value)}
          className="bg-[#1B1B1E] border-[#F8C630]/20 text-[#FBFFFE] placeholder:text-[#FBFFFE]/40"
        />
        <Button type="submit" className="bg-[#F8C630] text-[#1B1B1E] hover:bg-[#F8C630]/90">
          Add Repository
        </Button>
      </form>

      {/* Repo List */}
      {isLoading ? (
        <p className="text-[#FBFFFE]/60">Loading...</p>
      ) : repos?.length === 0 ? (
        <p className="text-[#FBFFFE]/60">No repositories tracked yet.</p>
      ) : (
        <div className="space-y-3">
          {repos?.map((repo) => (
            <div
              key={repo.id}
              className="bg-[#1B1B1E] border border-[#F8C630]/20 rounded-lg p-4 flex items-center justify-between"
            >
              <div className="flex items-center gap-4">
                {/* Provider Icon */}
                <div className={`w-10 h-10 rounded-lg flex items-center justify-center ${
                  repo.provider === 'github' ? 'bg-[#1B1B1E]' : 'bg-[#FC6D26]'
                }`}>
                  <svg className="w-5 h-5 text-[#FBFFFE]" fill="currentColor" viewBox="0 0 24 24">
                    <path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z"/>
                  </svg>
                </div>
                <div>
                  <p className="font-semibold text-[#FBFFFE]">{repo.name}</p>
                  <p className="text-sm text-[#FBFFFE]/60">{repo.owner}</p>
                </div>
              </div>
              <div className="flex items-center gap-4">
                <span className={`px-2 py-1 text-xs rounded ${
                  repo.isValid ? 'bg-green-500/20 text-green-400' : 'bg-red-500/20 text-red-400'
                }`}>
                  {repo.isValid ? 'Valid' : 'Invalid'}
                </span>
                <button
                  onClick={() => handleDelete(repo.id)}
                  className="text-[#FBFFFE]/60 hover:text-red-400 transition-colors"
                >
                  <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                  </svg>
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
```

**Step 2: Commit**

```bash
cd frontend && git add src/presentation/pages/ReposPage.tsx && git commit -m "feat: add repositories page"
```

---

## Task 5: Create Commits Page

**Files:**
- Create: `frontend/src/presentation/pages/CommitsPage.tsx`

**Step 1: Create CommitsPage component**

```tsx
import { useState } from 'react'
import { useListCommitsQuery, useGetMissingCommitsQuery, useLinkToJiraMutation } from '@/infrastructure/services/commit.service'
import { useListReposQuery } from '@/infrastructure/services/repo.service'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'

export function CommitsPage() {
  const [repoId, setRepoId] = useState<string>('')
  const [jiraKey, setJiraKey] = useState('')
  const [showUnlinked, setShowUnlinked] = useState(false)

  const filters = {
    ...(repoId && { repoId }),
    ...(jiraKey && { jiraKey })
  }

  const { data: commits } = useListCommitsQuery(filters)
  const { data: missingCommits } = useGetMissingCommitsQuery(undefined, { skip: !showUnlinked })
  const { data: repos } = useListReposQuery()
  const [linkToJira] = useLinkToJiraMutation()

  const displayCommits = showUnlinked ? missingCommits : commits?.commits

  const handleLink = async (sha: string, key: string) => {
    if (!key.trim()) return
    await linkToJira({ sha, jiraKey: key })
  }

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold text-[#FBFFFE]">Commits</h1>

      {/* Filters */}
      <div className="flex flex-wrap gap-4">
        <select
          value={repoId}
          onChange={(e) => setRepoId(e.target.value)}
          className="bg-[#1B1B1E] border border-[#F8C630]/20 rounded-lg px-4 py-2 text-[#FBFFFE]"
        >
          <option value="">All Repositories</option>
          {repos?.map((repo) => (
            <option key={repo.id} value={repo.id}>{repo.owner}/{repo.name}</option>
          ))}
        </select>

        <Input
          placeholder="Filter by Jira key..."
          value={jiraKey}
          onChange={(e) => setJiraKey(e.target.value)}
          className="bg-[#1B1B1E] border-[#F8C630]/20 text-[#FBFFFE] placeholder:text-[#FBFFFE]/40 w-48"
        />

        <Button
          variant={showUnlinked ? 'default' : 'outline'}
          onClick={() => setShowUnlinked(!showUnlinked)}
          className={showUnlinked ? 'bg-[#F8C630] text-[#1B1B1E]' : 'border-[#F8C630] text-[#F8C630]'}
        >
          Show Unlinked Only
        </Button>
      </div>

      {/* Commits Table */}
      <div className="bg-[#1B1B1E] border border-[#F8C630]/20 rounded-lg overflow-hidden">
        <table className="w-full">
          <thead className="bg-[#F8C630]/10">
            <tr>
              <th className="px-4 py-3 text-left text-[#FBFFFE] font-semibold text-sm">SHA</th>
              <th className="px-4 py-3 text-left text-[#FBFFFE] font-semibold text-sm">Message</th>
              <th className="px-4 py-3 text-left text-[#FBFFFE] font-semibold text-sm">Author</th>
              <th className="px-4 py-3 text-left text-[#FBFFFE] font-semibold text-sm">Date</th>
              <th className="px-4 py-3 text-left text-[#FBFFFE] font-semibold text-sm">Jira</th>
            </tr>
          </thead>
          <tbody>
            {displayCommits?.map((commit) => (
              <tr key={commit.id} className="border-t border-[#FBFFFE]/10">
                <td className="px-4 py-3">
                  <code className="text-[#F8C630] text-sm">{commit.sha.slice(0, 7)}</code>
                </td>
                <td className="px-4 py-3 text-[#FBFFFE] max-w-xs truncate">{commit.message}</td>
                <td className="px-4 py-3 text-[#FBFFFE]/60 text-sm">{commit.author}</td>
                <td className="px-4 py-3 text-[#FBFFFE]/60 text-sm">
                  {new Date(commit.date).toLocaleDateString()}
                </td>
                <td className="px-4 py-3">
                  {commit.jiraCardKey ? (
                    <span className="px-2 py-1 bg-[#F8C630]/20 text-[#F8C630] text-xs rounded">
                      {commit.jiraCardKey}
                    </span>
                  ) : (
                    <span className="text-[#FBFFFE]/40 text-sm">-</span>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>

        {displayCommits?.length === 0 && (
          <p className="p-4 text-center text-[#FBFFFE]/60">No commits found.</p>
        )}
      </div>
    </div>
  )
}
```

**Step 2: Commit**

```bash
cd frontend && git add src/presentation/pages/CommitsPage.tsx && git commit -m "feat: add commits page with filters"
```

---

## Task 6: Create Jira Page

**Files:**
- Create: `frontend/src/presentation/pages/JiraPage.tsx`

**Step 1: Create JiraPage component**

```tsx
import { useListSprintsQuery, useGetActiveSprintQuery, useListCardsQuery } from '@/infrastructure/services/jira.service'
import { Card, CardContent } from '@/components/ui/card'

export function JiraPage() {
  const { data: sprints } = useListSprintsQuery()
  const { data: activeSprint } = useGetActiveSprintQuery()
  const { data: cards } = useListCardsQuery(activeSprint?.id)

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold text-[#FBFFFE]">Jira</h1>

      {/* Active Sprint */}
      {activeSprint ? (
        <div className="bg-[#1B1B1E] border border-[#F8C630]/20 rounded-lg p-4">
          <div className="flex items-center justify-between mb-4">
            <div>
              <h2 className="text-lg font-semibold text-[#FBFFFE]">{activeSprint.name}</h2>
              <p className="text-sm text-[#FBFFFE]/60">
                {activeSprint.startDate && new Date(activeSprint.startDate).toLocaleDateString()} - {' '}
                {activeSprint.endDate && new Date(activeSprint.endDate).toLocaleDateString()}
              </p>
            </div>
            <span className="px-3 py-1 bg-green-500/20 text-green-400 text-sm rounded">
              {activeSprint.state}
            </span>
          </div>

          {/* Cards Grid */}
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {cards?.cards?.map((card) => (
              <Card key={card.key} className="bg-[#F8C630]/5 border-[#F8C630]/20">
                <CardContent className="pt-4">
                  <div className="flex items-start justify-between mb-2">
                    <span className="font-semibold text-[#F8C630]">{card.key}</span>
                    <span className={`px-2 py-0.5 text-xs rounded ${
                      card.status === 'Done' ? 'bg-green-500/20 text-green-400' :
                      card.status === 'In Progress' ? 'bg-blue-500/20 text-blue-400' :
                      'bg-gray-500/20 text-gray-400'
                    }`}>
                      {card.status}
                    </span>
                  </div>
                  <p className="text-[#FBFFFE] text-sm mb-2">{card.summary}</p>
                  {card.assignee && (
                    <p className="text-[#FBFFFE]/60 text-xs">Assignee: {card.assignee}</p>
                  )}
                </CardContent>
              </Card>
            ))}
          </div>
        </div>
      ) : (
        <div className="bg-[#1B1B1E] border border-[#F8C630]/20 rounded-lg p-8 text-center">
          <p className="text-[#FBFFFE]/60">No active sprint found. Configure your Jira integration in Settings.</p>
        </div>
      )}

      {/* All Sprints */}
      <div>
        <h2 className="text-lg font-semibold text-[#FBFFFE] mb-4">All Sprints</h2>
        <div className="space-y-2">
          {sprints?.map((sprint) => (
            <div
              key={sprint.id}
              className="bg-[#1B1B1E] border border-[#F8C630]/20 rounded-lg p-4 flex items-center justify-between"
            >
              <div>
                <p className="font-medium text-[#FBFFFE]">{sprint.name}</p>
                <p className="text-sm text-[#FBFFFE]/60">
                  {sprint.startDate && new Date(sprint.startDate).toLocaleDateString()}
                </p>
              </div>
              <span className={`px-2 py-1 text-xs rounded ${
                sprint.state === 'active' ? 'bg-green-500/20 text-green-400' :
                sprint.state === 'closed' ? 'bg-gray-500/20 text-gray-400' :
                'bg-blue-500/20 text-blue-400'
              }`}>
                {sprint.state}
              </span>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
```

**Step 2: Commit**

```bash
cd frontend && git add src/presentation/pages/JiraPage.tsx && git commit -m "feat: add jira page with sprints"
```

---

## Task 7: Create Reports Page

**Files:**
- Create: `frontend/src/presentation/pages/ReportsPage.tsx`

**Step 1: Create ReportsPage component**

```tsx
import { useState } from 'react'
import { useListReportsQuery, useGenerateReportMutation, useDeleteReportMutation } from '@/infrastructure/services/report.service'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'

export function ReportsPage() {
  const [date, setDate] = useState(new Date().toISOString().split('T')[0])
  const { data: reports, isLoading } = useListReportsQuery()
  const [generateReport, { isLoading: isGenerating }] = useGenerateReportMutation()
  const [deleteReport] = useDeleteReportMutation()

  const handleGenerate = async () => {
    await generateReport(date)
  }

  const handleDelete = async (id: string) => {
    if (confirm('Are you sure you want to delete this report?')) {
      await deleteReport(id)
    }
  }

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold text-[#FBFFFE]">Reports</h1>

      {/* Generate Report */}
      <div className="bg-[#1B1B1E] border border-[#F8C630]/20 rounded-lg p-4">
        <h2 className="text-lg font-semibold text-[#FBFFFE] mb-4">Generate Report</h2>
        <div className="flex gap-4 items-end">
          <div>
            <label className="block text-sm text-[#FBFFFE]/60 mb-1">Date</label>
            <Input
              type="date"
              value={date}
              onChange={(e) => setDate(e.target.value)}
              className="bg-[#1B1B1E] border-[#F8C630]/20 text-[#FBFFFE]"
            />
          </div>
          <Button
            onClick={handleGenerate}
            disabled={isGenerating}
            className="bg-[#F8C630] text-[#1B1B1E] hover:bg-[#F8C630]/90"
          >
            {isGenerating ? 'Generating...' : 'Generate'}
          </Button>
        </div>
      </div>

      {/* Reports List */}
      <div>
        <h2 className="text-lg font-semibold text-[#FBFFFE] mb-4">Past Reports</h2>
        {isLoading ? (
          <p className="text-[#FBFFFE]/60">Loading...</p>
        ) : reports?.reports.length === 0 ? (
          <p className="text-[#FBFFFE]/60">No reports generated yet.</p>
        ) : (
          <div className="space-y-2">
            {reports?.reports.map((report) => (
              <div
                key={report.id}
                className="bg-[#1B1B1E] border border-[#F8C630]/20 rounded-lg p-4 flex items-center justify-between"
              >
                <div>
                  <p className="font-medium text-[#FBFFFE]">{report.title}</p>
                  <p className="text-sm text-[#FBFFFE]/60">
                    {report.commitsCount} commits · {report.jiraCardsCount} Jira cards
                  </p>
                </div>
                <div className="flex items-center gap-2">
                  {report.fileUrl && (
                    <a
                      href={report.fileUrl}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-[#F8C630] hover:text-[#F8C630]/80 text-sm"
                    >
                      Download
                    </a>
                  )}
                  <button
                    onClick={() => handleDelete(report.id)}
                    className="text-[#FBFFFE]/60 hover:text-red-400 transition-colors"
                  >
                    Delete
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
```

**Step 2: Commit**

```bash
cd frontend && git add src/presentation/pages/ReportsPage.tsx && git commit -m "feat: add reports page"
```

---

## Task 8: Create Settings Page

**Files:**
- Create: `frontend/src/presentation/pages/SettingsPage.tsx`

**Step 1: Create SettingsPage component**

```tsx
import { useState } from 'react'
import { useGetProfileQuery, useUpdateProfileMutation, useValidateIntegrationsMutation } from '@/infrastructure/services/user.service'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'

export function SettingsPage() {
  const { data: profile } = useGetProfileQuery()
  const [updateProfile] = useUpdateProfileMutation()
  const [validate] = useValidateIntegrationsMutation()

  const [formData, setFormData] = useState({
    github_token: '',
    gitlab_token: '',
    gitlab_url: 'https://gitlab.com',
    jira_email: '',
    jira_token: '',
    jira_workspace: '',
    jira_username: ''
  })

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    const data = Object.fromEntries(
      Object.entries(formData).filter(([_, v]) => v.trim() !== '')
    )
    await updateProfile(data)
  }

  const handleValidate = async () => {
    await validate()
  }

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold text-[#FBFFFE]">Settings</h1>

      {/* Profile */}
      <div className="bg-[#1B1B1E] border border-[#F8C630]/20 rounded-lg p-4">
        <h2 className="text-lg font-semibold text-[#FBFFFE] mb-4">Profile</h2>
        <p className="text-[#FBFFFE]/60">{profile?.email}</p>
      </div>

      {/* Integrations */}
      <form onSubmit={handleSubmit} className="bg-[#1B1B1E] border border-[#F8C630]/20 rounded-lg p-4 space-y-4">
        <h2 className="text-lg font-semibold text-[#FBFFFE]">Integrations</h2>

        {/* GitHub */}
        <div className="space-y-2">
          <label className="block text-sm font-medium text-[#FBFFFE]">GitHub</label>
          <Input
            type="password"
            placeholder="ghp_xxxxxxxxxxxx"
            value={formData.github_token}
            onChange={(e) => setFormData({ ...formData, github_token: e.target.value })}
            className="bg-[#1B1B1E] border-[#F8C630]/20 text-[#FBFFFE] placeholder:text-[#FBFFFE]/40"
          />
          <p className="text-xs text-[#FBFFFE]/40">
            {profile?.hasGithubToken ? '✓ Configured' : 'Not configured'}
          </p>
        </div>

        {/* GitLab */}
        <div className="space-y-2">
          <label className="block text-sm font-medium text-[#FBFFFE]">GitLab</label>
          <Input
            type="password"
            placeholder="Personal Access Token"
            value={formData.gitlab_token}
            onChange={(e) => setFormData({ ...formData, gitlab_token: e.target.value })}
            className="bg-[#1B1B1E] border-[#F8C630]/20 text-[#FBFFFE] placeholder:text-[#FBFFFE]/40 mb-2"
          />
          <Input
            type="url"
            placeholder="https://gitlab.com"
            value={formData.gitlab_url}
            onChange={(e) => setFormData({ ...formData, gitlab_url: e.target.value })}
            className="bg-[#1B1B1E] border-[#F8C630]/20 text-[#FBFFFE] placeholder:text-[#FBFFFE]/40"
          />
          <p className="text-xs text-[#FBFFFE]/40">
            {profile?.hasGitlabToken ? '✓ Configured' : 'Not configured'}
          </p>
        </div>

        {/* Jira */}
        <div className="space-y-2">
          <label className="block text-sm font-medium text-[#FBFFFE]">Jira</label>
          <Input
            type="email"
            placeholder="Email"
            value={formData.jira_email}
            onChange={(e) => setFormData({ ...formData, jira_email: e.target.value })}
            className="bg-[#1B1B1E] border-[#F8C630]/20 text-[#FBFFFE] placeholder:text-[#FBFFFE]/40 mb-2"
          />
          <Input
            type="password"
            placeholder="API Token"
            value={formData.jira_token}
            onChange={(e) => setFormData({ ...formData, jira_token: e.target.value })}
            className="bg-[#1B1B1E] border-[#F8C630]/20 text-[#FBFFFE] placeholder:text-[#FBFFFE]/40 mb-2"
          />
          <Input
            type="text"
            placeholder="Workspace (e.g., myteam.atlassian.net)"
            value={formData.jira_workspace}
            onChange={(e) => setFormData({ ...formData, jira_workspace: e.target.value })}
            className="bg-[#1B1B1E] border-[#F8C630]/20 text-[#FBFFFE] placeholder:text-[#FBFFFE]/40 mb-2"
          />
          <Input
            type="text"
            placeholder="Username"
            value={formData.jira_username}
            onChange={(e) => setFormData({ ...formData, jira_username: e.target.value })}
            className="bg-[#1B1B1E] border-[#F8C630]/20 text-[#FBFFFE] placeholder:text-[#FBFFFE]/40"
          />
          <p className="text-xs text-[#FBFFFE]/40">
            {profile?.hasJiraToken ? '✓ Configured' : 'Not configured'}
          </p>
        </div>

        {/* Actions */}
        <div className="flex gap-2 pt-4">
          <Button type="submit" className="bg-[#F8C630] text-[#1B1B1E] hover:bg-[#F8C630]/90">
            Save Changes
          </Button>
          <Button type="button" onClick={handleValidate} variant="outline" className="border-[#F8C630] text-[#F8C630]">
            Validate
          </Button>
        </div>
      </form>
    </div>
  )
}
```

**Step 2: Commit**

```bash
cd frontend && git add src/presentation/pages/SettingsPage.tsx && git commit -m "feat: add settings page with integrations"
```

---

## Task 9: Verify Build

**Step 1: Run build**

```bash
cd frontend && npm run build
```

**Step 2: Verify no errors**

Expected: Build succeeds

**Step 3: Commit if any changes**

```bash
git add -A && git commit -m "fix: verify dashboard build"
```

---

## Summary

| Task | Description |
|------|-------------|
| 1 | Dashboard layout with Sidebar and TopBar |
| 2 | PDT-styled reusable Button component |
| 3 | Dashboard home page with stats |
| 4 | Repositories management page |
| 5 | Commits listing page with filters |
| 6 | Jira sprints and cards page |
| 7 | Reports generation and listing page |
| 8 | Settings with integration tokens |
| 9 | Verify build |
