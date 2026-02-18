import { useForm } from 'react-hook-form'
import { useNavigate, Link } from 'react-router-dom'
import { ILoginCredentials } from '@/domain/auth/interfaces/auth.interface'
import { useLoginMutation } from '@/infrastructure/services/auth.service'

// Shadcn UI Components
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle
} from '@/components/ui/card'

export const LoginPage = () => {
  const navigate = useNavigate()
  const [loginMutation, { isLoading }] = useLoginMutation()
  const {
    register,
    handleSubmit,
    formState: { errors }
  } = useForm<ILoginCredentials>()

  const onSubmit = async (data: ILoginCredentials) => {
    try {
      await loginMutation(data).unwrap()
      navigate('/dashboard')
    } catch {
      // Error is handled by RTK Query onQueryStarted
    }
  }

  return (
    <div className="mx-auto max-w-md space-y-6 p-6">
      {/* PDT Branding Header */}
      <div className="text-center mb-4">
        <Link to="/" className="inline-flex items-center gap-2">
          <span className="text-3xl font-bold text-pdt-primary">PDT</span>
        </Link>
        <p className="text-sm text-slate-500 mt-1">Personal Development Tracker</p>
      </div>

      <Card className="border-slate-200 shadow-lg">
        <CardHeader className="space-y-1">
          <CardTitle className="text-center text-2xl font-bold text-pdt-primary">
            Welcome Back
          </CardTitle>
          <CardDescription className="text-center">
            Enter your credentials to access your account
          </CardDescription>
        </CardHeader>

        <CardContent>
          <form className="space-y-4" onSubmit={handleSubmit(onSubmit)}>
            <div className="space-y-2">
              <Label htmlFor="email">Email</Label>
              <Input
                {...register('email', {
                  required: 'Email is required',
                  pattern: {
                    value: /^[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}$/i,
                    message: 'Invalid email address'
                  }
                })}
                id="email"
                type="email"
                placeholder="example@domain.com"
                autoComplete="email"
              />
              {errors.email && (
                <p className="text-sm font-medium text-destructive">
                  {errors.email.message}
                </p>
              )}
            </div>

            <div className="space-y-2">
              <div className="flex items-center justify-between">
                <Label htmlFor="password">Password</Label>
                <button
                  type="button"
                  className="text-sm text-primary underline-offset-4 hover:underline"
                >
                  Forgot password?
                </button>
              </div>
              <Input
                {...register('password', {
                  required: 'Password is required'
                })}
                id="password"
                type="password"
                autoComplete="current-password"
              />
              {errors.password && (
                <p className="text-sm font-medium text-destructive">
                  {errors.password.message}
                </p>
              )}
            </div>

            <Button className="w-full bg-pdt-primary hover:bg-pdt-primary-light" type="submit" disabled={isLoading}>
              {isLoading ? (
                <>
                  <svg
                    className="mr-2 size-4 animate-spin"
                    xmlns="http://www.w3.org/2000/svg"
                    fill="none"
                    viewBox="0 0 24 24"
                  >
                    <circle
                      className="opacity-25"
                      cx="12"
                      cy="12"
                      r="10"
                      stroke="currentColor"
                      strokeWidth="4"
                    ></circle>
                    <path
                      className="opacity-75"
                      fill="currentColor"
                      d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                    ></path>
                  </svg>
                  Processing...
                </>
              ) : (
                'Sign in'
              )}
            </Button>
          </form>
        </CardContent>

        <CardFooter className="flex justify-center">
          <p className="text-sm text-muted-foreground">
            Don&apos;t have an account?{' '}
            <Link to="/register" className="text-sm text-pdt-accent hover:text-pdt-accent-hover font-medium underline-offset-4 hover:underline">
              Sign up
            </Link>
          </p>
        </CardFooter>
      </Card>
    </div>
  )
}
