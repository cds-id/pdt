# PDT API Documentation & README Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Create comprehensive API documentation, architecture overview, setup guide, and README for the PDT backend.

**Architecture:** Modular markdown docs in `docs/api/` per domain, plus `docs/setup.md`, `docs/architecture.md`, and `backend/README.md` as the hub.

**Tech Stack:** Markdown (GitHub-flavored)

---

### Task 1: Security — Add credentials file to .gitignore

**Files:**
- Modify: `.gitignore`

**Step 1:** Add `pdt_credentials_*.txt` pattern to `.gitignore`

**Step 2:** Verify the credentials file is now ignored: `git status`

**Step 3: Commit**
```bash
git add .gitignore
git commit -m "chore: add credentials file pattern to .gitignore"
```

---

### Task 2: Write docs/setup.md

**Files:**
- Create: `docs/setup.md`

Content covers: prerequisites, database setup, environment variables table, token acquisition guides (GitHub PAT, GitLab PAT, Jira API token), R2 setup (optional), running the server.

**Step 1:** Write `docs/setup.md` with full onboarding content

**Step 2: Commit**
```bash
git add docs/setup.md
git commit -m "docs: add setup and onboarding guide"
```

---

### Task 3: Write docs/architecture.md

**Files:**
- Create: `docs/architecture.md`

Content covers: system overview ASCII diagram, data flow, background worker flow, security model, database ERD, integration points.

**Step 1:** Write `docs/architecture.md`

**Step 2: Commit**
```bash
git add docs/architecture.md
git commit -m "docs: add architecture overview"
```

---

### Task 4: Write API docs (all 7 domain files)

**Files:**
- Create: `docs/api/auth.md`
- Create: `docs/api/user.md`
- Create: `docs/api/repositories.md`
- Create: `docs/api/sync.md`
- Create: `docs/api/commits.md`
- Create: `docs/api/jira.md`
- Create: `docs/api/reports.md`

Each file follows the standard format: endpoint, method, headers, request body, response, error codes.

**Step 1:** Write all 7 API doc files

**Step 2: Commit**
```bash
git add docs/api/
git commit -m "docs: add complete API reference for all endpoints"
```

---

### Task 5: Write backend/README.md

**Files:**
- Create: `backend/README.md`

Content: project name, description, tech stack table, quick start, links to all docs, project structure tree.

**Step 1:** Write `backend/README.md`

**Step 2: Commit**
```bash
git add backend/README.md
git commit -m "docs: add backend README with project overview"
```

---

### Task 6: Security review and final commit

**Step 1:** Review all new files for sensitive data (no tokens, passwords, real emails)

**Step 2:** Run `git diff --cached` and `git status` to verify nothing sensitive is staged

**Step 3:** Verify `pdt_credentials_20260218_230441.txt` is NOT tracked
