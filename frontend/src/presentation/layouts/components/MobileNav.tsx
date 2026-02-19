import * as React from 'react'
import { X } from 'lucide-react'

import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { SidebarNav } from '@/presentation/components/dashboard/Sidebar'

interface MobileNavProps {
  isOpen: boolean
  onClose: () => void
}

export function MobileNav({ isOpen, onClose }: MobileNavProps) {
  React.useEffect(() => {
    if (isOpen) {
      document.body.style.overflow = 'hidden'
    } else {
      document.body.style.overflow = 'unset'
    }

    return () => {
      document.body.style.overflow = 'unset'
    }
  }, [isOpen])

  return (
    <>
      {/* Backdrop */}
      <div
        className={cn(
          'fixed inset-0 z-40 bg-pdt-primary/80 backdrop-blur-sm transition-opacity md:hidden',
          isOpen ? 'opacity-100' : 'pointer-events-none opacity-0'
        )}
        onClick={onClose}
      />

      {/* Drawer */}
      <aside
        className={cn(
          'fixed inset-y-0 left-0 z-50 flex w-[280px] max-w-[85vw] flex-col border-r border-pdt-background/20 bg-pdt-primary transition-transform duration-300 md:hidden',
          isOpen ? 'translate-x-0' : '-translate-x-full'
        )}
      >
        <div className="flex h-14 shrink-0 items-center justify-between border-b border-pdt-background/20 px-3">
          <div className="flex items-center gap-2">
            <img src="/logo.svg" alt="Logo" className="size-7" />
            <span className="text-base font-semibold text-pdt-neutral">Menu</span>
          </div>
          <Button variant="ghost" size="icon" onClick={onClose} className="text-pdt-neutral hover:bg-pdt-primary-light hover:text-pdt-neutral">
            <X className="size-5" />
            <span className="sr-only">Close menu</span>
          </Button>
        </div>

        <div className="flex-1 overflow-y-auto px-2 py-3 scrollbar-none">
          <SidebarNav />
        </div>
      </aside>
    </>
  )
}
