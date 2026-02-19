import * as React from 'react'
import { Link } from 'react-router-dom'
import { Menu, Bell } from 'lucide-react'
import { useDispatch } from 'react-redux'

import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger
} from '@/components/ui/dropdown-menu'
import { useAppSelector } from '@/application/hooks/useAppSelector'
import { logout } from '@/infrastructure/slices/auth/auth.slice'
import { useLogoutMutation } from '@/infrastructure/services/auth.service'

interface HeaderProps {
  className?: string
  onMenuClick?: () => void
  showMenuButton?: boolean
}

export function Header({
  className,
  onMenuClick,
  showMenuButton
}: HeaderProps) {
  const dispatch = useDispatch()
  const [logoutApi] = useLogoutMutation()
  const user = useAppSelector((state) => state.user)

  const userInitials = user?.name
    ? user.name
        .split(' ')
        .map((part: string) => part[0])
        .join('')
        .toUpperCase()
    : 'U'

  const handleLogout = async () => {
    try {
      await logoutApi().unwrap()
      dispatch(logout())
      localStorage.removeItem('auth_token')
    } catch (error) {
      console.error('Error during logout:', error)
      dispatch(logout())
      localStorage.removeItem('auth_token')
    }
  }

  return (
    <header
      className={cn(
        'flex h-16 items-center justify-between border-b border-pdt-background/20 bg-pdt-primary-light px-4 lg:px-6',
        className
      )}
    >
      <div className="flex items-center gap-4">
        {showMenuButton && (
          <Button
            variant="ghost"
            size="icon"
            onClick={onMenuClick}
            className="text-pdt-neutral hover:bg-pdt-primary hover:text-pdt-neutral md:hidden"
          >
            <Menu className="size-5" />
            <span className="sr-only">Toggle menu</span>
          </Button>
        )}

        <Link to="/dashboard" className="flex items-center gap-2">
          <img src="/logo.svg" alt="App logo" className="size-7 sm:size-8" />
          <span className="hidden text-base font-semibold text-pdt-neutral sm:inline-block md:text-lg">
            Dashboard
          </span>
        </Link>
      </div>

      <div className="flex items-center gap-2">
        <Button variant="ghost" size="icon" className="relative text-pdt-neutral hover:bg-pdt-primary hover:text-pdt-neutral">
          <Bell className="size-5" />
          <span className="absolute right-1 top-1 size-2 rounded-full bg-pdt-background" />
          <span className="sr-only">Notifications</span>
        </Button>

        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" className="relative size-8 rounded-full hover:bg-pdt-primary">
              <Avatar className="size-8">
                <AvatarImage src="" alt={user?.name || 'User'} />
                <AvatarFallback className="bg-pdt-background text-pdt-primary">{userInitials}</AvatarFallback>
              </Avatar>
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="w-56 border-pdt-background/20 bg-pdt-primary-light">
            <DropdownMenuLabel className="font-normal">
              <div className="flex flex-col space-y-1">
                <p className="text-sm font-medium leading-none text-pdt-neutral">
                  {user?.name || 'User'}
                </p>
                <p className="text-xs leading-none text-pdt-neutral/60">
                  {user?.email || 'user@example.com'}
                </p>
              </div>
            </DropdownMenuLabel>
            <DropdownMenuSeparator className="bg-pdt-neutral/10" />
            <DropdownMenuItem asChild className="text-pdt-neutral/70 focus:bg-pdt-primary focus:text-pdt-neutral">
              <Link to="/profile">Profile</Link>
            </DropdownMenuItem>
            <DropdownMenuItem asChild className="text-pdt-neutral/70 focus:bg-pdt-primary focus:text-pdt-neutral">
              <Link to="/settings">Settings</Link>
            </DropdownMenuItem>
            <DropdownMenuSeparator className="bg-pdt-neutral/10" />
            <DropdownMenuItem onClick={handleLogout} className="text-pdt-neutral/70 focus:bg-pdt-primary focus:text-pdt-neutral">Logout</DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </header>
  )
}
