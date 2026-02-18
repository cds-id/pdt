# PDT Landing Page & API Mapping Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Create distinctive PDT landing page and map all 20+ backend API endpoints to frontend services

**Architecture:** Full landing page with hero, features, integrations sections using React + Tailwind. API services layer with TypeScript interfaces for each backend endpoint.

**Tech Stack:** React, Tailwind CSS, TypeScript, Vite

---

## Task 1: Update index.html Title

**Files:**
- Modify: `frontend/index.html`

**Step 1: Update page title**

```html
<title>PDT - Personal Daily Tracker</title>
```

**Step 2: Commit**

```bash
cd frontend && git add index.html && git commit -m "feat: update page title to PDT"
```

---

## Task 2: Add PDT Branding Colors to Tailwind

**Files:**
- Modify: `frontend/tailwind.config.mjs`
- Modify: `frontend/src/index.css`

**Step 1: Add PDT brand colors to tailwind config**

In `tailwind.config.mjs`, add to `extend.colors`:

```javascript
pdt: {
  primary: {
    DEFAULT: '#1e1b4b',    // Indigo 950
    light: '#312e81',      // Indigo 900
    dark: '#0f0d24',       // Darker shade
  },
  accent: {
    DEFAULT: '#14b8a6',    // Teal 500
    hover: '#2dd4bf',      // Teal 400
    dark: '#0f766e',      // Teal 700
  }
}
```

**Step 2: Update CSS variables in index.css**

Replace `--primary: 222.2 47.4% 11.2%;` with:

```css
--primary: 247 81% 29%;      /* #1e1b4b */
--primary-foreground: 210 40% 98%;
--accent: 168 74% 53%;       /* #14b8a6 */
--accent-foreground: 222.2 47.4% 11.2%;
```

**Step 3: Commit**

```bash
cd frontend && git add tailwind.config.mjs src/index.css && git commit -m "feat: add PDT brand colors"
```

---

## Task 3: Create API Constants with All Endpoints

**Files:**
- Modify: `frontend/src/infrastructure/constants/api.constants.ts`

**Step 1: Write complete API constants**

```typescript
/**
 * API service constants - All backend endpoints
 */
export const API_CONSTANTS = {
  BASE_URL: import.meta.env.VITE_API_URL || 'http://localhost:3000',
  API_PREFIX: '/api',

  // Auth Endpoints
  AUTH: {
    REGISTER: '/auth/register',
    LOGIN: '/auth/login',
    LOGOUT: '/auth/logout'
  },

  // User Endpoints
  USER: {
    PROFILE: '/user/profile',
    UPDATE: '/user/profile',
    VALIDATE: '/user/profile/validate'
  },

  // Repository Endpoints
  REPOS: {
    LIST: '/repos',
    ADD: '/repos',
    DELETE: (id: string) => `/repos/${id}`,
    VALIDATE: (id: string) => `/repos/${id}/validate`
  },

  // Sync Endpoints
  SYNC: {
    COMMITS: '/sync/commits',
    STATUS: '/sync/status'
  },

  // Commit Endpoints
  COMMITS: {
    LIST: '/commits',
    MISSING: '/commits/missing',
    LINK: (sha: string) => `/commits/${sha}/link`
  },

  // Jira Endpoints
  JIRA: {
    SPRINTS: '/jira/sprints',
    SPRINT: (id: string) => `/jira/sprints/${id}`,
    ACTIVE_SPRINT: '/jira/active-sprint',
    CARDS: '/jira/cards',
    CARD: (key: string) => `/jira/cards/${key}`
  },

  // Report Endpoints
  REPORTS: {
    GENERATE: '/reports/generate',
    LIST: '/reports',
    GET: (id: string) => `/reports/${id}`,
    DELETE: (id: string) => `/reports/${id}`,
    TEMPLATES_LIST: '/reports/templates',
    TEMPLATES_CREATE: '/reports/templates',
    TEMPLATES_UPDATE: (id: string) => `/reports/templates/${id}`,
    TEMPLATES_DELETE: (id: string) => `/reports/templates/${id}`,
    TEMPLATES_PREVIEW: '/reports/templates/preview'
  }
}

// Helper to build full URL
export const buildUrl = (endpoint: string): string => {
  return `${API_CONSTANTS.BASE_URL}${API_CONSTANTS.API_PREFIX}${endpoint}`
}
```

