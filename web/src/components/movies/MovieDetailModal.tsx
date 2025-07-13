import { useEffect, useState } from 'react'
import { X, Star, Calendar, Clock, Users, ExternalLink, Plus, Minus } from 'lucide-react'
import { Movie } from '../../hooks/useMovies'
import { useLists } from '../../hooks/useLists'

interface MovieDetailModalProps {
  movie: Movie | null
  isOpen: boolean
  onClose: () => void
}

export function MovieDetailModal({ movie, isOpen, onClose }: MovieDetailModalProps) {
  const { lists, addMovieToList, removeMovieFromList, getMovieInLists } = useLists()
  const [isAddingToList, setIsAddingToList] = useState(false)
  const [feedback, setFeedback] = useState<{type: 'success' | 'error', message: string} | null>(null)
  const [movieInLists, setMovieInLists] = useState<Set<number>>(new Set())

  // Reset feedback and load movie lists when modal opens
  useEffect(() => {
    if (isOpen && movie) {
      setFeedback(null)
      // Load which lists contain this movie
      getMovieInLists(movie.tmdb_id).then(listIds => {
        setMovieInLists(new Set(listIds))
      })
    }
  }, [isOpen, movie, getMovieInLists])

  // Close modal on Escape key
  useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        onClose()
      }
    }

    if (isOpen) {
      document.addEventListener('keydown', handleEscape)
      document.body.style.overflow = 'hidden'
    }

    return () => {
      document.removeEventListener('keydown', handleEscape)
      document.body.style.overflow = 'unset'
    }
  }, [isOpen, onClose])

  if (!isOpen || !movie) return null

  const formatRuntime = (runtime?: number): string => {
    if (!runtime) return ''
    const hours = Math.floor(runtime / 60)
    const minutes = runtime % 60
    return `${hours}h ${minutes}m`
  }

  const parseGenres = (genres?: string | string[]): string[] => {
    if (!genres) return []
    if (Array.isArray(genres)) return genres
    try {
      return JSON.parse(genres)
    } catch {
      return []
    }
  }

  const genreList = parseGenres(movie.genres)

  return (
    <div className="fixed inset-0 z-50 overflow-y-auto">
      {/* Backdrop */}
      <div 
        className="fixed inset-0 bg-black/50 transition-opacity"
        onClick={onClose}
      />
      
      {/* Modal */}
      <div className="flex min-h-full items-center justify-center p-2 sm:p-4">
        <div className="relative bg-white dark:bg-gray-800 rounded-lg shadow-xl max-w-4xl w-full max-h-[95vh] sm:max-h-[90vh] overflow-y-auto">
          {/* Close Button - Sticky on mobile for better accessibility */}
          <div className="sticky top-0 z-10 flex justify-end p-4 bg-white dark:bg-gray-800 rounded-t-lg">
            <button
              onClick={onClose}
              className="bg-black/20 hover:bg-black/40 text-white rounded-full p-2 transition-colors"
            >
              <X size={20} />
            </button>
          </div>

          <div className="flex flex-col md:flex-row -mt-4">
            {/* Movie Poster */}
            <div className="md:w-1/3 aspect-[2/3] md:aspect-auto bg-gray-200 dark:bg-gray-700 relative">
              {movie.poster_url ? (
                <img
                  src={movie.poster_url}
                  alt={movie.title}
                  className="w-full h-full object-cover"
                />
              ) : (
                <div className="w-full h-full flex items-center justify-center text-gray-400 dark:text-gray-500">
                  <div className="text-center">
                    <div className="text-6xl mb-4">🎬</div>
                    <div className="text-lg">No Poster Available</div>
                  </div>
                </div>
              )}
            </div>

            {/* Movie Details */}
            <div className="md:w-2/3 p-6 pt-4 flex flex-col">
              {/* Title and Tagline */}
              <div className="mb-4">
                <h1 className="text-2xl md:text-3xl font-bold text-gray-900 dark:text-white mb-2">
                  {movie.title}
                </h1>
                {movie.tagline && (
                  <p className="text-lg text-gray-600 dark:text-gray-400 italic">
                    "{movie.tagline}"
                  </p>
                )}
              </div>

              {/* Movie Stats */}
              <div className="flex flex-wrap items-center gap-4 mb-4 text-sm text-gray-600 dark:text-gray-400">
                {movie.year && (
                  <div className="flex items-center gap-1">
                    <Calendar size={16} />
                    {movie.year}
                  </div>
                )}
                
                {movie.runtime && (
                  <div className="flex items-center gap-1">
                    <Clock size={16} />
                    {formatRuntime(movie.runtime)}
                  </div>
                )}

                {movie.vote_avg && movie.vote_avg > 0 && (
                  <div className="flex items-center gap-1">
                    <Star size={16} className="fill-yellow-400 text-yellow-400" />
                    {movie.vote_avg.toFixed(1)}/10
                  </div>
                )}

                {movie.vote_count && (
                  <div className="flex items-center gap-1">
                    <Users size={16} />
                    {movie.vote_count.toLocaleString()} votes
                  </div>
                )}
              </div>

              {/* Genres */}
              {genreList.length > 0 && (
                <div className="flex flex-wrap gap-2 mb-4">
                  {genreList.map((genre, index) => (
                    <span
                      key={index}
                      className="bg-blue-100 dark:bg-blue-900 text-blue-800 dark:text-blue-200 px-3 py-1 rounded-full text-sm font-medium"
                    >
                      {genre}
                    </span>
                  ))}
                </div>
              )}

              {/* Synopsis */}
              {movie.synopsis && (
                <div className="mb-6">
                  <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-2">
                    Overview
                  </h3>
                  <p className="text-gray-700 dark:text-gray-300 leading-relaxed">
                    {movie.synopsis}
                  </p>
                </div>
              )}

              {/* Status */}
              {movie.status && (
                <div className="mb-4">
                  <span className="text-sm text-gray-600 dark:text-gray-400">
                    Status: <span className="text-gray-900 dark:text-white">{movie.status}</span>
                  </span>
                </div>
              )}

              {/* External Links */}
              <div className="mb-6">
                <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-3">
                  External Links
                </h3>
                <div className="flex flex-wrap gap-3">
                  {/* IMDb Link */}
                  {movie.external_ids?.imdb_id && (
                    <a
                      href={`https://www.imdb.com/title/${movie.external_ids.imdb_id}`}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="flex items-center gap-2 bg-yellow-500 hover:bg-yellow-600 text-black px-4 py-2 rounded-lg font-medium transition-colors"
                    >
                      <ExternalLink size={16} />
                      IMDb
                    </a>
                  )}
                  
                  {/* TMDB Link */}
                  <a
                    href={`https://www.themoviedb.org/movie/${movie.tmdb_id}`}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="flex items-center gap-2 bg-blue-500 hover:bg-blue-600 text-white px-4 py-2 rounded-lg font-medium transition-colors"
                  >
                    <ExternalLink size={16} />
                    TMDB
                  </a>
                  
                  {/* Rotten Tomatoes Search Link */}
                  <a
                    href={`https://www.rottentomatoes.com/search?search=${encodeURIComponent(movie.title + (movie.year ? ` ${movie.year}` : ''))}`}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="flex items-center gap-2 bg-red-500 hover:bg-red-600 text-white px-4 py-2 rounded-lg font-medium transition-colors"
                  >
                    <ExternalLink size={16} />
                    Rotten Tomatoes
                  </a>
                  
                  {/* Letterboxd Search Link */}
                  <a
                    href={`https://letterboxd.com/search/films/${encodeURIComponent(movie.title)}/`}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="flex items-center gap-2 bg-green-600 hover:bg-green-700 text-white px-4 py-2 rounded-lg font-medium transition-colors"
                  >
                    <ExternalLink size={16} />
                    Letterboxd
                  </a>
                </div>
              </div>

              {/* Add to Lists */}
              {lists.length > 0 && (
                <div className="mb-6">
                  <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-3">
                    Add to Your Lists
                  </h3>
                  
                  {/* Feedback */}
                  {feedback && (
                    <div className={`mb-3 p-3 rounded-lg text-sm ${
                      feedback.type === 'success' 
                        ? 'bg-green-50 dark:bg-green-900/20 text-green-800 dark:text-green-200 border border-green-200 dark:border-green-800'
                        : 'bg-red-50 dark:bg-red-900/20 text-red-800 dark:text-red-200 border border-red-200 dark:border-red-800'
                    }`}>
                      {feedback.message}
                    </div>
                  )}
                  <div className="grid grid-cols-1 gap-2 max-h-32 overflow-y-auto">
                    {lists.map((list) => {
                      const isInList = movieInLists.has(list.id)
                      
                      return (
                        <button
                          key={list.id}
                          onClick={async () => {
                            if (!movie) return
                            setIsAddingToList(true)
                            setFeedback(null)
                            
                            try {
                              if (isInList) {
                                // Remove from list
                                await removeMovieFromList(list.id, movie.tmdb_id)
                                setMovieInLists(prev => {
                                  const newSet = new Set(prev)
                                  newSet.delete(list.id)
                                  return newSet
                                })
                                setFeedback({
                                  type: 'success',
                                  message: `Removed "${movie.title}" from "${list.name}"`
                                })
                              } else {
                                // Add to list
                                await addMovieToList(list.id, movie.tmdb_id)
                                setMovieInLists(prev => new Set([...prev, list.id]))
                                setFeedback({
                                  type: 'success',
                                  message: `Added "${movie.title}" to "${list.name}"`
                                })
                              }
                            } catch (error) {
                              console.error('Failed to toggle movie in list:', error)
                              setFeedback({
                                type: 'error',
                                message: error instanceof Error ? error.message : 'Failed to update list'
                              })
                            }
                            setIsAddingToList(false)
                          }}
                          disabled={isAddingToList}
                          className={`flex items-center justify-between p-2 text-left rounded-lg transition-colors disabled:opacity-50 ${
                            isInList
                              ? 'bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 hover:bg-green-100 dark:hover:bg-green-900/30'
                              : 'bg-gray-50 dark:bg-gray-700 hover:bg-gray-100 dark:hover:bg-gray-600'
                          }`}
                        >
                          <div>
                            <div className={`font-medium text-sm ${
                              isInList ? 'text-green-800 dark:text-green-200' : 'text-gray-900 dark:text-white'
                            }`}>
                              {list.name}
                            </div>
                            <div className={`text-xs ${
                              isInList ? 'text-green-600 dark:text-green-300' : 'text-gray-500 dark:text-gray-400'
                            }`}>
                              {list.movie_count} {list.movie_count === 1 ? 'movie' : 'movies'}
                              {isInList && ' • Added'}
                            </div>
                          </div>
                          {isInList ? (
                            <Minus size={16} className="text-red-500 hover:text-red-600" />
                          ) : (
                            <Plus size={16} className="text-green-500 hover:text-green-600" />
                          )}
                        </button>
                      )
                    })}
                  </div>
                </div>
              )}

            </div>
          </div>
        </div>
      </div>
    </div>
  )
}