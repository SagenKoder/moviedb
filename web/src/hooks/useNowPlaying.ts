import { useState, useEffect, useCallback } from 'react'
import { useAuth0 } from '@auth0/auth0-react'

interface NowPlayingItem {
  ratingKey: string
  title: string
  year: number
  summary: string
  thumb: string
  duration: number
  viewOffset: number
  playerState: string
  progress: number
  tmdbId?: number
  localMovie?: {
    title: string
    year?: number
    posterUrl?: string
    synopsis: string
  }
}

interface NowPlayingResponse {
  nowPlaying: NowPlayingItem[]
  connected: boolean
  count: number
  error?: string
}

export function useNowPlaying() {
  const { getAccessTokenSilently } = useAuth0()
  const [nowPlaying, setNowPlaying] = useState<NowPlayingItem[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [connected, setConnected] = useState(false)

  // Fetch now playing data
  const fetchNowPlaying = useCallback(async () => {
    try {
      setError(null)
      
      const token = await getAccessTokenSilently()
      const response = await fetch('/api/plex/now-playing', {
        headers: {
          'Authorization': `Bearer ${token}`,
        },
      })

      if (!response.ok) {
        throw new Error('Failed to fetch now playing')
      }

      const data: NowPlayingResponse = await response.json()
      setNowPlaying(data.nowPlaying || [])
      setConnected(data.connected)
      
      if (data.error) {
        setError(data.error)
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error')
      setNowPlaying([])
      setConnected(false)
    } finally {
      setLoading(false)
    }
  }, [getAccessTokenSilently])

  // Auto-refresh every 10 seconds
  useEffect(() => {
    // Initial fetch
    fetchNowPlaying()
    
    // Set up interval for updates
    const interval = setInterval(() => {
      fetchNowPlaying()
    }, 10000) // 10 seconds
    
    return () => clearInterval(interval)
  }, [fetchNowPlaying])

  // Get the first/primary item being watched
  const primaryNowPlaying = nowPlaying.length > 0 ? nowPlaying[0] : null

  return {
    nowPlaying,
    primaryNowPlaying,
    loading,
    error,
    connected,
    refresh: fetchNowPlaying,
  }
}