**Step 2: Commit**

```bash
cd frontend && git add src/infrastructure/constants/api.constants.ts && git commit -m "feat: add all API endpoint constants"
```

---

## Task 4: Create Auth Service

**Files:**
- Create: `frontend/src/infrastructure/services/auth.service.ts`

**Step 1: Write auth service**

```typescript
import { API_CONSTANTS, buildUrl } from '../constants/api.constants'

export interface LoginRequest {
  email: string
  password: string
}

export interface RegisterRequest {
  email: string
  password: string
  name: string
}

export interface AuthResponse {
  token: string
  user: {
    id: string
    email: string
    name: string
  }
}

export const authService = {
  async login(credentials: LoginRequest): Promise<AuthResponse> {
    const response = await fetch(buildUrl(API_CONSTANTS.AUTH.LOGIN), {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(credentials)
    })
    if (!response.ok) throw new Error('Login failed')
    return response.json()
  },

  async register(data: RegisterRequest): Promise<AuthResponse> {
    const response = await fetch(buildUrl(API_CONSTANTS.AUTH.REGISTER), {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data)
    })
    if (!response.ok) throw new Error('Registration failed')
    return response.json()
  },

  async logout(): Promise<void> {
    const token = localStorage.getItem('token')
    if (!token) return

    await fetch(buildUrl(API_CONSTANTS.AUTH.LOGOUT), {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json'
      }
    })
    localStorage.removeItem('token')
  }
}
```

**Step 2: Commit**

```bash
cd frontend && git add src/infrastructure/services/auth.service.ts && git commit -m "feat: add auth service with login/register"
```

---

## Task 5: Create User Service

**Files:**
- Create: `frontend/src/infrastructure/services/user.service.ts`

**Step 1: Write user service**

```typescript
import { API_CONSTANTS, buildUrl } from '../constants/api.constants'

export interface UserProfile {
  id: string
  email: string
  name: string
  githubToken?: string
  gitlabToken?: string
  jiraToken?: string
  jiraDomain?: string
  createdAt: string
  updatedAt: string
}

export interface UpdateProfileRequest {
  name?: string
  githubToken?: string
  gitlabToken?: string
  jiraToken?: string
  jiraDomain?: string
}

export interface ValidateResponse {
  github: { valid: boolean; error?: string }
  gitlab: { valid: boolean; error?: string }
  jira: { valid: boolean; error?: string }
}

const getAuthHeaders = (): HeadersInit => ({
  'Authorization': `Bearer ${localStorage.getItem('token')}`,
  'Content-Type': 'application/json'
})

export const userService = {
  async getProfile(): Promise<UserProfile> {
    const response = await fetch(buildUrl(API_CONSTANTS.USER.PROFILE), {
      headers: getAuthHeaders()
    })
    if (!response.ok) throw new Error('Failed to fetch profile')
    return response.json()
  },

  async updateProfile(data: UpdateProfileRequest): Promise<UserProfile> {
    const response = await fetch(buildUrl(API_CONSTANTS.USER.UPDATE), {
      method: 'PUT',
      headers: getAuthHeaders(),
      body: JSON.stringify(data)
    })
    if (!response.ok) throw new Error('Failed to update profile')
    return response.json()
  },

  async validateIntegrations(): Promise<ValidateResponse> {
    const response = await fetch(buildUrl(API_CONSTANTS.USER.VALIDATE), {
      method: 'POST',
      headers: getAuthHeaders()
    })
    if (!response.ok) throw new Error('Validation failed')
    return response.json()
  }
}
```

**Step 2: Commit**

```bash
cd frontend && git add src/infrastructure/services/user.service.ts && git commit -m "feat: add user service for profile management"
```

---

## Task 6: Create Repo Service

**Files:**
- Create: `frontend/src/infrastructure/services/repo.service.ts`

**Step 1: Write repo service**

