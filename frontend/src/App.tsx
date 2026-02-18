import { RouterProvider } from 'react-router-dom'
import { router } from '@/presentation/routes'
import { ErrorBoundary } from '@/presentation/components/common/ErrorBoundary'

function App() {
  return (
    <ErrorBoundary>
      <RouterProvider router={router} />
    </ErrorBoundary>
  )
}

export default App
