import { useParams, useSearchParams } from 'react-router-dom'
import { useAuth0 } from '@auth0/auth0-react'
import { useState, useEffect } from 'react'
import { useLists } from '../hooks/useLists'
import { useMovies, Movie } from '../hooks/useMovies'
import { MovieCard } from '../components/movies/MovieCard'
import { MovieDetailModal } from '../components/movies/MovieDetailModal'
import { Filter, User, Loader2, Film, Play, Edit3 } from 'lucide-react'
import { EditListModal } from '../components/lists/EditListModal'

export function Profile() {
  const { userId } = useParams<{ userId: string }>()
  const [searchParams, setSearchParams] = useSearchParams()
  const { user } = useAuth0()
  const { loading: listsLoading, error: listsError, getAllUserMovies, getListDetails, getUserLists, getUserProfile } = useLists()
  const [lists, setLists] = useState<any[]>([])
  const [profileUser, setProfileUser] = useState<any>(null)
  const [userLoading, setUserLoading] = useState(false)
  const [userError, setUserError] = useState<string | null>(null)
  const { getMovieDetails } = useMovies()
  
  // Get filter from URL or default to 'all'
  const selectedFilter = searchParams.get('list') || 'all'
  const [movies, setMovies] = useState<Movie[]>([])
  const [moviesLoading, setMoviesLoading] = useState(false)
  const [moviesError, setMoviesError] = useState<string | null>(null)
  const [selectedMovie, setSelectedMovie] = useState<Movie | null>(null)
  const [showModal, setShowModal] = useState(false)
  const [editingList, setEditingList] = useState<any>(null)
  const [currentPage, setCurrentPage] = useState(1)
  const [totalPages, setTotalPages] = useState(1)
  const [totalMovies, setTotalMovies] = useState(0)

  const isOwnProfile = !userId || userId === user?.sub
  
  // Handle filter changes and update URL
  const handleFilterChange = (newFilter: string) => {
    const newSearchParams = new URLSearchParams(searchParams)
    if (newFilter === 'all') {
      newSearchParams.delete('list')
    } else {
      newSearchParams.set('list', newFilter)
    }
    setSearchParams(newSearchParams)
  }

  // Filter lists based on selection (privacy is handled server-side)
  const filteredLists = selectedFilter === 'all' 
    ? lists 
    : lists.filter(list => list.id.toString() === selectedFilter)

  // Get total movies across all lists (for "All Lists" dropdown option)
  const totalMoviesFromLists = lists.reduce((sum, list) => sum + list.movie_count, 0)
  
  // Get movies count for current filter display (use pagination data when showing all movies)
  const currentFilterMovieCount = selectedFilter === 'all' 
    ? totalMovies 
    : filteredLists.reduce((sum, list) => sum + list.movie_count, 0)

  // Load user profile and lists on mount and when userId changes
  useEffect(() => {
    const loadUserData = async () => {
      setUserLoading(true)
      setUserError(null)
      
      try {
        // Load user profile and lists in parallel
        const [userProfile, userLists] = await Promise.all([
          getUserProfile(userId),
          getUserLists(userId)
        ])
        
        setProfileUser(userProfile)
        setLists(userLists)
      } catch (err) {
        console.error('Failed to load user data:', err)
        setUserError(err instanceof Error ? err.message : 'Failed to load user data')
        setProfileUser(null)
        setLists([])
      } finally {
        setUserLoading(false)
      }
    }
    
    loadUserData()
  }, [userId])

  // Load movies when filter changes AND lists are loaded
  useEffect(() => {
    if (lists.length > 0) {
      setCurrentPage(1) // Reset to first page when filter changes
      loadMovies()
    }
  }, [selectedFilter, lists.length]) // Watch lists.length instead of the whole lists array

  // Load movies when page changes
  useEffect(() => {
    if (lists.length > 0) {
      loadMovies()
    }
  }, [currentPage])

  const loadMovies = async () => {
    if (!lists.length) return
    
    setMoviesLoading(true)
    setMoviesError(null)
    setMovies([]) // Clear existing movies first to prevent duplicates
    
    try {
      let formattedMovies: Movie[] = []
      
      if (selectedFilter === 'all') {
        // Get all user movies with pagination
        const response = await getAllUserMovies(userId, currentPage, 20)
        setTotalPages(response.total_pages || 1)
        setTotalMovies(response.total || 0)
        formattedMovies = (response.movies || []).map((movie: any) => ({
          id: movie.id,
          tmdb_id: movie.tmdb_id,
          title: movie.title,
          year: movie.year,
          poster_url: movie.poster_url,
          synopsis: movie.synopsis,
        }))
      } else {
        // Get movies from specific list (no pagination for single list view for now)
        const selectedList = lists.find(l => l.id.toString() === selectedFilter)
        if (selectedList) {
          const listDetail = await getListDetails(selectedList.id)
          formattedMovies = (listDetail.movies || []).map((movie: any) => ({
            id: movie.id,
            tmdb_id: movie.tmdb_id,
            title: movie.title,
            year: movie.year,
            poster_url: movie.poster_url,
            synopsis: movie.synopsis,
          }))
          setTotalPages(1)
          setTotalMovies(formattedMovies.length)
        }
      }
      
      setMovies(formattedMovies)
    } catch (err) {
      console.error('Failed to load movies:', err)
      setMoviesError(err instanceof Error ? err.message : 'Failed to load movies')
      setMovies([])
      setTotalPages(1)
      setTotalMovies(0)
    } finally {
      setMoviesLoading(false)
    }
  }

  // Handle page change
  const handlePageChange = (page: number) => {
    setCurrentPage(page)
  }

  // Handle movie click like Movies page
  const handleMovieClick = async (movie: Movie) => {
    setSelectedMovie(movie)
    setShowModal(true)

    try {
      // Get detailed movie info
      const detailedMovie = await getMovieDetails(movie.tmdb_id)
      setSelectedMovie(detailedMovie)
    } catch (err) {
      console.error('Failed to load movie details:', err)
      // Keep the basic movie info if detailed fetch fails
    }
  }

  return (
    <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      {/* Profile Header */}
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow mb-8 p-6 transition-colors duration-200">
        {userLoading ? (
          <div className="flex items-center space-x-4">
            <div className="w-16 h-16 bg-gray-300 dark:bg-gray-600 rounded-full animate-pulse"></div>
            <div>
              <div className="h-8 bg-gray-300 dark:bg-gray-600 rounded animate-pulse w-48 mb-2"></div>
              <div className="h-4 bg-gray-300 dark:bg-gray-600 rounded animate-pulse w-32"></div>
            </div>
          </div>
        ) : userError ? (
          <div className="text-center py-8">
            <div className="text-red-500 mb-2">
              <User className="w-12 h-12 mx-auto mb-4 opacity-50" />
            </div>
            <h2 className="text-xl font-semibold text-gray-900 dark:text-white mb-2">User not found</h2>
            <p className="text-gray-600 dark:text-gray-400">{userError}</p>
          </div>
        ) : (
          <div className="flex items-start space-x-6">
            {/* Avatar */}
            <div className="w-20 h-20 rounded-full flex items-center justify-center overflow-hidden flex-shrink-0">
              {profileUser?.avatar_url ? (
                <img 
                  src={profileUser.avatar_url} 
                  alt={profileUser.name}
                  className="w-full h-full object-cover"
                />
              ) : (
                <div className="w-full h-full bg-blue-600 flex items-center justify-center">
                  <User className="w-10 h-10 text-white" />
                </div>
              )}
            </div>
            
            {/* User Info */}
            <div className="flex-1">
              <h1 className="text-2xl font-bold text-gray-900 dark:text-white mb-2">
                {profileUser?.name || (isOwnProfile ? (user?.given_name || user?.name || 'Your Profile') : 'Unknown User')}
              </h1>
              {profileUser?.username && (
                <p className="text-sm text-gray-500 dark:text-gray-400 mb-3">@{profileUser.username}</p>
              )}
              
              {/* Stats */}
              <div className="flex items-center space-x-6 text-sm text-gray-600 dark:text-gray-400">
                <div className="flex items-center space-x-2">
                  <Film size={16} />
                  <span className="font-medium">{lists.length}</span>
                  <span>{lists.length === 1 ? 'list' : 'lists'}</span>
                </div>
                <div className="flex items-center space-x-2">
                  <Play size={16} />
                  <span className="font-medium">{totalMoviesFromLists}</span>
                  <span>{totalMoviesFromLists === 1 ? 'movie' : 'movies'}</span>
                </div>
              </div>
              
              <p className="text-gray-600 dark:text-gray-300 mt-3">
                {isOwnProfile ? 'Your movie collection and lists' : `${profileUser?.name || 'User'}'s movie profile`}
              </p>
            </div>
          </div>
        )}
      </div>

      {/* Lists Overview */}
      {lists.length > 0 && (
        <div className="bg-white dark:bg-gray-800 rounded-lg shadow mb-6 transition-colors duration-200">
          <div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700">
            <h2 className="text-lg font-semibold text-gray-900 dark:text-white">
              {isOwnProfile ? 'Your Lists' : `${profileUser?.name || 'User'}'s Lists`}
            </h2>
          </div>
          <div className="p-6">
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
              {/* All Movies Option */}
              <div
                className={`border rounded-lg p-4 cursor-pointer transition-colors relative group ${
                  selectedFilter === 'all' 
                    ? 'border-blue-500 dark:border-blue-400 bg-blue-50 dark:bg-blue-900/20' 
                    : 'border-gray-200 dark:border-gray-700 hover:border-blue-500 dark:hover:border-blue-400'
                }`}
                onClick={() => handleFilterChange('all')}
              >
                <div className="flex items-start justify-between mb-1">
                  <h3 className="font-semibold text-gray-900 dark:text-white pr-8">All Movies</h3>
                  <span className="text-xs text-gray-400 dark:text-gray-500 flex-shrink-0">
                    All Lists
                  </span>
                </div>
                <p className="text-sm text-gray-600 dark:text-gray-400 mb-2">
                  View all movies from all your lists
                </p>
                <span className="text-sm text-gray-500 dark:text-gray-400">
                  {totalMoviesFromLists} {totalMoviesFromLists === 1 ? 'movie' : 'movies'}
                </span>
              </div>

              {/* Individual Lists */}
              {lists.map((list) => (
                <div
                  key={list.id}
                  className={`border rounded-lg p-4 cursor-pointer transition-colors relative group ${
                    selectedFilter === list.id.toString() 
                      ? 'border-blue-500 dark:border-blue-400 bg-blue-50 dark:bg-blue-900/20' 
                      : 'border-gray-200 dark:border-gray-700 hover:border-blue-500 dark:hover:border-blue-400'
                  }`}
                  onClick={() => handleFilterChange(list.id.toString())}
                >
                  <div className="flex items-start justify-between mb-1">
                    <h3 className="font-semibold text-gray-900 dark:text-white pr-8">{list.name}</h3>
                    <span className="text-xs text-gray-400 dark:text-gray-500 flex-shrink-0">
                      {list.is_public ? 'Public' : 'Private'}
                    </span>
                  </div>
                  {list.description && (
                    <p className="text-sm text-gray-600 dark:text-gray-400 mb-2 line-clamp-2">
                      {list.description}
                    </p>
                  )}
                  <div className="flex items-center justify-between">
                    <span className="text-sm text-gray-500 dark:text-gray-400">
                      {list.movie_count} {list.movie_count === 1 ? 'movie' : 'movies'}
                    </span>
                    {isOwnProfile && (
                      <button
                        onClick={(e) => {
                          e.stopPropagation()
                          setEditingList(list)
                        }}
                        className="p-1 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 opacity-0 group-hover:opacity-100 transition-all duration-200"
                        title="Edit list"
                      >
                        <Edit3 size={16} />
                      </button>
                    )}
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>
      )}

      {/* Error State */}
      {listsError && (
        <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-4 mb-6">
          <p className="text-red-800 dark:text-red-200">Failed to load profile data: {listsError}</p>
        </div>
      )}

      {/* Feed Content */}
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow transition-colors duration-200">
        <div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700">
          <h2 className="text-lg font-semibold text-gray-900 dark:text-white">
            {selectedFilter === 'all' ? 'All Movies' : lists.find(l => l.id.toString() === selectedFilter)?.name}
          </h2>
          <p className="text-sm text-gray-600 dark:text-gray-400">
            {currentFilterMovieCount} {currentFilterMovieCount === 1 ? 'movie' : 'movies'} total
          </p>
        </div>

        <div className="p-6">
          {/* Error Message */}
          {moviesError && (
            <div className="mb-6 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-4">
              <p className="text-red-800 dark:text-red-200">
                Error loading movies: {moviesError}
              </p>
            </div>
          )}

          {/* Movies Grid - Same pattern as Movies page */}
          {moviesLoading && movies.length === 0 ? (
            <div className="flex justify-center items-center py-12">
              <Loader2 size={40} className="animate-spin text-blue-600" />
            </div>
          ) : movies.length === 0 ? (
            <div className="text-center py-12">
              <div className="text-6xl mb-4">🎬</div>
              <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-2">
                {selectedFilter === 'all' ? 'No movies yet' : 'This list is empty'}
              </h3>
              <p className="text-gray-600 dark:text-gray-400">
                {isOwnProfile 
                  ? selectedFilter === 'all' 
                    ? 'Start by creating a list and adding some movies'
                    : 'Add some movies to this list to see them here'
                  : 'This user hasn\'t added any movies yet'
                }
              </p>
            </div>
          ) : (
            <>
              <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 gap-6">
                {movies.map((movie) => (
                  <MovieCard
                    key={movie.id}
                    movie={movie}
                    onClick={handleMovieClick}
                  />
                ))}
              </div>

              {/* Pagination - only show for "All Movies" view */}
              {selectedFilter === 'all' && totalPages > 1 && (
                <div className="flex justify-center items-center mt-8 gap-2">
                  <button
                    onClick={() => handlePageChange(currentPage - 1)}
                    disabled={currentPage === 1 || moviesLoading}
                    className="bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 px-3 py-2 rounded-lg disabled:opacity-50 disabled:cursor-not-allowed hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors"
                  >
                    Previous
                  </button>
                  
                  <span className="text-gray-600 dark:text-gray-400 px-4">
                    Page {currentPage} of {totalPages}
                  </span>
                  
                  <button
                    onClick={() => handlePageChange(currentPage + 1)}
                    disabled={currentPage === totalPages || moviesLoading}
                    className="bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 px-3 py-2 rounded-lg disabled:opacity-50 disabled:cursor-not-allowed hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors"
                  >
                    Next
                  </button>
                </div>
              )}
            </>
          )}
        </div>
      </div>


      {/* Movie Detail Modal - Same as Movies page */}
      <MovieDetailModal
        movie={selectedMovie}
        isOpen={showModal}
        onClose={() => {
          setShowModal(false)
          setSelectedMovie(null)
        }}
      />

      {/* Edit List Modal */}
      {isOwnProfile && (
        <EditListModal
          list={editingList}
          isOpen={!!editingList}
          onClose={() => setEditingList(null)}
          onUpdate={async () => {
            // Reload user data to refresh lists and movie counts
            const [userProfile, userLists] = await Promise.all([
              getUserProfile(userId),
              getUserLists(userId)
            ])
            
            setProfileUser(userProfile)
            setLists(userLists)
            
            // Reload movies if we're viewing all movies or the edited list
            if (lists.length > 0) {
              loadMovies()
            }
          }}
        />
      )}
    </div>
  )
}