```typescript
import { API_CONSTANTS, buildUrl } from '../constants/api.constants'

export interface Repository {
  id: string
  url: string
  name: string
  provider: 'github' | 'gitlab'
  isValid: boolean
  lastSyncedAt?: string
  createdAt: string
}

export interface AddRepoRequest {
  url: string
}

const getAuthHeaders = (): HeadersInit => ({
  'Authorization': `Bearer ${localStorage.getItem('token')}`,
  'Content-Type': 'application/json'
})

export const repoService = {
  async listRepos(): Promise<Repository[]> {
    const response = await fetch(buildUrl(API_CONSTANTS.REPOS.LIST), {
      headers: getAuthHeaders()
    })
    if (!response.ok) throw new Error('Failed to fetch repositories')
    return response.json()
  },

  async addRepo(data: AddRepoRequest): Promise<Repository> {
    const response = await fetch(buildUrl(API_CONSTANTS.REPOS.ADD), {
      method: 'POST',
      headers: getAuthHeaders(),
      body: JSON.stringify(data)
    })
    if (!response.ok) throw new Error('Failed to add repository')
    return response.json()
  },

  async deleteRepo(id: string): Promise<void> {
    const response = await fetch(buildUrl(API_CONSTANTS.REPOS.DELETE(id)), {
      method: 'DELETE',
      headers: getAuthHeaders()
    })
    if (!response.ok) throw new Error('Failed to delete repository')
  },

  async validateRepo(id: string): Promise<{ valid: boolean; error?: string }> {
    const response = await fetch(buildUrl(API_CONSTANTS.REPOS.VALIDATE(id)), {
      method: 'POST',
      headers: getAuthHeaders()
    })
    if (!response.ok) throw new Error('Failed to validate repository')
    return response.json()
  }
}
```

**Step 2: Commit**

```bash
cd frontend && git add src/infrastructure/services/repo.service.ts && git commit -m "feat: add repo service for repository management"
```

---

## Task 7: Create Sync Service

**Files:**
- Create: `frontend/src/infrastructure/services/sync.service.ts`

**Step 1: Write sync service**

```typescript
import { API_CONSTANTS, buildUrl } from '../constants/api.constants'

export interface SyncStatus {
  lastSyncAt?: string
  isRunning: boolean
  commitsToday: number
}

export interface TriggerSyncResponse {
  success: boolean
  message: string
}

const getAuthHeaders = (): HeadersInit => ({
  'Authorization': `Bearer ${localStorage.getItem('token')}`,
  'Content-Type': 'application/json'
})

export const syncService = {
  async triggerSync(): Promise<TriggerSyncResponse> {
    const response = await fetch(buildUrl(API_CONSTANTS.SYNC.COMMITS), {
      method: 'POST',
      headers: getAuthHeaders()
    })
    if (!response.ok) throw new Error('Failed to trigger sync')
    return response.json()
  },

  async getStatus(): Promise<SyncStatus> {
    const response = await fetch(buildUrl(API_CONSTANTS.SYNC.STATUS), {
      headers: getAuthHeaders()
    })
    if (!response.ok) throw new Error('Failed to fetch sync status')
    return response.json()
  }
}
```

**Step 2: Commit**

```bash
cd frontend && git add src/infrastructure/services/sync.service.ts && git commit -m "feat: add sync service for commit synchronization"
```

---

## Task 8: Create Commit Service

**Files:**
- Create: `frontend/src/infrastructure/services/commit.service.ts`

**Step 1: Write commit service**

```typescript
import { API_CONSTANTS, buildUrl } from '../constants/api.constants'

export interface Commit {
  id: string
  sha: string
  message: string
  author: string
  authorEmail: string
  date: string
  repoId: string
  jiraKey?: string
  url: string
}

export interface CommitFilters {
  repoId?: string
  jiraKey?: string
  fromDate?: string
  toDate?: string
  page?: number
  limit?: number
}

export interface CommitListResponse {
  commits: Commit[]
  total: number
  page: number
  limit: number
}

export interface LinkToJiraRequest {
  jiraKey: string
}

const getAuthHeaders = (): HeadersInit => ({
  'Authorization': `Bearer ${localStorage.getItem('token')}`,
  'Content-Type': 'application/json'
})

export const commitService = {
  async listCommits(filters?: CommitFilters): Promise<CommitListResponse> {
    const params = new URLSearchParams()
    if (filters?.repoId) params.append('repoId', filters.repoId)
    if (filters?.jiraKey) params.append('jiraKey', filters.jiraKey)
    if (filters?.fromDate) params.append('fromDate', filters.fromDate)
    if (filters?.toDate) params.append('toDate', filters.toDate)
    if (filters?.page) params.append('page', String(filters.page))
    if (filters?.limit) params.append('limit', String(filters.limit))

    const url = `${buildUrl(API_CONSTANTS.COMMITS.LIST)}?${params}`
    const response = await fetch(url, { headers: getAuthHeaders() })
    if (!response.ok) throw new Error('Failed to fetch commits')
    return response.json()
  },

  async getMissingCommits(): Promise<Commit[]> {
    const response = await fetch(buildUrl(API_CONSTANTS.COMMITS.MISSING), {
      headers: getAuthHeaders()
    })
    if (!response.ok) throw new Error('Failed to fetch missing commits')
    return response.json()
  },

  async linkToJira(sha: string, data: LinkToJiraRequest): Promise<Commit> {
    const response = await fetch(buildUrl(API_CONSTANTS.COMMITS.LINK(sha)), {
      method: 'POST',
      headers: getAuthHeaders(),
      body: JSON.stringify(data)
    })
    if (!response.ok) throw new Error('Failed to link commit to Jira')
    return response.json()
  }
}
```

