import { useState, useEffect } from 'react'
import { useAuth0 } from '@auth0/auth0-react'

export interface WatchProvider {
  name: string
  logoPath?: string
  providerType: 'flatrate' | 'rent' | 'buy' | 'free' | 'plex'
  price?: string
  link?: string
  plexServer?: string
}

export interface WatchProvidersData {
  tmdbId: number
  region: string
  tmdbLink?: string
  providers: WatchProvider[]
  plexAvailable: boolean
  cachedAt: string
  expiresAt: string
}

export function useWatchProviders(tmdbId: number | null, region: string = 'US') {
  const { getAccessTokenSilently } = useAuth0()
  const [watchProviders, setWatchProviders] = useState<WatchProvidersData | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const fetchWatchProviders = async () => {
    if (!tmdbId) return

    setLoading(true)
    setError(null)

    try {
      // Try to get auth token, but don't fail if user is not authenticated
      let token = null
      try {
        token = await getAccessTokenSilently()
      } catch (authError) {
        // User not authenticated, continue without token
        console.log('User not authenticated, fetching providers without Plex data')
      }

      const headers: HeadersInit = {
        'Content-Type': 'application/json',
      }
      
      if (token) {
        headers['Authorization'] = `Bearer ${token}`
      }

      const url = new URL(`/api/movies/${tmdbId}/watch-providers`, window.location.origin)
      if (region && region !== 'US') {
        url.searchParams.set('region', region)
      }

      const response = await fetch(url.toString(), {
        headers,
      })

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`)
      }

      const data: WatchProvidersData = await response.json()
      setWatchProviders(data)
    } catch (err) {
      console.error('Failed to fetch watch providers:', err)
      setError(err instanceof Error ? err.message : 'Failed to fetch watch providers')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchWatchProviders()
  }, [tmdbId, region])

  return {
    watchProviders,
    loading,
    error,
    refetch: fetchWatchProviders,
  }
}