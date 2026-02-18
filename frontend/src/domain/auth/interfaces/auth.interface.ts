export interface ILoginCredentials {
  email: string
  password: string
}

export interface IAuthState {
  token: string | null
  isAuthenticated: boolean
  error: string | null
}
