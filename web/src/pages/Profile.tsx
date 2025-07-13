import { useParams, useSearchParams } from 'react-router-dom'
import { useAuth0 } from '@auth0/auth0-react'
import { useState, useEffect } from 'react'
import { useLists } from '../hooks/useLists'
import { useMovies, Movie } from '../hooks/useMovies'
import { MovieCard } from '../components/movies/MovieCard'
import { MovieDetailModal } from '../components/movies/MovieDetailModal'
import { Filter, User, Loader2 } from 'lucide-react'

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
  const totalMovies = lists.reduce((sum, list) => sum + list.movie_count, 0)
  
  // Get movies count for current filter display
  const currentFilterMovieCount = filteredLists.reduce((sum, list) => sum + list.movie_count, 0)

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
      loadMovies()
    }
  }, [selectedFilter, lists.length]) // Watch lists.length instead of the whole lists array

  const loadMovies = async () => {
    if (!lists.length) return
    
    setMoviesLoading(true)
    setMoviesError(null)
    
    try {
      let formattedMovies: Movie[] = []
      
      if (selectedFilter === 'all') {
        // Get all user movies
        const userMovies = await getAllUserMovies()
        formattedMovies = userMovies.map((movie: any) => ({
          id: movie.id,
          tmdb_id: movie.tmdb_id,
          title: movie.title,
          year: movie.year,
          poster_url: movie.poster_url,
          synopsis: movie.synopsis,
        }))
      } else {
        // Get movies from specific list
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
        }
      }
      
      setMovies(formattedMovies)
    } catch (err) {
      console.error('Failed to load movies:', err)
      setMoviesError(err instanceof Error ? err.message : 'Failed to load movies')
      setMovies([])
    } finally {
      setMoviesLoading(false)
    }
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
          <div className="flex items-center space-x-4">
            <div className="w-16 h-16 bg-blue-600 rounded-full flex items-center justify-center">
              <User className="w-8 h-8 text-white" />
            </div>
            <div>
              <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
                {profileUser?.name || (isOwnProfile ? (user?.given_name || user?.name || 'Your Profile') : 'Unknown User')}
              </h1>
              <p className="text-gray-600 dark:text-gray-300">
                {isOwnProfile ? 'Your movie collection and lists' : `${profileUser?.name || 'User'}'s movie profile`}
              </p>
              {profileUser?.username && (
                <p className="text-sm text-gray-500 dark:text-gray-400">@{profileUser.username}</p>
              )}
            </div>
          </div>
        )}
      </div>

      {/* Filter Controls */}
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow mb-6 p-4 transition-colors duration-200">
        <div className="flex items-center space-x-4">
          <Filter className="w-5 h-5 text-gray-500 dark:text-gray-400" />
          <span className="text-sm font-medium text-gray-700 dark:text-gray-300">Filter by list:</span>
          
          <select
            value={selectedFilter}
            onChange={(e) => handleFilterChange(e.target.value)}
            className="bg-white dark:bg-gray-700 border border-gray-300 dark:border-gray-600 rounded-md px-3 py-1 text-sm text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="all">All Lists ({totalMovies} movies)</option>
            {lists.map((list) => (
              <option key={list.id} value={list.id.toString()}>
                {list.name} ({list.movie_count} {list.movie_count === 1 ? 'movie' : 'movies'})
                {isOwnProfile && !list.is_public && ' 🔒'}
              </option>
            ))}
          </select>

          {listsLoading && <Loader2 className="w-4 h-4 animate-spin text-gray-500" />}
        </div>
      </div>

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
            <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 gap-6">
              {movies.map((movie) => (
                <MovieCard
                  key={movie.id}
                  movie={movie}
                  onClick={handleMovieClick}
                />
              ))}
            </div>
          )}
        </div>
      </div>

      {/* Lists Overview (if showing all) */}
      {selectedFilter === 'all' && lists.length > 0 && (
        <div className="mt-8 bg-white dark:bg-gray-800 rounded-lg shadow transition-colors duration-200">
          <div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700">
            <h2 className="text-lg font-semibold text-gray-900 dark:text-white">Your Lists</h2>
          </div>
          <div className="p-6">
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
              {lists.map((list) => (
                <div
                  key={list.id}
                  className="border border-gray-200 dark:border-gray-700 rounded-lg p-4 cursor-pointer hover:border-blue-500 dark:hover:border-blue-400 transition-colors"
                  onClick={() => handleFilterChange(list.id.toString())}
                >
                  <h3 className="font-semibold text-gray-900 dark:text-white mb-1">{list.name}</h3>
                  {list.description && (
                    <p className="text-sm text-gray-600 dark:text-gray-400 mb-2 line-clamp-2">
                      {list.description}
                    </p>
                  )}
                  <div className="flex items-center justify-between">
                    <span className="text-sm text-gray-500 dark:text-gray-400">
                      {list.movie_count} {list.movie_count === 1 ? 'movie' : 'movies'}
                    </span>
                    <span className="text-xs text-gray-400 dark:text-gray-500">
                      {list.is_public ? 'Public' : 'Private'}
                    </span>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>
      )}

      {/* Movie Detail Modal - Same as Movies page */}
      <MovieDetailModal
        movie={selectedMovie}
        isOpen={showModal}
        onClose={() => {
          setShowModal(false)
          setSelectedMovie(null)
        }}
      />
    </div>
  )
}