import { createContext, useContext, useState, useEffect, ReactNode } from 'react'
import { useAuth0 } from '@auth0/auth0-react'

interface DarkModeContextType {
  isDarkMode: boolean
  toggleDarkMode: () => void
  isLoading: boolean
}

const DarkModeContext = createContext<DarkModeContextType | undefined>(undefined)

interface DarkModeProviderProps {
  children: ReactNode
}

export function DarkModeProvider({ children }: DarkModeProviderProps) {
  const { isAuthenticated, getAccessTokenSilently } = useAuth0()
  const [isDarkMode, setIsDarkMode] = useState(false)
  const [isLoading, setIsLoading] = useState(true)

  // Load user preferences from API
  useEffect(() => {
    if (isAuthenticated) {
      loadUserPreferences()
    } else {
      setIsLoading(false)
    }
  }, [isAuthenticated])

  // Apply dark mode class to document
  useEffect(() => {
    if (isDarkMode) {
      document.documentElement.classList.add('dark')
    } else {
      document.documentElement.classList.remove('dark')
    }
  }, [isDarkMode])

  const loadUserPreferences = async () => {
    try {
      const token = await getAccessTokenSilently()
      const response = await fetch('/api/me/preferences', {
        headers: {
          Authorization: `Bearer ${token}`,
        },
      })

      if (response.ok) {
        const data = await response.json()
        setIsDarkMode(data.darkMode || false)
      } else {
        // If preferences don't exist, use default (light mode)
        setIsDarkMode(false)
      }
    } catch (error) {
      console.error('Failed to load user preferences:', error)
      setIsDarkMode(false) // Default to light mode on error
    } finally {
      setIsLoading(false)
    }
  }

  const saveUserPreferences = async (darkMode: boolean) => {
    try {
      const token = await getAccessTokenSilently()
      await fetch('/api/me/preferences', {
        method: 'PUT',
        headers: {
          Authorization: `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ darkMode }),
      })
    } catch (error) {
      console.error('Failed to save user preferences:', error)
    }
  }

  const toggleDarkMode = () => {
    const newDarkMode = !isDarkMode
    setIsDarkMode(newDarkMode)
    if (isAuthenticated) {
      saveUserPreferences(newDarkMode)
    }
  }

  return (
    <DarkModeContext.Provider value={{ isDarkMode, toggleDarkMode, isLoading }}>
      {children}
    </DarkModeContext.Provider>
  )
}

export function useDarkMode() {
  const context = useContext(DarkModeContext)
  if (context === undefined) {
    throw new Error('useDarkMode must be used within a DarkModeProvider')
  }
  return context
}