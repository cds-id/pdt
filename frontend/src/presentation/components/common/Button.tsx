import React from 'react'
import { classNames } from '@/utils'

interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: 'primary' | 'secondary'
  isLoading?: boolean
}

export const Button = ({
  children,
  variant = 'primary',
  isLoading,
  className,
  ...props
}: ButtonProps) => {
  return (
    <button
      className={classNames(
        'px-4 py-2 rounded-md font-medium transition-colors',
        variant === 'primary' && 'bg-indigo-600 text-white hover:bg-indigo-700',
        variant === 'secondary' &&
          'bg-gray-200 text-gray-800 hover:bg-gray-300',
        isLoading && 'opacity-70 cursor-not-allowed',
        className
      )}
      disabled={isLoading}
      {...props}
    >
      {isLoading ? 'Loading...' : children}
    </button>
  )
}
