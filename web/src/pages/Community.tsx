import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth0 } from '@auth0/auth0-react'
import { Search, Loader2, User, Calendar, Film, Play } from 'lucide-react'

interface CommunityUser {
  id: number
  auth0_id: string
  name: string
  username?: string
  avatar_url?: string
  created_at: string
  list_count: number
  movie_count: number
}

export function Community() {
  const navigate = useNavigate()
  const { getAccessTokenSilently } = useAuth0()
  const [users, setUsers] = useState<CommunityUser[]>([])
  const [searchQuery, setSearchQuery] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [currentPage, setCurrentPage] = useState(1)
  const [totalPages, setTotalPages] = useState(1)

  // Load initial users on mount and when page changes
  useEffect(() => {
    loadUsers()
  }, [currentPage])

  const loadUsers = async (query: string = searchQuery) => {
    setLoading(true)
    setError(null)
    
    try {
      // Build URL with pagination and search parameters
      const params = new URLSearchParams({
        page: currentPage.toString(),
        limit: '20'
      })
      
      if (query) {
        params.append('search', query)
      }
      
      const response = await fetch(`/api/users?${params.toString()}`, {
        headers: {
          'Authorization': `Bearer ${await getToken()}`,
          'Content-Type': 'application/json',
        }
      })
      
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`)
      }
      
      const data = await response.json()
      setUsers(data.users || [])
      setTotalPages(data.total_pages || 1)
    } catch (err) {
      console.error('Failed to load users:', err)
      setError(err instanceof Error ? err.message : 'Failed to load users')
      setUsers([])
      setTotalPages(1)
    } finally {
      setLoading(false)
    }
  }

  const getToken = async () => {
    try {
      return await getAccessTokenSilently()
    } catch (error) {
      console.error('Failed to get access token:', error)
      throw error
    }
  }

  const handleSearch = async (e: React.FormEvent) => {
    e.preventDefault()
    setCurrentPage(1) // Reset to first page when searching
    loadUsers(searchQuery)
  }

  const clearSearch = () => {
    setSearchQuery('')
    setCurrentPage(1)
    loadUsers('')
  }

  const handlePageChange = (page: number) => {
    setCurrentPage(page)
  }

  const handleUserClick = (user: CommunityUser) => {
    // Navigate to the user's profile page
    navigate(`/profile/${user.auth0_id}`)
  }

  const formatDate = (dateString: string) => {
    try {
      return new Date(dateString).toLocaleDateString()
    } catch {
      return 'Unknown'
    }
  }

  return (
    <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      {/* Header */}
      <div className="mb-8">
        <h1 className="text-3xl font-bold text-gray-900 dark:text-white mb-2">
          Community
        </h1>
        <p className="text-gray-600 dark:text-gray-300">
          Discover and connect with other movie enthusiasts
        </p>
      </div>

      {/* Search Bar */}
      <div className="mb-8">
        <form onSubmit={handleSearch} className="flex gap-4">
          <div className="flex-1 relative">
            <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
              <Search className="h-5 w-5 text-gray-400" />
            </div>
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="Search for users by name or username..."
              className="block w-full pl-10 pr-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-500 dark:placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            />
          </div>
          <button
            type="submit"
            disabled={loading}
            className="bg-blue-600 hover:bg-blue-700 disabled:bg-blue-400 text-white px-6 py-2 rounded-lg font-medium transition-colors flex items-center gap-2"
          >
            {loading ? (
              <>
                <Loader2 size={20} className="animate-spin" />
                Searching...
              </>
            ) : (
              'Search'
            )}
          </button>
          {searchQuery && (
            <button
              type="button"
              onClick={clearSearch}
              className="bg-gray-200 dark:bg-gray-700 hover:bg-gray-300 dark:hover:bg-gray-600 text-gray-900 dark:text-white px-4 py-2 rounded-lg font-medium transition-colors"
            >
              Clear
            </button>
          )}
        </form>
      </div>

      {/* Results Info */}
      {users.length > 0 && (
        <div className="mb-6">
          <p className="text-gray-600 dark:text-gray-400">
            {searchQuery ? (
              <>Showing {users.length} result{users.length === 1 ? '' : 's'} for "{searchQuery}"</>
            ) : (
              <>Showing {users.length} community member{users.length === 1 ? '' : 's'}</>
            )}
          </p>
        </div>
      )}

      {/* Error Message */}
      {error && (
        <div className="mb-6 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-4">
          <p className="text-red-800 dark:text-red-200">
            Error loading users: {error}
          </p>
        </div>
      )}

      {/* Users Grid */}
      {loading && users.length === 0 ? (
        <div className="flex justify-center items-center py-12">
          <Loader2 size={40} className="animate-spin text-blue-600" />
        </div>
      ) : users.length === 0 ? (
        <div className="text-center py-12">
          <div className="text-6xl mb-4">👥</div>
          <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-2">
            {searchQuery ? 'No users found' : 'No community members yet'}
          </h3>
          <p className="text-gray-600 dark:text-gray-400">
            {searchQuery ? 'Try a different search term' : 'Be the first to join the community!'}
          </p>
        </div>
      ) : (
        <>
          <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-6">
            {users.map((user) => (
              <div
                key={user.id}
                onClick={() => handleUserClick(user)}
                className="bg-white dark:bg-gray-800 rounded-lg shadow hover:shadow-lg transition-all duration-200 cursor-pointer hover:scale-105 border border-gray-200 dark:border-gray-700"
              >
                <div className="p-6">
                  {/* User Avatar */}
                  <div className="w-16 h-16 rounded-full flex items-center justify-center mx-auto mb-4 overflow-hidden">
                    {user.avatar_url ? (
                      <img 
                        src={user.avatar_url} 
                        alt={user.name}
                        className="w-full h-full object-cover"
                      />
                    ) : (
                      <div className="w-full h-full bg-blue-600 flex items-center justify-center">
                        <User className="w-8 h-8 text-white" />
                      </div>
                    )}
                  </div>
                  
                  {/* User Info */}
                  <div className="text-center">
                    <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-1 truncate" title={user.name}>
                      {user.name}
                    </h3>
                    
                    {user.username && (
                      <p className="text-sm text-gray-500 dark:text-gray-400 mb-2">
                        @{user.username}
                      </p>
                    )}
                    
                    {/* Member Since */}
                    <div className="flex items-center justify-center gap-1 text-xs text-gray-500 dark:text-gray-400 mb-3">
                      <Calendar size={12} />
                      <span>Joined {formatDate(user.created_at)}</span>
                    </div>
                    
                    {/* User Stats */}
                    <div className="flex items-center justify-center gap-3 text-xs text-gray-500 dark:text-gray-400">
                      <div className="flex items-center gap-1">
                        <Film size={12} />
                        <span>{user.list_count} list{user.list_count === 1 ? '' : 's'}</span>
                      </div>
                      <div className="flex items-center gap-1">
                        <Play size={12} />
                        <span>{user.movie_count} movie{user.movie_count === 1 ? '' : 's'}</span>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            ))}
          </div>

          {/* Pagination */}
          {totalPages > 1 && (
            <div className="flex justify-center items-center mt-8 gap-2">
              <button
                onClick={() => handlePageChange(currentPage - 1)}
                disabled={currentPage === 1 || loading}
                className="bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 px-3 py-2 rounded-lg disabled:opacity-50 disabled:cursor-not-allowed hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors"
              >
                Previous
              </button>
              
              <span className="text-gray-600 dark:text-gray-400 px-4">
                Page {currentPage} of {totalPages}
              </span>
              
              <button
                onClick={() => handlePageChange(currentPage + 1)}
                disabled={currentPage === totalPages || loading}
                className="bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 px-3 py-2 rounded-lg disabled:opacity-50 disabled:cursor-not-allowed hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors"
              >
                Next
              </button>
            </div>
          )}
        </>
      )}
    </div>
  )
}