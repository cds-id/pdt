import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useNavigate, Link } from 'react-router-dom'
import { Eye, EyeOff } from 'lucide-react'

import { useRegisterMutation } from '@/infrastructure/services/auth.service'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

interface RegisterFormData {
  name: string
  email: string
  password: string
}

export function RegisterPage() {
  const navigate = useNavigate()
  const [registerMutation, { isLoading, error }] = useRegisterMutation()
  const [showPassword, setShowPassword] = useState(false)

  const {
    register,
    handleSubmit,
    formState: { errors }
  } = useForm<RegisterFormData>()

  const onSubmit = async (data: RegisterFormData) => {
    try {
      await registerMutation(data).unwrap()
      navigate('/dashboard/home')
    } catch {
      // Error handled by RTK Query onQueryStarted
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-pdt-background px-4">
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
          <h2 className="mb-1 text-center text-2xl font-bold text-pdt-neutral">
            Create your account
          </h2>
          <p className="mb-6 text-center text-sm text-pdt-neutral/60">
            Start tracking your development progress
          </p>

          {error && (
            <div className="mb-4 rounded-lg bg-red-500/10 p-3 text-sm text-red-400">
              Registration failed. Please try again.
            </div>
          )}

          <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="name" className="text-pdt-neutral/70">Name</Label>
              <Input
                {...register('name', { required: 'Name is required' })}
                id="name"
                type="text"
                placeholder="John Doe"
                autoComplete="name"
                className="border-pdt-neutral/20 bg-pdt-primary-light text-pdt-neutral placeholder:text-pdt-neutral/30 focus:border-pdt-background focus:ring-pdt-background"
              />
              {errors.name && <p className="text-sm text-red-400">{errors.name.message}</p>}
            </div>

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
                className="border-pdt-neutral/20 bg-pdt-primary-light text-pdt-neutral placeholder:text-pdt-neutral/30 focus:border-pdt-background focus:ring-pdt-background"
              />
              {errors.email && <p className="text-sm text-red-400">{errors.email.message}</p>}
            </div>

            <div className="space-y-2">
              <Label htmlFor="password" className="text-pdt-neutral/70">Password</Label>
              <div className="relative">
                <Input
                  {...register('password', {
                    required: 'Password is required',
                    minLength: { value: 8, message: 'Password must be at least 8 characters' }
                  })}
                  id="password"
                  type={showPassword ? 'text' : 'password'}
                  placeholder="Min. 8 characters"
                  autoComplete="new-password"
                  className="border-pdt-neutral/20 bg-pdt-primary-light pr-10 text-pdt-neutral placeholder:text-pdt-neutral/30 focus:border-pdt-background focus:ring-pdt-background"
                />
                <button
                  type="button"
                  onClick={() => setShowPassword(!showPassword)}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-pdt-neutral/40 hover:text-pdt-neutral/70"
                >
                  {showPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                </button>
              </div>
              {errors.password && <p className="text-sm text-red-400">{errors.password.message}</p>}
            </div>

            <Button
              type="submit"
              disabled={isLoading}
              className="w-full bg-pdt-background text-pdt-primary font-semibold hover:bg-pdt-background/90"
            >
              {isLoading ? (
                <>
                  <svg className="mr-2 h-4 w-4 animate-spin" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                    <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
                  </svg>
                  Creating account...
                </>
              ) : (
                'Create Account'
              )}
            </Button>
          </form>

          <p className="mt-6 text-center text-sm text-pdt-neutral/60">
            Already have an account?{' '}
            <Link to="/login" className="font-medium text-pdt-background underline-offset-4 hover:underline">
              Sign in
            </Link>
          </p>
        </div>
      </div>
    </div>
  )
}
