import { createBrowserRouter, Navigate } from 'react-router-dom'
import PublicLayout from '../layouts/PublicLayout'
import { DashboardLayout } from '../layouts/DashboardLayout'
import { LoginPage } from '../pages/auth/LoginPage'
import { DashboardPage } from '../pages/dashboard/DashboardPage'
import { NotFoundPage } from '../pages/NotFoundPage'
import { LandingPage } from '../pages/LandingPage'

export const router = createBrowserRouter([
  // Public routes (landing, login, register)
  {
    path: '/',
    element: <PublicLayout />,
    errorElement: <NotFoundPage />,
    children: [
      {
        path: '/',
        element: <LandingPage />
      },
      {
        path: 'login',
        element: <LoginPage />
      },
      {
        path: 'register',
        element: <LoginPage />
      }
    ]
  },
  // Protected routes with DashboardLayout (sidebar + header)
  {
    path: '/',
    element: <DashboardLayout />,
    children: [
      {
        path: 'dashboard',
        element: <DashboardPage />
      },
      {
        path: 'analytics',
        element: <div className="p-6">Analytics Page Coming Soon</div>
      },
      {
        path: 'users',
        element: <div className="p-6">Users Management Coming Soon</div>
      },
      {
        path: 'reports',
        element: <div className="p-6">Reports Page Coming Soon</div>
      },
      {
        path: 'profile',
        element: <div className="p-6">Profile Page Coming Soon</div>
      },
      {
        path: 'settings',
        element: <div className="p-6">Settings Page Coming Soon</div>
      },
      {
        path: 'help',
        element: <div className="p-6">Help & Support Coming Soon</div>
      }
    ]
  },
  // Fallback route
  {
    path: '*',
    element: <NotFoundPage />
  }
])
