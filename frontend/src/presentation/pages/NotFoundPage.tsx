import { Link } from 'react-router-dom'
import { Button } from '@/components/ui/button'

export const NotFoundPage = () => {
  return (
    <div className="flex min-h-screen items-center justify-center bg-pdt-primary px-4">
      <div className="text-center">
        <h1 className="mb-2 text-6xl font-bold text-pdt-accent">404</h1>
        <p className="mb-6 text-lg text-pdt-neutral/60">Page not found</p>
        <Button asChild variant="pdt">
          <Link to="/">Go back home</Link>
        </Button>
      </div>
    </div>
  )
}
