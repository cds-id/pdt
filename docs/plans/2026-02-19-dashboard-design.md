# PDT Dashboard Design

> **For Claude:** Use superpowers:writing-plans to create implementation plan

**Goal:** Create dashboard UI pages that use the already-mapped API services

**Architecture:** Sidebar navigation with PDT branding, main content area with dark theme, responsive design

**Tech Stack:** React, TypeScript, Tailwind CSS, shadcn/ui, RTK Query

---

## Color Palette

| Role | Color | Usage |
|------|-------|-------|
| Primary | `#F8C630` | Sidebar, accents, highlights |
| Background | `#1B1B1E` | Main content area |
| Foreground | `#FBFFFE` | Text on dark backgrounds |
| Accent | `#96031A` | Hover states, important actions |
| Border | `#F8C630` | Card borders, separators |

---

## Layout Structure

### DashboardLayout (Sidebar + Content)

```
+------------------+----------------------------------------+
|     SIDEBAR      |              TOP BAR                  |
|  (240px fixed)  |  (user dropdown, notifications)       |
|                  +----------------------------------------+
|  - Logo          |                                        |
|  - Navigation    |           MAIN CONTENT                |
|    - Dashboard   |     (padded, scrollable)             |
|    - Repos       |                                        |
|    - Commits     |     - Page header                     |
|    - Jira        |     - Stats cards                    |
|    - Reports     |     - Content grid                    |
|    - Settings    |                                        |
|                  |                                        |
+------------------+----------------------------------------+
```

### Responsive Breakpoints

- **Mobile (<768px):** Hamburger menu, full-width content
- **Tablet (768-1024px):** Collapsible sidebar
- **Desktop (>1024px):** Fixed sidebar

---

## Page Designs

### 1. Dashboard Home (`/dashboard`)

**Layout:**
- Welcome header with user email
- 3-column stats grid (desktop), 1-column (mobile)
- Recent commits section
- Quick actions section

**Stats Cards:**
| Card | Data Source | Display |
|------|-------------|---------|
| Total Commits (30d) | `/api/commits` + filter | Count + trend |
| Linked to Jira | `/api/commits` + `has_link=true` | Count + percentage |
| Active Sprint Cards | `/api/jira/active-sprint` | Count + sprint name |

**Recent Commits:**
- Last 5 commits from `/api/commits`
- Show: SHA (7 chars), message (truncated), date, repo name, Jira key badge

**Quick Actions:**
- "Sync Now" button → POST `/api/sync/commits`
- "Generate Report" button → opens date picker → POST `/api/reports/generate`

---

### 2. Repositories Page (`/dashboard/repos`)

**Layout:**
- Page header with "Add Repository" button
- Repository list (cards or table)

**Add Repository:**
- Modal or inline form
- Input: Repository URL
- Action: POST `/api/repos`

**Repository List:**
- Fields: Name, Owner, Provider (GitHub/GitLab icon), Status badge, Last synced, Actions
- Actions: Validate, Delete

**API Endpoints:**
- GET `/api/repos` - list repos
- POST `/api/repos` - add repo
- DELETE `/api/repos/:id` - remove repo
- POST `/api/repos/:id/validate` - validate repo

---

### 3. Commits Page (`/dashboard/commits`)

**Layout:**
- Filter bar (repo, date range, Jira key, link status)
- Commit table

**Filters:**
| Filter | Type | API Param |
|--------|------|-----------|
| Repository | Select dropdown | `repo_id` |
| Date Range | Date picker | `from`, `to` |
| Jira Key | Text input | `jira_card_key` |
| Link Status | Toggle | `has_link` |

**Commit Table Columns:**
- SHA (7 chars, clickable → external link)
- Message (truncated, full on hover)
- Author
- Date
- Repository
- Jira Key (badge or "-")
- Actions (Link to Jira)

**API Endpoints:**
- GET `/api/commits` - list with filters
- GET `/api/commits/missing` - unlinked commits
- POST `/api/commits/:sha/link` - manual link

---

### 4. Jira Page (`/dashboard/jira`)

**Layout:**
- Sprint selector tabs
- Active sprint overview
- Cards grid

**Sprint Overview:**
- Sprint name, state (active/closed/future)
- Start date - End date
- Card count by status

**Cards Grid:**
- Card key (badge)
- Summary
- Status (color-coded)
- Assignee
- Linked commits count

**API Endpoints:**
- GET `/api/jira/sprints` - list sprints
- GET `/api/jira/active-sprint` - current sprint
- GET `/api/jira/cards` - cards in sprint
- GET `/api/jira/cards/:key` - card detail with commits

---

### 5. Reports Page (`/dashboard/reports`)

**Layout:**
- "Generate Report" section
- Reports list

**Generate Report:**
- Date picker (default: today)
- Template selector (optional)
- Button: Generate → POST `/api/reports/generate`

**Reports List:**
- Table: Date, Title, Commits count, Cards count, Actions
- Actions: View, Download (if file_url), Delete

**API Endpoints:**
- POST `/api/reports/generate` - create report
- GET `/api/reports` - list reports
- GET `/api/reports/:id` - get single report
- DELETE `/api/reports/:id` - delete report

---

### 6. Settings Page (`/dashboard/settings`)

**Layout:**
- Profile section (email, read-only)
- Integration tokens section

**Integration Form:**
| Integration | Fields |
|-------------|--------|
| GitHub | Token input |
| GitLab | Token input, URL input |
| Jira | Email, Token, Workspace, Username |

**Validation:**
- "Validate" button → POST `/api/user/profile/validate`
- Show configured/unconfigured status per integration

**API Endpoints:**
- GET `/api/user/profile` - get current settings
- PUT `/api/user/profile` - update tokens
- POST `/api/user/profile/validate` - validate integrations

---

## Components

### Reusable Components Needed

1. **Sidebar** - Navigation with PDT branding
2. **TopBar** - User dropdown, page title
3. **StatsCard** - Icon, value, label, trend
4. **DataTable** - Sortable, filterable table
5. **FilterBar** - Reusable filter controls
6. **Modal** - For forms and confirmations
7. **Badge** - Status indicators
8. **Button** - Primary, secondary, outline variants (PDT themed)
9. **Input** - Text, password, date picker
10. **Select** - Dropdown selections

---

## API Service Alignment

The frontend services are already mapped. This implementation focuses on:

1. Using existing hooks from services
2. Adding any missing response type mappings
3. Creating pages that consume these hooks

---

## Responsive Behavior

| Component | Mobile | Tablet | Desktop |
|-----------|--------|--------|---------|
| Sidebar | Hidden, hamburger menu | Collapsed icons | Full width |
| Stats grid | 1 column | 2 columns | 3 columns |
| Table | Horizontal scroll | Scroll | Full display |
| Cards | 1 column | 2 columns | 3-4 columns |

---

## Acceptance Criteria

- [ ] Dashboard loads with user stats
- [ ] All 6 pages are navigable
- [ ] Forms submit to correct API endpoints
- [ ] Responsive on mobile (iPhone SE)
- [ ] PDT branding consistent with landing page
- [ ] Loading and error states handled