**Step 2: Commit**

```bash
cd frontend && git add src/infrastructure/services/commit.service.ts && git commit -m "feat: add commit service for commit management"
```

---

## Task 9: Create Jira Service

**Files:**
- Create: `frontend/src/infrastructure/services/jira.service.ts`

**Step 1: Write jira service**

```typescript
import { API_CONSTANTS, buildUrl } from '../constants/api.constants'

export interface JiraSprint {
  id: string
  name: string
  state: 'active' | 'closed' | 'future'
  startDate?: string
  endDate?: string
  completeDate?: string
}

export interface JiraCard {
  key: string
  summary: string
  status: string
  type: 'story' | 'bug' | 'task' | 'subtask'
  assignee?: string
  priority: string
  created: string
  updated: string
  commits: string[]
}

export interface JiraCardsResponse {
  cards: JiraCard[]
  total: number
}

const getAuthHeaders = (): HeadersInit => ({
  'Authorization': `Bearer ${localStorage.getItem('token')}`,
  'Content-Type': 'application/json'
})

export const jiraService = {
  async listSprints(): Promise<JiraSprint[]> {
    const response = await fetch(buildUrl(API_CONSTANTS.JIRA.SPRINTS), {
      headers: getAuthHeaders()
    })
    if (!response.ok) throw new Error('Failed to fetch sprints')
    return response.json()
  },

  async getSprint(id: string): Promise<JiraSprint> {
    const response = await fetch(buildUrl(API_CONSTANTS.JIRA.SPRINT(id)), {
      headers: getAuthHeaders()
    })
    if (!response.ok) throw new Error('Failed to fetch sprint')
    return response.json()
  },

  async getActiveSprint(): Promise<JiraSprint> {
    const response = await fetch(buildUrl(API_CONSTANTS.JIRA.ACTIVE_SPRINT), {
      headers: getAuthHeaders()
    })
    if (!response.ok) throw new Error('Failed to fetch active sprint')
    return response.json()
  },

  async listCards(sprintId?: string): Promise<JiraCardsResponse> {
    const params = sprintId ? `?sprintId=${sprintId}` : ''
    const response = await fetch(`${buildUrl(API_CONSTANTS.JIRA.CARDS)}${params}`, {
      headers: getAuthHeaders()
    })
    if (!response.ok) throw new Error('Failed to fetch cards')
    return response.json()
  },

  async getCard(key: string): Promise<JiraCard> {
    const response = await fetch(buildUrl(API_CONSTANTS.JIRA.CARD(key)), {
      headers: getAuthHeaders()
    })
    if (!response.ok) throw new Error('Failed to fetch card')
    return response.json()
  }
}
```

**Step 2: Commit**

```bash
cd frontend && git add src/infrastructure/services/jira.service.ts && git commit -m "feat: add jira service for Jira integration"
```

---

## Task 10: Create Report Service

**Files:**
- Create: `frontend/src/infrastructure/services/report.service.ts`

**Step 1: Write report service**

