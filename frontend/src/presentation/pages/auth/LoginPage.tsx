import { useForm } from 'react-hook-form'
import { useNavigate, Link } from 'react-router-dom'
import { ILoginCredentials } from '@/domain/auth/interfaces/auth.interface'
import { useLoginMutation } from '@/infrastructure/services/auth.service'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

export const LoginPage = () => {
  const navigate = useNavigate()
  const [loginMutation, { isLoading, error }] = useLoginMutation()
  const {
    register,
    handleSubmit,
    formState: { errors }
  } = useForm<ILoginCredentials>()

  const onSubmit = async (data: ILoginCredentials) => {
    try {
      await loginMutation(data).unwrap()
      navigate('/dashboard/home')
    } catch {
      // Error handled by RTK Query onQueryStarted
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-pdt-accent px-4">
      <div className="w-full max-w-md">
        <div className="mb-8 text-center">
          <Link to="/" className="inline-flex items-center gap-2">
            <svg width="40" height="40" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
              <circle cx="12" cy="12" r="10" stroke="#1B1B1E" strokeWidth="2"/>
              <path d="M7 14L10 11L13 13L17 8" stroke="#1B1B1E" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round"/>
              <circle cx="17" cy="8" r="1.5" fill="#1B1B1E"/>
            </svg>
            <span className="text-3xl font-bold text-pdt-primary">PDT</span>
          </Link>
          <p className="mt-2 text-pdt-primary/70">Personal Development Tracker</p>
        </div>

        <div className="rounded-xl border-2 border-pdt-primary/10 bg-pdt-primary p-6 shadow-xl sm:p-8">
          <h2 className="mb-1 text-center text-2xl font-bold text-pdt-neutral">Welcome Back</h2>
          <p className="mb-6 text-center text-sm text-pdt-neutral/60">
            Enter your credentials to access your account
          </p>

          {error && (
            <div className="mb-4 rounded-lg bg-red-500/10 p-3 text-sm text-red-400">
              Login failed. Please check your credentials.
            </div>
          )}

          <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="email" className="text-pdt-neutral/70">Email</Label>
              <Input
                {...register('email', {
                  required: 'Email is required',
                  pattern: { value: /^[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}$/i, message: 'Invalid email address' }
                })}
                id="email"
                type="email"
                placeholder="you@example.com"
                autoComplete="email"
                className="border-pdt-neutral/20 bg-pdt-primary-light text-pdt-neutral placeholder:text-pdt-neutral/30 focus:border-pdt-accent focus:ring-pdt-accent"
              />
              {errors.email && <p className="text-sm text-red-400">{errors.email.message}</p>}
            </div>

            <div className="space-y-2">
              <Label htmlFor="password" className="text-pdt-neutral/70">Password</Label>
              <Input
                {...register('password', { required: 'Password is required' })}
                id="password"
                type="password"
                autoComplete="current-password"
                className="border-pdt-neutral/20 bg-pdt-primary-light text-pdt-neutral placeholder:text-pdt-neutral/30 focus:border-pdt-accent focus:ring-pdt-accent"
              />
              {errors.password && <p className="text-sm text-red-400">{errors.password.message}</p>}
            </div>

            <Button
              type="submit"
              disabled={isLoading}
              className="w-full bg-pdt-accent text-pdt-primary font-semibold hover:bg-pdt-accent/90"
            >
              {isLoading ? (
                <>
                  <svg className="mr-2 h-4 w-4 animate-spin" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                    <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
                  </svg>
                  Signing in...
                </>
              ) : (
                'Sign in'
              )}
            </Button>
          </form>

          <p className="mt-6 text-center text-sm text-pdt-neutral/60">
            Don&apos;t have an account?{' '}
            <Link to="/register" className="font-medium text-pdt-accent underline-offset-4 hover:underline">
              Sign up
            </Link>
          </p>
        </div>
      </div>
    </div>
  )
}
