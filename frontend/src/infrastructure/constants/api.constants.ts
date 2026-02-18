/**
 * API service constants - All backend endpoints
 */
export const API_CONSTANTS = {
  BASE_URL: import.meta.env.VITE_API_URL || 'http://localhost:8080',
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
