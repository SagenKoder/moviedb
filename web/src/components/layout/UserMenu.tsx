import { useState, useRef, useEffect } from 'react'
import { useAuth0 } from '@auth0/auth0-react'
import { ChevronDown, Moon, Sun } from 'lucide-react'
import { LogoutButton } from '../auth/LogoutButton'

interface UserMenuProps {
  isDarkMode: boolean
  onToggleDarkMode: () => void
}

export function UserMenu({ isDarkMode, onToggleDarkMode }: UserMenuProps) {
  const { user } = useAuth0()
  const [isOpen, setIsOpen] = useState(false)
  const menuRef = useRef<HTMLDivElement>(null)

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

          {/* Logout */}
          <div className="border-t border-gray-200 dark:border-gray-700 pt-2 px-4 py-2">
            <LogoutButton className="w-full bg-red-600 hover:bg-red-700 text-white font-semibold py-2 px-4 rounded-lg transition-colors duration-200" />
          </div>
        </div>
      )}
    </div>
  )
}