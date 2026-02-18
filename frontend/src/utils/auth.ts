/**
 * Authentication utilities for token validation and redirection
 */

/**
 * Check if the user's auth token exists and is not expired
 * @returns boolean indicating if the user is authenticated
 */
export const isAuthenticated = (): boolean => {
  // Get token from localStorage
  const token = localStorage.getItem('auth_token') || ''

  if (!token) return false

  try {
    // Simple JWT token structure checking
    const tokenParts = token.split('.')
    if (tokenParts.length !== 3) return false

    // Decode the payload
    const payload = JSON.parse(atob(tokenParts[1]))

    // Check if token is expired
    const expiry = payload.exp * 1000 // Convert to milliseconds
    const now = Date.now()

    return expiry > now
  } catch (error) {
    console.error('Error validating token:', error)
    return false
  }
}

/**
 * Get user information from token
 * @returns User object or null if not authenticated
 */
export const getUserFromToken = () => {
  if (!isAuthenticated()) return null

  const token = localStorage.getItem('auth_token') || ''
  try {
    const payload = JSON.parse(atob(token.split('.')[1]))
    return {
      id: payload.sub,
      name: payload.name || 'User',
      email: payload.email || ''
      // Add other user properties as needed
    }
  } catch (error) {
    console.error('Error extracting user data from token:', error)
    return null
  }
}
