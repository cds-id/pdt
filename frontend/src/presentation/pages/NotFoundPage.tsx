import React from 'react'
import { Link } from 'react-router-dom'

export const NotFoundPage = () => {
  return (
    <div className="flex min-h-screen items-center justify-center">
      <div className="text-center">
        <h1 className="mb-4 text-4xl font-bold">404</h1>
        <p className="mb-4">Page not found</p>
        <Link to="/" className="text-indigo-600 hover:text-indigo-500">
          Go back home
        </Link>
      </div>
    </div>
  )
}
