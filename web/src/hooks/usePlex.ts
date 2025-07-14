import { useState, useEffect, useCallback } from 'react'
import { useAuth0 } from '@auth0/auth0-react'

interface PlexUser {
  username: string
  email: string
  thumb: string
  serverCount: number
}

interface PlexStatus {
  connected: boolean
  username?: string
  friendlyName?: string
  email?: string
  thumb?: string
  serverCount?: number
  connectedAt?: string
}

interface PlexPinData {
  pinId: number
  pinCode: string
  expiresAt: string
}

interface PlexAuthResult {
  authorized: boolean
  expiresAt?: string
  user?: PlexUser
}

export function usePlex() {
  const { getAccessTokenSilently } = useAuth0()
  const [status, setStatus] = useState<PlexStatus>({ connected: false })
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // Fetch Plex status
  const fetchStatus = useCallback(async () => {
    try {
      setLoading(true)
      setError(null)
      
      const token = await getAccessTokenSilently()
      const response = await fetch('/api/plex/status', {
        headers: {
          'Authorization': `Bearer ${token}`,
        },
      })

      if (!response.ok) {
        throw new Error('Failed to fetch Plex status')
      }

      const data: PlexStatus = await response.json()
      setStatus(data)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error')
      setStatus({ connected: false })
    } finally {
      setLoading(false)
    }
  }, [getAccessTokenSilently])

  // Start Plex authentication flow
  const startAuth = useCallback(async (): Promise<PlexPinData> => {
    try {
      setError(null)
      
      const token = await getAccessTokenSilently()
      const response = await fetch('/api/plex/auth/start', {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${token}`,
        },
      })

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}))
        throw new Error(errorData.error || 'Failed to start Plex authentication')
      }

      return await response.json()
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error'
      setError(errorMessage)
      throw new Error(errorMessage)
    }
  }, [getAccessTokenSilently])

  // Check if PIN has been authorized
  const checkAuth = useCallback(async (pinId: number): Promise<PlexAuthResult> => {
    try {
      setError(null)
      
      const token = await getAccessTokenSilently()
      const response = await fetch(`/api/plex/auth/check?pinId=${pinId}`, {
        headers: {
          'Authorization': `Bearer ${token}`,
        },
      })

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}))
        throw new Error(errorData.error || 'Failed to check authentication')
      }

      const result: PlexAuthResult = await response.json()
      
      // If authorized, refresh status
      if (result.authorized) {
        await fetchStatus()
      }
      
      return result
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error'
      setError(errorMessage)
      throw new Error(errorMessage)
    }
  }, [getAccessTokenSilently, fetchStatus])

  // Disconnect Plex account
  const disconnect = useCallback(async () => {
    try {
      setError(null)
      
      const token = await getAccessTokenSilently()
      const response = await fetch('/api/plex/disconnect', {
        method: 'DELETE',
        headers: {
          'Authorization': `Bearer ${token}`,
        },
      })

      if (!response.ok) {
        throw new Error('Failed to disconnect Plex account')
      }

      // Update status to disconnected
      setStatus({ connected: false })
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error'
      setError(errorMessage)
      throw new Error(errorMessage)
    }
  }, [getAccessTokenSilently])

  // Poll for PIN authorization
  const pollForAuth = useCallback(async (
    pinId: number,
    onUpdate?: (result: PlexAuthResult) => void,
    intervalMs: number = 2000,
    maxAttempts: number = 30
  ): Promise<PlexAuthResult> => {
    let attempts = 0
    
    return new Promise((resolve, reject) => {
      const interval = setInterval(async () => {
        try {
          attempts++
          const result = await checkAuth(pinId)
          
          onUpdate?.(result)
          
          if (result.authorized) {
            clearInterval(interval)
            resolve(result)
          } else if (attempts >= maxAttempts) {
            clearInterval(interval)
            reject(new Error('Authentication timeout'))
          }
        } catch (err) {
          clearInterval(interval)
          reject(err)
        }
      }, intervalMs)
    })
  }, [checkAuth])

  // Load status on mount
  useEffect(() => {
    fetchStatus()
  }, [fetchStatus])

  return {
    status,
    loading,
    error,
    startAuth,
    checkAuth,
    disconnect,
    pollForAuth,
    refreshStatus: fetchStatus,
  }
}