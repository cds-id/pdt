import { Navigate, Outlet, useNavigate } from 'react-router-dom'
import { isAuthenticated } from '@/utils/auth'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger
} from '@/components/ui/dropdown-menu'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Button } from '@/components/ui/button'
import { useDispatch } from 'react-redux'
import { logout } from '@/infrastructure/slices/auth/auth.slice'
import { useAppSelector } from '@/application/hooks/useAppSelector'

/**
 * @deprecated Use DashboardLayout instead for dashboard routes.
 * This layout is kept for backward compatibility.
 *
 * Protected layout for authenticated pages
 * Redirects to login if user is not authenticated
 */
const ProtectedLayout = () => {
  const dispatch = useDispatch()
  const navigate = useNavigate()
  const user = useAppSelector((state) => state.user)

  // If user is not authenticated, redirect to login
  if (!isAuthenticated()) {
    return <Navigate to="/login" replace />
  }

  const userInitials = user?.name
    ? user.name
        .split(' ')
        .map((part: string) => part[0])
        .join('')
        .toUpperCase()
    : 'U'

  const handleLogout = () => {
    dispatch(logout())
    localStorage.removeItem('auth_token')
    navigate('/login', { replace: true })
  }

  return (
    <div className="flex min-h-screen flex-col bg-slate-50">
      <header className="bg-white shadow-sm">
        <div className="mx-auto flex max-w-7xl items-center justify-between p-4 sm:px-6 lg:px-8">
          <div className="flex items-center space-x-2">
            <img src="/logo.svg" alt="App logo" className="size-8" />
            <h1 className="text-xl font-bold text-gray-900">Dashboard</h1>
          </div>
          <div className="flex items-center">
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button
                  variant="ghost"
                  className="relative size-8 rounded-full"
                >
                  <Avatar>
                    <AvatarImage alt={user?.name || 'User'} />
                    <AvatarFallback>{userInitials}</AvatarFallback>
                  </Avatar>
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuLabel>My Account</DropdownMenuLabel>
                <DropdownMenuSeparator />
                <DropdownMenuItem>Profile</DropdownMenuItem>
                <DropdownMenuItem>Settings</DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem onClick={handleLogout}>
                  Logout
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </div>
      </header>

      <main className="flex-1 overflow-y-auto p-6">
        <div className="mx-auto max-w-7xl">
          <Outlet />
        </div>
      </main>
    </div>
  )
}

export default ProtectedLayout
