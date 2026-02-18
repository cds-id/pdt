import { Navigate, Outlet } from 'react-router-dom'
import { isAuthenticated } from '@/utils/auth'

/**
 * Public layout for landing page and auth pages
 * Full-width structure for landing page
 */
const PublicLayout = () => {
  // If user is authenticated, redirect to dashboard
  if (isAuthenticated()) {
    return <Navigate to="/dashboard" replace />
  }

  return (
    <div className="min-h-screen bg-[#F8C630]">
      <Outlet />
    </div>
  )
}

export default PublicLayout
