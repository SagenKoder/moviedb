import { Play, Pause, Eye } from 'lucide-react'
import { useNowPlaying } from '../../hooks/useNowPlaying'
import { Link } from 'react-router-dom'

export function NowPlaying() {
  const { primaryNowPlaying, loading, connected } = useNowPlaying()

  // Don't show anything if not connected to Plex or nothing playing
  if (!connected || !primaryNowPlaying || loading) {
    return null
  }

  const formatTime = (milliseconds: number): string => {
    const seconds = Math.floor(milliseconds / 1000)
    const minutes = Math.floor(seconds / 60)
    const hours = Math.floor(minutes / 60)
    
    if (hours > 0) {
      return `${hours}:${(minutes % 60).toString().padStart(2, '0')}:${(seconds % 60).toString().padStart(2, '0')}`
    }
    return `${minutes}:${(seconds % 60).toString().padStart(2, '0')}`
  }

  const formatDuration = (milliseconds: number): string => {
    return formatTime(milliseconds)
  }

  const getStateIcon = (state: string) => {
    switch (state.toLowerCase()) {
      case 'playing':
        return <Play size={14} className="text-green-500" />
      case 'paused':
        return <Pause size={14} className="text-yellow-500" />
      default:
        return <Eye size={14} className="text-blue-500" />
    }
  }

  const getStateText = (state: string) => {
    switch (state.toLowerCase()) {
      case 'playing':
        return 'Playing'
      case 'paused':
        return 'Paused'
      default:
        return 'Watching'
    }
  }

  // Create the link - prefer TMDB movie page if we have mapping
  const movieLink = primaryNowPlaying.tmdbId 
    ? `/movies/${primaryNowPlaying.tmdbId}`
    : '#' // Fallback - could show a Plex-only page in the future

  return (
    <div className="flex items-center space-x-3 px-4 py-2 bg-gradient-to-r from-orange-500/10 to-orange-600/10 border border-orange-200 dark:border-orange-800 rounded-lg">
      {/* State Icon */}
      <div className="flex items-center space-x-1">
        {getStateIcon(primaryNowPlaying.playerState)}
        <span className="text-xs font-medium text-gray-600 dark:text-gray-400">
          {getStateText(primaryNowPlaying.playerState)}
        </span>
      </div>

      {/* Movie Info */}
      <div className="flex-1 min-w-0">
        {primaryNowPlaying.tmdbId ? (
          <Link 
            to={movieLink}
            className="block hover:text-orange-600 dark:hover:text-orange-400 transition-colors"
          >
            <div className="font-medium text-sm truncate">
              {primaryNowPlaying.localMovie?.title || primaryNowPlaying.title}
            </div>
            {primaryNowPlaying.year && (
              <div className="text-xs text-gray-500 dark:text-gray-400">
                {primaryNowPlaying.year}
              </div>
            )}
          </Link>
        ) : (
          <div>
            <div className="font-medium text-sm truncate">
              {primaryNowPlaying.title}
            </div>
            {primaryNowPlaying.year && (
              <div className="text-xs text-gray-500 dark:text-gray-400">
                {primaryNowPlaying.year} • No TMDB link
              </div>
            )}
          </div>
        )}
      </div>

      {/* Progress Info */}
      <div className="flex flex-col items-end text-xs text-gray-500 dark:text-gray-400">
        <div className="flex items-center space-x-1">
          <span>{formatTime(primaryNowPlaying.viewOffset)}</span>
          <span>/</span>
          <span>{formatDuration(primaryNowPlaying.duration)}</span>
        </div>
        <div className="flex items-center space-x-1">
          <div className="w-16 bg-gray-200 dark:bg-gray-700 rounded-full h-1">
            <div 
              className="bg-orange-500 h-1 rounded-full transition-all duration-300"
              style={{ width: `${Math.min(100, Math.max(0, primaryNowPlaying.progress))}%` }}
            />
          </div>
          <span className="text-xs">{primaryNowPlaying.progress}%</span>
        </div>
      </div>
    </div>
  )
}