```typescript
import { API_CONSTANTS, buildUrl } from '../constants/api.constants'

export interface Report {
  id: string
  date: string
  content: string
  commitsCount: number
  jiraCardsCount: number
  createdAt: string
}

export interface ReportTemplate {
  id: string
  name: string
  content: string
  createdAt: string
  updatedAt: string
}

export interface GenerateReportRequest {
  date: string
}

export interface ReportListResponse {
  reports: Report[]
  total: number
}

const getAuthHeaders = (): HeadersInit => ({
  'Authorization': `Bearer ${localStorage.getItem('token')}`,
  'Content-Type': 'application/json'
})

export const reportService = {
  async generateReport(date: string): Promise<Report> {
    const response = await fetch(buildUrl(API_CONSTANTS.REPORTS.GENERATE), {
      method: 'POST',
      headers: getAuthHeaders(),
      body: JSON.stringify({ date })
    })
    if (!response.ok) throw new Error('Failed to generate report')
    return response.json()
  },

  async listReports(fromDate?: string, toDate?: string): Promise<ReportListResponse> {
    const params = new URLSearchParams()
    if (fromDate) params.append('fromDate', fromDate)
    if (toDate) params.append('toDate', toDate)

    const url = `${buildUrl(API_CONSTANTS.REPORTS.LIST)}?${params}`
    const response = await fetch(url, { headers: getAuthHeaders() })
    if (!response.ok) throw new Error('Failed to fetch reports')
    return response.json()
  },

  async getReport(id: string): Promise<Report> {
    const response = await fetch(buildUrl(API_CONSTANTS.REPORTS.GET(id)), {
      headers: getAuthHeaders()
    })
    if (!response.ok) throw new Error('Failed to fetch report')
    return response.json()
  },

  async deleteReport(id: string): Promise<void> {
    const response = await fetch(buildUrl(API_CONSTANTS.REPORTS.DELETE(id)), {
      method: 'DELETE',
      headers: getAuthHeaders()
    })
    if (!response.ok) throw new Error('Failed to delete report')
  },

  // Templates
  async createTemplate(name: string, content: string): Promise<ReportTemplate> {
    const response = await fetch(buildUrl(API_CONSTANTS.REPORTS.TEMPLATES_CREATE), {
      method: 'POST',
      headers: getAuthHeaders(),
      body: JSON.stringify({ name, content })
    })
    if (!response.ok) throw new Error('Failed to create template')
    return response.json()
  },

  async listTemplates(): Promise<ReportTemplate[]> {
    const response = await fetch(buildUrl(API_CONSTANTS.REPORTS.TEMPLATES_LIST), {
      headers: getAuthHeaders()
    })
    if (!response.ok) throw new Error('Failed to fetch templates')
    return response.json()
  },

  async updateTemplate(id: string, name: string, content: string): Promise<ReportTemplate> {
    const response = await fetch(buildUrl(API_CONSTANTS.REPORTS.TEMPLATES_UPDATE(id)), {
      method: 'PUT',
      headers: getAuthHeaders(),
      body: JSON.stringify({ name, content })
    })
    if (!response.ok) throw new Error('Failed to update template')
    return response.json()
  },

  async deleteTemplate(id: string): Promise<void> {
    const response = await fetch(buildUrl(API_CONSTANTS.REPORTS.TEMPLATES_DELETE(id)), {
      method: 'DELETE',
      headers: getAuthHeaders()
    })
    if (!response.ok) throw new Error('Failed to delete template')
  },

  async previewTemplate(content: string, date: string): Promise<{ preview: string }> {
    const response = await fetch(buildUrl(API_CONSTANTS.REPORTS.TEMPLATES_PREVIEW), {
      method: 'POST',
      headers: getAuthHeaders(),
      body: JSON.stringify({ content, date })
    })
    if (!response.ok) throw new Error('Failed to preview template')
    return response.json()
  }
}
```

**Step 2: Commit**

```bash
cd frontend && git add src/infrastructure/services/report.service.ts && git commit -m "feat: add report service for daily reports"
```

---

## Task 11: Create Landing Page Component

**Files:**
- Create: `frontend/src/presentation/pages/LandingPage.tsx`

**Step 1: Write landing page**

