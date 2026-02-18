import * as React from 'react'

import type { SidebarContextValue } from './sidebar.types'

const SidebarContext = React.createContext<SidebarContextValue | undefined>(
  undefined
)

const MOBILE_BREAKPOINT = 768

export function SidebarProvider({ children }: { children: React.ReactNode }) {
  const [isCollapsed, setIsCollapsed] = React.useState(false)
  const [isOpen, setIsOpen] = React.useState(false)
  const [isMobile, setIsMobile] = React.useState(false)

  React.useEffect(() => {
    const checkMobile = () => {
      setIsMobile(window.innerWidth < MOBILE_BREAKPOINT)
    }

    checkMobile()
    window.addEventListener('resize', checkMobile)

    return () => window.removeEventListener('resize', checkMobile)
  }, [])

  const toggle = React.useCallback(() => {
    if (isMobile) {
      setIsOpen((prev) => !prev)
    } else {
      setIsCollapsed((prev) => !prev)
    }
  }, [isMobile])

  const value = React.useMemo(
    () => ({
      isCollapsed,
      isMobile,
      isOpen,
      toggle,
      setIsOpen
    }),
    [isCollapsed, isMobile, isOpen, toggle]
  )

  return (
    <SidebarContext.Provider value={value}>{children}</SidebarContext.Provider>
  )
}

export function useSidebar() {
  const context = React.useContext(SidebarContext)
  if (context === undefined) {
    throw new Error('useSidebar must be used within a SidebarProvider')
  }
  return context
}
