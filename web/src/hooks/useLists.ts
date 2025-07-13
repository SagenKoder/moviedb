import { useState, useEffect } from 'react'
import { useAuth0 } from '@auth0/auth0-react'

export interface List {
  id: number
  name: string
  description: string
  is_public: boolean
  created_at: string
  movie_count: number
}

export interface CreateListRequest {
  name: string
  description: string
  is_public: boolean
}

export function useLists() {
  const { getAccessTokenSilently } = useAuth0()
  const [lists, setLists] = useState<List[]>([])
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

  const fetchLists = async () => {
    setLoading(true)
    setError(null)

    try {
      const data = await apiCall('/api/lists')
      setLists(data.lists || [])
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to fetch lists'
      setError(errorMessage)
    } finally {
      setLoading(false)
    }
  }

  const createList = async (listData: CreateListRequest): Promise<List> => {
    setLoading(true)
    setError(null)

    try {
      const data = await apiCall('/api/lists', {
        method: 'POST',
        body: JSON.stringify(listData),
      })
      
      // Refresh lists after creating
      await fetchLists()
      return data
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to create list'
      setError(errorMessage)
      throw err
    } finally {
      setLoading(false)
    }
  }

  const updateList = async (listId: number, listData: CreateListRequest): Promise<List> => {
    setLoading(true)
    setError(null)

    try {
      const data = await apiCall(`/api/lists/${listId}`, {
        method: 'PUT',
        body: JSON.stringify(listData),
      })
      
      // Refresh lists after updating
      await fetchLists()
      return data
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to update list'
      setError(errorMessage)
      throw err
    } finally {
      setLoading(false)
    }
  }

  const deleteList = async (listId: number) => {
    setLoading(true)
    setError(null)

    try {
      await apiCall(`/api/lists/${listId}`, {
        method: 'DELETE',
      })
      
      // Refresh lists after deleting
      await fetchLists()
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to delete list'
      setError(errorMessage)
      throw err
    } finally {
      setLoading(false)
    }
  }

  const addMovieToList = async (listId: number, movieId: number) => {
    setLoading(true)
    setError(null)

    try {
      await apiCall(`/api/lists/${listId}/movies/${movieId}`, {
        method: 'POST',
      })
      
      // Refresh lists to update movie counts
      await fetchLists()
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to add movie to list'
      setError(errorMessage)
      throw err
    } finally {
      setLoading(false)
    }
  }

  const removeMovieFromList = async (listId: number, movieId: number) => {
    setLoading(true)
    setError(null)

    try {
      await apiCall(`/api/lists/${listId}/movies/${movieId}`, {
        method: 'DELETE',
      })
      
      // Refresh lists to update movie counts
      await fetchLists()
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to remove movie from list'
      setError(errorMessage)
      throw err
    } finally {
      setLoading(false)
    }
  }

  const getMovieInLists = async (movieId: number): Promise<number[]> => {
    try {
      const data = await apiCall(`/api/movies/${movieId}/lists`)
      return data.list_ids || []
    } catch (err) {
      console.error('Failed to get movie lists:', err)
      return []
    }
  }

  const getListDetails = async (listId: number) => {
    try {
      const data = await apiCall(`/api/lists/${listId}`)
      return data
    } catch (err) {
      console.error('Failed to get list details:', err)
      throw err
    }
  }

  const getAllUserMovies = async () => {
    try {
      const data = await apiCall('/api/me/movies')
      return data.movies || []
    } catch (err) {
      console.error('Failed to get all user movies:', err)
      throw err
    }
  }

  const getUserLists = async (userId?: string) => {
    try {
      const endpoint = userId ? `/api/users/${userId}/lists` : '/api/lists'
      const data = await apiCall(endpoint)
      return data.lists || []
    } catch (err) {
      console.error('Failed to get user lists:', err)
      throw err
    }
  }

  const getUserProfile = async (userId?: string) => {
    try {
      const endpoint = userId ? `/api/users/${userId}` : '/api/me'
      const data = await apiCall(endpoint)
      return data
    } catch (err) {
      console.error('Failed to get user profile:', err)
      throw err
    }
  }

  // Load lists on mount
  useEffect(() => {
    fetchLists()
  }, [])

  return {
    lists,
    loading,
    error,
    fetchLists,
    createList,
    updateList,
    deleteList,
    addMovieToList,
    removeMovieFromList,
    getMovieInLists,
    getListDetails,
    getAllUserMovies,
    getUserLists,
    getUserProfile,
  }
}