```tsx
import { Link } from 'react-router-dom'

export function LandingPage() {
  return (
    <div className="min-h-screen bg-slate-50">
      {/* Navigation */}
      <nav className="fixed top-0 left-0 right-0 z-50 bg-white/80 backdrop-blur-md border-b border-slate-200">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between items-center h-16">
            <div className="flex items-center">
              <span className="text-xl font-bold text-pdt-primary">PDT</span>
              <span className="ml-2 text-sm text-slate-600">Personal Daily Tracker</span>
            </div>
            <div className="flex items-center space-x-4">
              <Link
                to="/login"
                className="text-slate-600 hover:text-pdt-primary transition-colors"
              >
                Login
              </Link>
              <Link
                to="/register"
                className="px-4 py-2 bg-pdt-primary text-white rounded-lg hover:bg-pdt-primary-light transition-colors"
              >
                Get Started
              </Link>
            </div>
          </div>
        </div>
      </nav>

      {/* Hero Section */}
      <section className="pt-32 pb-20 px-4">
        <div className="max-w-7xl mx-auto text-center">
          <h1 className="text-5xl md:text-6xl font-bold text-slate-900 mb-6">
            Your Personal Daily
            <span className="text-pdt-accent"> Development Tracker</span>
          </h1>
          <p className="text-xl text-slate-600 mb-8 max-w-2xl mx-auto">
            Automatically track commits across GitHub & GitLab, link Jira cards,
            and generate daily reports. Stay on top of your development work.
          </p>
          <div className="flex justify-center gap-4">
            <Link
              to="/register"
              className="px-8 py-3 bg-pdt-accent text-white font-semibold rounded-lg hover:bg-pdt-accent-hover transition-colors"
            >
              Get Started Free
            </Link>
            <Link
              to="/login"
              className="px-8 py-3 border border-slate-300 text-slate-700 font-semibold rounded-lg hover:bg-slate-100 transition-colors"
            >
              View Demo
            </Link>
          </div>
        </div>
      </section>

      {/* Integrations Section */}
      <section className="py-20 bg-white">
        <div className="max-w-7xl mx-auto px-4">
          <h2 className="text-3xl font-bold text-center text-slate-900 mb-12">
            Seamless Integrations
          </h2>
          <div className="grid md:grid-cols-3 gap-8">
            {/* GitHub */}
            <div className="p-6 border border-slate-200 rounded-xl hover:shadow-lg transition-shadow">
              <div className="w-12 h-12 bg-slate-900 rounded-lg flex items-center justify-center mb-4">
                <svg className="w-6 h-6 text-white" fill="currentColor" viewBox="0 0 24 24">
                  <path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z"/>
                </svg>
              </div>
              <h3 className="text-xl font-semibold text-slate-900 mb-2">GitHub</h3>
              <p className="text-slate-600">Track commits from your GitHub repositories automatically.</p>
            </div>

            {/* GitLab */}
            <div className="p-6 border border-slate-200 rounded-xl hover:shadow-lg transition-shadow">
              <div className="w-12 h-12 bg-orange-500 rounded-lg flex items-center justify-center mb-4">
                <svg className="w-6 h-6 text-white" fill="currentColor" viewBox="0 0 24 24">
                  <path d="M22.65 14.39L12 22.13 1.35 14.39a.84.84 0 0 1-.3-.94l1.22-3.78 2.44-7.51A.42.42 0 0 1 4.82 2a.43.43 0 0 1 .58 0 .42.42 0 0 1 .11.18l2.44 7.49h8.1l2.44-7.51A.42.42 0 0 1 18.6 2a.43.43 0 0 1 .58 0 .42.42 0 0 1 .11.18l2.44 7.51L23 13.45a.84.84 0 0 1-.35.94z"/>
                </svg>
              </div>
              <h3 className="text-xl font-semibold text-slate-900 mb-2">GitLab</h3>
              <p className="text-slate-600">Sync commits from your GitLab projects seamlessly.</p>
            </div>

            {/* Jira */}
            <div className="p-6 border border-slate-200 rounded-xl hover:shadow-lg transition-shadow">
              <div className="w-12 h-12 bg-blue-600 rounded-lg flex items-center justify-center mb-4">
                <svg className="w-6 h-6 text-white" fill="currentColor" viewBox="0 0 24 24">
                  <path d="M11.571 11.513H0a5.218 5.218 0 0 0 5.232 5.215h2.13v2.057A5.215 5.215 0 0 0 12.575 24V12.518a1.005 1.005 0 0 0-1.005-1.005zm5.723-5.756H5.436a5.215 5.215 0 0 0 5.215 5.214h2.129v2.058a5.218 5.218 0 0 0 5.215 5.214V6.758a1.001 1.001 0 0 0-1.001-1.001zM23.013 0H11.455a5.215 5.215 0 0 0 5.215 5.215h2.129v2.057A5.215 5.215 0 0 0 24 12.483V1.005A1.005 1.005 0 0 0 23.013 0z"/>
                </svg>
              </div>
              <h3 className="text-xl font-semibold text-slate-900 mb-2">Jira</h3>
              <p className="text-slate-600">Link commits to Jira cards and track your sprint progress.</p>
            </div>
          </div>
        </div>
      </section>

      {/* Features Section */}
      <section className="py-20 bg-slate-50">
        <div className="max-w-7xl mx-auto px-4">
          <h2 className="text-3xl font-bold text-center text-slate-900 mb-12">
            Everything You Need to Track Your Work
          </h2>
          <div className="grid md:grid-cols-2 lg:grid-cols-4 gap-8">
            <div className="text-center">
              <div className="w-14 h-14 bg-pdt-accent/10 rounded-xl flex items-center justify-center mx-auto mb-4">
                <svg className="w-7 h-7 text-pdt-accent" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4m0 5c0 2.21-3.582 4-8 4s-8-1.79-8-4" />
                </svg>
              </div>
              <h3 className="font-semibold text-slate-900 mb-2">Automatic Sync</h3>
              <p className="text-slate-600 text-sm">Commits synced automatically from GitHub & GitLab</p>
            </div>

            <div className="text-center">
              <div className="w-14 h-14 bg-pdt-accent/10 rounded-xl flex items-center justify-center mx-auto mb-4">
                <svg className="w-7 h-7 text-pdt-accent" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" />
                </svg>
              </div>
              <h3 className="font-semibold text-slate-900 mb-2">Jira Integration</h3>
              <p className="text-slate-600 text-sm">Link commits to Jira cards effortlessly</p>
            </div>

            <div className="text-center">
              <div className="w-14 h-14 bg-pdt-accent/10 rounded-xl flex items-center justify-center mx-auto mb-4">
                <svg className="w-7 h-7 text-pdt-accent" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 17v-2m3 2v-4m3 4v-6m2 10H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                </svg>
              </div>
              <h3 className="font-semibold text-slate-900 mb-2">Daily Reports</h3>
              <p className="text-slate-600 text-sm">Automated daily summaries of your work</p>
            </div>

            <div className="text-center">
              <div className="w-14 h-14 bg-pdt-accent/10 rounded-xl flex items-center justify-center mx-auto mb-4">
                <svg className="w-7 h-7 text-pdt-accent" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
              </div>
              <h3 className="font-semibold text-slate-900 mb-2">Background Sync</h3>
              <p className="text-slate-600 text-sm">Data stays fresh with automatic background updates</p>
            </div>
          </div>
        </div>
      </section>

      {/* How It Works Section */}
      <section className="py-20 bg-white">
        <div className="max-w-7xl mx-auto px-4">
          <h2 className="text-3xl font-bold text-center text-slate-900 mb-12">
            How It Works
          </h2>
          <div className="grid md:grid-cols-3 gap-8">
            <div className="text-center">
              <div className="w-12 h-12 bg-pdt-primary text-white rounded-full flex items-center justify-center mx-auto mb-4 text-xl font-bold">
                1
              </div>
              <h3 className="text-lg font-semibold text-slate-900 mb-2">Connect Your Accounts</h3>
              <p className="text-slate-600">Link your GitHub, GitLab, and Jira accounts with secure tokens</p>
            </div>

            <div className="text-center">
              <div className="w-12 h-12 bg-pdt-primary text-white rounded-full flex items-center justify-center mx-auto mb-4 text-xl font-bold">
                2
              </div>
              <h3 className="text-lg font-semibold text-slate-900 mb-2">Add Repositories</h3>
              <p className="text-slate-600">Select which repositories you want to track</p>
            </div>

            <div className="text-center">
              <div className="w-12 h-12 bg-pdt-primary text-white rounded-full flex items-center justify-center mx-auto mb-4 text-xl font-bold">
                3
              </div>
              <h3 className="text-lg font-semibold text-slate-900 mb-2">Get Daily Reports</h3>
              <p className="text-slate-600">Receive automated daily reports of your development work</p>
            </div>
          </div>
        </div>
      </section>

      {/* CTA Section */}
      <section className="py-20 bg-pdt-primary">
        <div className="max-w-7xl mx-auto px-4 text-center">
          <h2 className="text-3xl font-bold text-white mb-4">
            Start Tracking Your Development Work Today
          </h2>
          <p className="text-slate-300 mb-8 max-w-xl mx-auto">
            Join developers who use PDT to stay organized and track their daily progress effortlessly.
          </p>
          <Link
            to="/register"
            className="inline-block px-8 py-3 bg-pdt-accent text-white font-semibold rounded-lg hover:bg-pdt-accent-hover transition-colors"
          >
            Sign Up Free
          </Link>
        </div>
      </section>

      {/* Footer */}
      <footer className="py-8 bg-slate-900 text-slate-400">
        <div className="max-w-7xl mx-auto px-4 text-center">
          <p>&copy; 2026 PDT - Personal Daily Tracker. All rights reserved.</p>
        </div>
      </footer>
    </div>
  )
}
```

