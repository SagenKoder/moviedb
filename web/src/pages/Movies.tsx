import { useState, useEffect } from 'react'
import { Search, Loader2 } from 'lucide-react'
import { useMovies, Movie } from '../hooks/useMovies'
import { MovieCard } from '../components/movies/MovieCard'
import { MovieDetailModal } from '../components/movies/MovieDetailModal'

export function Movies() {
  const { loading, error, searchMovies, getMovieDetails } = useMovies()
  const [movies, setMovies] = useState<Movie[]>([])
  const [searchQuery, setSearchQuery] = useState('')
  const [currentPage, setCurrentPage] = useState(1)
  const [totalPages, setTotalPages] = useState(1)
  const [selectedMovie, setSelectedMovie] = useState<Movie | null>(null)
  const [showModal, setShowModal] = useState(false)

  // Load popular movies on initial page load
  useEffect(() => {
    loadMovies()
  }, [currentPage])

  const loadMovies = async (query: string = searchQuery) => {
    try {
      const response = await searchMovies(query, currentPage)
      setMovies(response.results)
      setTotalPages(response.total_pages || 1)
    } catch (err) {
      console.error('Failed to load movies:', err)
    }
  }

  const handleSearch = async (e: React.FormEvent) => {
    e.preventDefault()
    setCurrentPage(1)
    loadMovies(searchQuery)
  }

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

  const handleCloseModal = () => {
    setShowModal(false)
    setSelectedMovie(null)
  }

  const handlePageChange = (page: number) => {
    setCurrentPage(page)
  }

  const clearSearch = () => {
    setSearchQuery('')
    setCurrentPage(1)
    loadMovies('')
  }

  return (
    <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      {/* Header */}
      <div className="mb-8">
        <h1 className="text-3xl font-bold text-gray-900 dark:text-white mb-2">
          Movies
        </h1>
        <p className="text-gray-600 dark:text-gray-300">
          Discover and explore our collection of movies
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
              placeholder="Search for movies..."
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
      {movies.length > 0 && (
        <div className="mb-6">
          <p className="text-gray-600 dark:text-gray-400">
            {searchQuery ? (
              <>Showing results for "{searchQuery}"</>
            ) : (
              <>Showing popular movies</>
            )}
            {totalPages > 1 && (
              <> • Page {currentPage} of {totalPages}</>
            )}
          </p>
        </div>
      )}

      {/* Error Message */}
      {error && (
        <div className="mb-6 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-4">
          <p className="text-red-800 dark:text-red-200">
            Error loading movies: {error}
          </p>
        </div>
      )}

      {/* Movies Grid */}
      {loading && movies.length === 0 ? (
        <div className="flex justify-center items-center py-12">
          <Loader2 size={40} className="animate-spin text-blue-600" />
        </div>
      ) : movies.length === 0 ? (
        <div className="text-center py-12">
          <div className="text-6xl mb-4">🎬</div>
          <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-2">
            No movies found
          </h3>
          <p className="text-gray-600 dark:text-gray-400">
            {searchQuery ? 'Try a different search term' : 'No movies available'}
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

      {/* Movie Detail Modal */}
      <MovieDetailModal
        movie={selectedMovie}
        isOpen={showModal}
        onClose={handleCloseModal}
      />
    </div>
  )
}