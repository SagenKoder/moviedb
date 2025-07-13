import { useState } from 'react'
import { useAuth0 } from '@auth0/auth0-react'

export interface Movie {
  id: number
  tmdb_id: number
  title: string
  year?: number
  poster_url?: string
  synopsis?: string
  runtime?: number
  genres?: string | string[]
  vote_avg?: number
  vote_count?: number
  tagline?: string
  status?: string
  backdrop_url?: string
  external_ids?: {
    imdb_id?: string
  }
}

export interface MovieSearchResponse {
  results: Movie[]
  page: number
  total_pages?: number
  total_results?: number
}

export function useMovies() {
  const { getAccessTokenSilently } = useAuth0()
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const apiCall = async (url: string, options: RequestInit = {}) => {
    const token = await getAccessTokenSilently()
    
    const response = await fetch(url, {
      ...options,
      headers: {
        ...options.headers,
        Authorization: `Bearer ${token}`,
        'Content-Type': 'application/json',
      },
    })

    if (!response.ok) {
      // Try to get error message from response body
      try {
        const errorText = await response.text()
        throw new Error(errorText || `HTTP error! status: ${response.status}`)
      } catch {
        throw new Error(`HTTP error! status: ${response.status}`)
      }
    }

    return response.json()
  }

  const searchMovies = async (query: string = '', page: number = 1): Promise<MovieSearchResponse> => {
    setLoading(true)
    setError(null)

    try {
      const params = new URLSearchParams()
      if (query) params.append('search', query)
      params.append('page', page.toString())

      const data = await apiCall(`/api/movies?${params.toString()}`)
      return data
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to search movies'
      setError(errorMessage)
      throw err
    } finally {
      setLoading(false)
    }
  }

  const getMovieDetails = async (movieId: number): Promise<Movie> => {
    setLoading(true)
    setError(null)

    try {
      const data = await apiCall(`/api/movies/${movieId}`)
      return data
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to get movie details'
      setError(errorMessage)
      throw err
    } finally {
      setLoading(false)
    }
  }

  return {
    loading,
    error,
    searchMovies,
    getMovieDetails,
  }
}