import { useState, useRef, useEffect } from 'react'
import { useAuth0 } from '@auth0/auth0-react'
import { ChevronDown, Moon, Sun, Server, CheckCircle, Loader2 } from 'lucide-react'
import { LogoutButton } from '../auth/LogoutButton'
import { PlexConnectionModal } from '../plex/PlexConnectionModal'
import { usePlex } from '../../hooks/usePlex'

interface UserMenuProps {
  isDarkMode: boolean
  onToggleDarkMode: () => void
}

export function UserMenu({ isDarkMode, onToggleDarkMode }: UserMenuProps) {
  const { user } = useAuth0()
  const [isOpen, setIsOpen] = useState(false)
  const [showPlexModal, setShowPlexModal] = useState(false)
  const menuRef = useRef<HTMLDivElement>(null)
  const { status: plexStatus, loading: plexLoading, disconnect: disconnectPlex, refreshStatus: refreshPlexStatus } = usePlex()

  // Close menu when clicking outside
  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (menuRef.current && !menuRef.current.contains(event.target as Node)) {
        setIsOpen(false)
      }
    }

    document.addEventListener('mousedown', handleClickOutside)
    return () => {
      document.removeEventListener('mousedown', handleClickOutside)
    }
  }, [])

  return (
    <>
      <div className="relative" ref={menuRef}>
      {/* User Info Button */}
      <button
        onClick={() => setIsOpen(!isOpen)}
        className="flex items-center space-x-2 text-gray-700 dark:text-gray-200 hover:text-gray-900 dark:hover:text-white transition-colors duration-200 p-2 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700"
      >
        {user?.picture && (
          <img
            src={user.picture}
            alt={user.name || 'User'}
            className="h-8 w-8 rounded-full"
          />
        )}
        <span className="font-medium">
          {user?.name || user?.email}
        </span>
        <ChevronDown
          size={16}
          className={`transition-transform duration-200 ${
            isOpen ? 'rotate-180' : ''
          }`}
        />
      </button>

      {/* Dropdown Menu */}
      {isOpen && (
        <div className="absolute right-0 mt-2 w-56 bg-white dark:bg-gray-800 rounded-lg shadow-lg border border-gray-200 dark:border-gray-700 py-2 z-50">
          {/* User Info */}
          <div className="px-4 py-3 border-b border-gray-200 dark:border-gray-700">
            <p className="text-sm font-medium text-gray-900 dark:text-white">
              {user?.name}
            </p>
            <p className="text-sm text-gray-500 dark:text-gray-400">
              {user?.email}
            </p>
          </div>

          {/* Dark Mode Toggle */}
          <button
            onClick={() => {
              onToggleDarkMode()
              setIsOpen(false)
            }}
            className="flex items-center w-full px-4 py-3 text-sm text-gray-700 dark:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors duration-200"
          >
            {isDarkMode ? (
              <Sun size={16} className="mr-3 text-yellow-500" />
            ) : (
              <Moon size={16} className="mr-3 text-blue-500" />
            )}
            {isDarkMode ? 'Light Mode' : 'Dark Mode'}
          </button>

          {/* Plex Integration */}
          <div className="border-t border-gray-200 dark:border-gray-700 mt-2 pt-2">
            {plexLoading ? (
              <div className="flex items-center px-4 py-3 text-sm text-gray-500 dark:text-gray-400">
                <Loader2 size={16} className="mr-3 animate-spin" />
                Loading Plex status...
              </div>
            ) : plexStatus.connected ? (
              <div className="px-4 py-3">
                <div className="flex items-center justify-between mb-2">
                  <div className="flex items-center text-sm text-gray-700 dark:text-gray-200">
                    <CheckCircle size={16} className="mr-3 text-green-500" />
                    <div>
                      <div className="font-medium">Plex Connected</div>
                      <div className="text-sm font-medium text-gray-900 dark:text-white">
                        {plexStatus.friendlyName || plexStatus.username || 'Unknown User'}
                      </div>
                      <div className="text-xs text-gray-500 dark:text-gray-400">
                        {plexStatus.serverCount} server{plexStatus.serverCount !== 1 ? 's' : ''}
                      </div>
                    </div>
                  </div>
                </div>
                <button
                  onClick={async () => {
                    try {
                      await disconnectPlex()
                      setIsOpen(false)
                    } catch (err) {
                      // Handle error if needed
                    }
                  }}
                  className="w-full text-left text-xs text-red-600 hover:text-red-800 dark:text-red-400 dark:hover:text-red-300 transition-colors"
                >
                  Disconnect Plex
                </button>
              </div>
            ) : (
              <button
                onClick={() => {
                  setShowPlexModal(true)
                  setIsOpen(false)
                }}
                className="flex items-center w-full px-4 py-3 text-sm text-gray-700 dark:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors duration-200"
              >
                <Server size={16} className="mr-3 text-orange-500" />
                Connect Plex
              </button>
            )}
          </div>

          {/* Logout */}
          <div className="border-t border-gray-200 dark:border-gray-700 pt-2 px-4 py-2">
            <LogoutButton className="w-full bg-red-600 hover:bg-red-700 text-white font-semibold py-2 px-4 rounded-lg transition-colors duration-200" />
          </div>
        </div>
      )}
      </div>

      {/* Plex Connection Modal */}
      <PlexConnectionModal
        isOpen={showPlexModal}
        onClose={() => setShowPlexModal(false)}
        onSuccess={async () => {
          setShowPlexModal(false)
          // Explicitly refresh the status to ensure UI updates
          await refreshPlexStatus()
        }}
      />
    </>
  )
}