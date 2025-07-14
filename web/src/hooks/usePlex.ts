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

interface SyncJob {
  job_id: number
  status: string
  progress: number
  current_step: string
  total_items: number
  processed_items: number
  successful_items: number
  failed_items: number
  error_message?: string
  started_at?: string
  completed_at?: string
  created_at: string
}

interface SyncStatus {
  isActive: boolean
  currentJob?: SyncJob
  lastSync?: string
  error?: string
}

export function usePlex() {
  const { getAccessTokenSilently } = useAuth0()
  const [status, setStatus] = useState<PlexStatus>({ connected: false })
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [syncStatus, setSyncStatus] = useState<SyncStatus>({ isActive: false })
  const [syncLoading, setSyncLoading] = useState(false)

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

  // Trigger full sync
  const triggerSync = useCallback(async (): Promise<SyncJob> => {
    setSyncLoading(true)
    setSyncStatus({ isActive: false, error: undefined })
    
    try {
      const token = await getAccessTokenSilently()
      const response = await fetch('/api/plex/sync/enhanced', {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
      })

      if (!response.ok) {
        const errorText = await response.text()
        throw new Error(`Sync failed: ${errorText}`)
      }

      const result = await response.json()
      console.log('Sync response:', result) // Debug logging
      
      // Ensure we have a valid job_id
      if (!result.job_id) {
        throw new Error('No job ID returned from sync request')
      }
      
      // Start polling for job status
      const job = await pollJobStatus(result.job_id)
      
      setSyncStatus({
        isActive: true,
        currentJob: job,
      })
      
      return job
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error'
      setSyncStatus({
        isActive: false,
        error: errorMessage,
      })
      throw err
    } finally {
      setSyncLoading(false)
    }
  }, [getAccessTokenSilently])

  // Poll job status
  const pollJobStatus = useCallback(async (jobId: number): Promise<SyncJob> => {
    const token = await getAccessTokenSilently()
    const response = await fetch(`/api/plex/sync/status/${jobId}`, {
      headers: {
        'Authorization': `Bearer ${token}`,
      },
    })

    if (!response.ok) {
      throw new Error('Failed to get job status')
    }

    const job: SyncJob = await response.json()
    
    // Update sync status
    setSyncStatus({
      isActive: job.status === 'pending' || job.status === 'running',
      currentJob: job,
      lastSync: job.status === 'completed' ? job.completed_at : undefined,
      error: job.status === 'failed' ? job.error_message : undefined,
    })
    
    return job
  }, [getAccessTokenSilently])

  // Start job polling when sync is active
  useEffect(() => {
    if (!syncStatus.isActive || !syncStatus.currentJob) return
    
    const pollInterval = setInterval(async () => {
      try {
        const job = await pollJobStatus(syncStatus.currentJob!.job_id)
        
        // Stop polling if job is completed
        if (job.status === 'completed' || job.status === 'failed' || job.status === 'cancelled') {
          clearInterval(pollInterval)
        }
      } catch (err) {
        console.error('Failed to poll job status:', err)
        clearInterval(pollInterval)
      }
    }, 2000) // Poll every 2 seconds

    return () => clearInterval(pollInterval)
  }, [syncStatus.isActive, syncStatus.currentJob?.job_id, pollJobStatus])

  // Cancel sync job
  const cancelSync = useCallback(async () => {
    if (!syncStatus.currentJob) return
    
    try {
      const token = await getAccessTokenSilently()
      await fetch(`/api/plex/sync/cancel/${syncStatus.currentJob.job_id}`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${token}`,
        },
      })
      
      setSyncStatus({
        isActive: false,
        error: undefined,
      })
    } catch (err) {
      console.error('Failed to cancel sync:', err)
    }
  }, [getAccessTokenSilently, syncStatus.currentJob])

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
    // Sync functionality
    syncStatus,
    syncLoading,
    triggerSync,
    cancelSync,
  }
}