**Step 2: Commit**

```bash
cd frontend && git add src/presentation/pages/LandingPage.tsx && git commit -m "feat: add landing page with hero, features, and integrations"
```

---

## Task 12: Update Routes

**Files:**
- Modify: `frontend/src/presentation/routes/index.tsx`

**Step 1: Read current routes**

```bash
cat frontend/src/presentation/routes/index.tsx
```

**Step 2: Update to add landing page route**

Replace with:

```tsx
import { createBrowserRouter, Navigate } from 'react-router-dom'
import { LandingPage } from '../pages/LandingPage'
import { LoginPage } from '../pages/auth/LoginPage'
import { DashboardPage } from '../pages/dashboard/DashboardPage'
import { DashboardLayout } from '../layouts/DashboardLayout'
import { ProtectedLayout } from '../layouts/ProtectedLayout'

const router = createBrowserRouter([
  {
    path: '/',
    element: <LandingPage />
  },
  {
    path: '/login',
    element: <LoginPage />
  },
  {
    path: '/register',
    element: <LoginPage />
  },
  {
    element: <ProtectedLayout />,
    children: [
      {
        element: <DashboardLayout />,
        children: [
          {
            path: '/dashboard',
            element: <DashboardPage />
          }
        ]
      }
    ]
  },
  {
    path: '*',
    element: <Navigate to="/" replace />
  }
])

export { router }
```

