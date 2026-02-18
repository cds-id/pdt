import { Navigate, Outlet } from 'react-router-dom'
import { isAuthenticated } from '@/utils/auth'

/**
 * Public layout for non-authenticated pages
 * Redirects to dashboard if user is already authenticated
 */
const PublicLayout = () => {
  // If user is authenticated, redirect to dashboard
  if (isAuthenticated()) {
    return <Navigate to="/dashboard" replace />
  }

  return (
    <div className="flex min-h-screen flex-col justify-center bg-slate-50 py-12 sm:px-6 lg:px-8">
      <div className="sm:mx-auto sm:w-full sm:max-w-md">
        <h2 className="text-center text-3xl font-bold tracking-tight text-gray-900">
          <img className="mx-auto h-12 w-auto" src="/logo.svg" alt="App logo" />
        </h2>
      </div>
      <Outlet />
    </div>
  )
}

export default PublicLayout
