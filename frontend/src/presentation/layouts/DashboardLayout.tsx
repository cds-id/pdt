import * as React from 'react'
import { Navigate, Outlet, useLocation } from 'react-router-dom'

import { isAuthenticated } from '@/utils/auth'
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator
} from '@/components/ui/breadcrumb'
import { getBreadcrumbsForPath } from '@/config/navigation'

import { TooltipProvider } from '@/components/ui/tooltip'

import {
  Sidebar,
  SidebarProvider,
  useSidebar
} from '../components/dashboard/Sidebar'
import { Header, MobileNav } from './components'

function DashboardLayoutContent() {
  const { isOpen, setIsOpen, isMobile } = useSidebar()
  const location = useLocation()
  const breadcrumbs = getBreadcrumbsForPath(location.pathname)

  return (
    <div className="flex h-screen w-full overflow-hidden bg-pdt-primary">
      {/* Desktop Sidebar */}
      <Sidebar />

      {/* Mobile Navigation */}
      <MobileNav isOpen={isOpen} onClose={() => setIsOpen(false)} />

      {/* Main Content */}
      <div className="flex min-w-0 flex-1 flex-col overflow-hidden">
        <Header showMenuButton={isMobile} onMenuClick={() => setIsOpen(true)} />

        <main className="flex-1 overflow-y-auto overflow-x-hidden bg-pdt-primary p-3 sm:p-4 lg:p-6">
          {/* Breadcrumbs */}
          {breadcrumbs.length > 1 && (
            <Breadcrumb className="mb-4">
              <BreadcrumbList>
                {breadcrumbs.map((crumb, index) => (
                  <React.Fragment key={crumb.href}>
                    <BreadcrumbItem>
                      {index === breadcrumbs.length - 1 ? (
                        <BreadcrumbPage className="text-pdt-neutral">{crumb.title}</BreadcrumbPage>
                      ) : (
                        <BreadcrumbLink href={crumb.href} className="text-pdt-neutral/60 hover:text-pdt-accent">
                          {crumb.title}
                        </BreadcrumbLink>
                      )}
                    </BreadcrumbItem>
                    {index < breadcrumbs.length - 1 && <BreadcrumbSeparator />}
                  </React.Fragment>
                ))}
              </BreadcrumbList>
            </Breadcrumb>
          )}

          <Outlet />
        </main>
      </div>
    </div>
  )
}

/**
 * Dashboard layout with sidebar navigation
 * Provides collapsible sidebar, mobile drawer, and breadcrumbs
 */
export function DashboardLayout() {
  if (!isAuthenticated()) {
    return <Navigate to="/login" replace />
  }

  return (
    <TooltipProvider>
      <SidebarProvider>
        <DashboardLayoutContent />
      </SidebarProvider>
    </TooltipProvider>
  )
}

export default DashboardLayout