**Step 3: Commit**

```bash
cd frontend && git add src/presentation/routes/index.tsx && git commit -m "feat: add landing page route"
```

---

## Task 13: Update Login Page Branding

**Files:**
- Modify: `frontend/src/presentation/pages/auth/LoginPage.tsx`

**Step 1: Read current login page**

```bash
cat frontend/src/presentation/pages/auth/LoginPage.tsx
```

**Step 2: Update branding colors in login page**

- Change primary buttons to use `bg-pdt-primary`
- Update header to show PDT branding
- Add PDT logo/text to login form

**Step 3: Commit**

```bash
cd frontend && git add src/presentation/pages/auth/LoginPage.tsx && git commit -m "feat: update login page with PDT branding"
```

---

## Task 14: Final Verification

**Step 1: Run frontend to verify**

```bash
cd frontend && npm run dev
```

**Step 2: Verify all changes**
- Landing page loads at /
- All colors applied correctly
- API services are properly typed

**Step 3: Commit any remaining changes**

```bash
cd frontend && git add -A && git commit -m "feat: complete PDT landing page and API integration"
```

---

## Summary

| Task | Description |
|------|-------------|
| 1 | Update index.html title |
| 2 | Add PDT branding colors |
| 3 | Create API constants (20+ endpoints) |
| 4 | Create auth.service.ts |
| 5 | Create user.service.ts |
| 6 | Create repo.service.ts |
| 7 | Create sync.service.ts |
| 8 | Create commit.service.ts |
| 9 | Create jira.service.ts |
| 10 | Create report.service.ts |
| 11 | Create LandingPage.tsx |
| 12 | Update routes |
| 13 | Update LoginPage branding |
| 14 | Final verification |

---

**Plan complete and saved to `docs/plans/2026-02-19-landing-page-api-mapping-design.md`. Two execution options:**

1. **Subagent-Driven (this session)** - I dispatch fresh subagent per task, review between tasks, fast iteration

2. **Parallel Session (separate)** - Open new session with executing-plans, batch execution with checkpoints

**Which approach?**
