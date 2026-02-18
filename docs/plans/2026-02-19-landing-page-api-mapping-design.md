# PDT Landing Page & API Integration Design

## Overview
- **Date**: 2026-02-19
- **Status**: Approved
- **Goal**: Create distinctive landing page and map all backend API endpoints to frontend

---

## Branding

### Color Palette
| Role | Color | Hex |
|------|-------|-----|
| Primary (Dark Navy) | Indigo 950 | `#1e1b4b` |
| Primary Light | Indigo 900 | `#312e81` |
| Accent (Teal) | Teal 500 | `#14b8a6` |
| Accent Hover | Teal 400 | `#2dd4bf` |
| Background | Slate 50 | `#f8fafc` |
| Surface | White | `#ffffff` |
| Text Primary | Slate 900 | `#0f172a` |
| Text Secondary | Slate 600 | `#475569` |
| Border | Slate 200 | `#e2e8f0` |

### Typography
- **Headings**: Bold, high contrast
- **Body**: Inter (current)
- **Tagline**: "Track Your Daily Development Work"

---

## Landing Page Sections

### 1. Hero Section
- **Headline**: "Your Personal Daily Development Tracker"
- **Subheadline**: "Automatically track commits across GitHub & GitLab, link Jira cards, and generate daily reports."
- **CTA Buttons**: "Get Started" (primary), "View Demo" (secondary)
- **Visual**: Abstract code/commit visualization or dashboard preview

### 2. Integrations Section
- **Grid**: 3 cards for GitHub, GitLab, Jira
- **Icon + Name + Brief description**
- **Visual**: Colored icons with brand colors

### 3. Features Section
- **Feature 1**: "Automatic Commit Sync" - pulls from GitHub/GitLab
- **Feature 2**: "Jira Integration" - links commits to cards
- **Feature 3**: "Daily Reports" - automated summaries
- **Feature 4**: "Background Sync" - keeps data fresh

### 4. How It Works Section
- **Step 1**: Connect your accounts (GitHub, GitLab, Jira)
- **Step 2**: Add repositories to track
- **Step 3**: Get daily reports automatically

### 5. CTA Section
- **Headline**: "Start Tracking Today"
- **Button**: "Sign Up Free"

### 6. Footer
- Copyright, links, social

---

## API Mapping

### Endpoint Constants Structure

```typescript
// src/infrastructure/constants/api.constants.ts
export const API_CONSTANTS = {
  BASE_URL: import.meta.env.VITE_API_URL || 'http://localhost:3000',

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
```

### Service Files

| Service File | Functions |
|--------------|-----------|
| `auth.service.ts` | register, login, logout |
| `user.service.ts` | getProfile, updateProfile, validateIntegrations |
| `repo.service.ts` | listRepos, addRepo, deleteRepo, validateRepo |
| `sync.service.ts` | triggerSync, getSyncStatus |
| `commit.service.ts` | listCommits, getMissingCommits, linkToJira |
| `jira.service.ts` | listSprints, getSprint, getActiveSprint, listCards, getCard |
| `report.service.ts` | generateReport, listReports, getReport, deleteReport, createTemplate, listTemplates, updateTemplate, deleteTemplate, previewTemplate |

---

## Frontend Updates Required

1. **index.html**: Change title to "PDT - Personal Daily Tracker"
2. **index.css**: Add PDT color variables
3. **New Landing Page**: `/src/presentation/pages/LandingPage.tsx`
4. **New Route**: Add landing page to routes
5. **API Constants**: Expand with all endpoints
6. **API Services**: Create 7 service files
7. **Update existing pages**: Login page with PDT branding

---

## Acceptance Criteria

- [ ] Landing page displays all 6 sections
- [ ] Colors match PDT branding palette
- [ ] All 20+ API endpoints mapped in constants
- [ ] All 7 service files created with functions
- [ ] Page title updated to "PDT - Personal Daily Tracker"
- [ ] Login page reflects PDT branding
- [ ] Routes configured to show landing page first
