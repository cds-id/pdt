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
