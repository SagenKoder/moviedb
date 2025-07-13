import { Star, Calendar, Clock } from 'lucide-react'
import { Movie } from '../../hooks/useMovies'

interface MovieCardProps {
  movie: Movie
  onClick: (movie: Movie) => void
}

export function MovieCard({ movie, onClick }: MovieCardProps) {
  const handleClick = () => {
    onClick(movie)
  }

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
    <div 
      className="bg-white dark:bg-gray-800 rounded-lg shadow-md hover:shadow-lg transition-all duration-200 cursor-pointer group overflow-hidden"
      onClick={handleClick}
    >
      {/* Movie Poster */}
      <div className="aspect-[2/3] bg-gray-200 dark:bg-gray-700 relative overflow-hidden">
        {movie.poster_url ? (
          <img
            src={movie.poster_url}
            alt={movie.title}
            className="w-full h-full object-cover group-hover:scale-105 transition-transform duration-200"
            loading="lazy"
          />
        ) : (
          <div className="w-full h-full flex items-center justify-center text-gray-400 dark:text-gray-500">
            <div className="text-center">
              <div className="text-4xl mb-2">🎬</div>
              <div className="text-sm">No Poster</div>
            </div>
          </div>
        )}
        
        {/* Rating Badge */}
        {movie.vote_avg && movie.vote_avg > 0 && (
          <div className="absolute top-2 right-2 bg-black/70 text-white text-xs px-2 py-1 rounded-full flex items-center gap-1">
            <Star size={12} className="fill-yellow-400 text-yellow-400" />
            {movie.vote_avg.toFixed(1)}
          </div>
        )}
      </div>

      {/* Movie Info */}
      <div className="p-4">
        <h3 className="font-semibold text-gray-900 dark:text-white mb-2 line-clamp-2 group-hover:text-blue-600 dark:group-hover:text-blue-400 transition-colors">
          {movie.title}
        </h3>
        
        <div className="flex items-center gap-4 text-sm text-gray-600 dark:text-gray-400 mb-3">
          {movie.year && (
            <div className="flex items-center gap-1">
              <Calendar size={14} />
              {movie.year}
            </div>
          )}
          
          {movie.runtime && (
            <div className="flex items-center gap-1">
              <Clock size={14} />
              {formatRuntime(movie.runtime)}
            </div>
          )}
        </div>

        {/* Genres */}
        {genreList.length > 0 && (
          <div className="flex flex-wrap gap-1 mb-3">
            {genreList.slice(0, 2).map((genre, index) => (
              <span
                key={index}
                className="text-xs bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 px-2 py-1 rounded-full"
              >
                {genre}
              </span>
            ))}
            {genreList.length > 2 && (
              <span className="text-xs text-gray-500 dark:text-gray-400">
                +{genreList.length - 2} more
              </span>
            )}
          </div>
        )}

        {/* Synopsis Preview */}
        {movie.synopsis && (
          <p className="text-sm text-gray-600 dark:text-gray-400 line-clamp-2">
            {movie.synopsis}
          </p>
        )}
      </div>
    </div>
  )
}

// CSS for line-clamp utility (add to index.css if not already present)
// .line-clamp-2 {
//   display: -webkit-box;
//   -webkit-line-clamp: 2;
//   -webkit-box-orient: vertical;
//   overflow: hidden;
// }