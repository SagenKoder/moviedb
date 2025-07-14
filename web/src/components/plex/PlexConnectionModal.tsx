import { useState, useEffect } from 'react'
import { X, ExternalLink, Loader2, CheckCircle, AlertCircle } from 'lucide-react'
import { usePlex } from '../../hooks/usePlex'

interface PlexConnectionModalProps {
  isOpen: boolean
  onClose: () => void
  onSuccess?: () => void
}

export function PlexConnectionModal({ isOpen, onClose, onSuccess }: PlexConnectionModalProps) {
  const { startAuth, pollForAuth, error } = usePlex()
  const [step, setStep] = useState<'loading' | 'pin' | 'waiting' | 'success' | 'error'>('loading')
  const [pinData, setPinData] = useState<{ pinId: number; pinCode: string; expiresAt: string } | null>(null)
  const [timeLeft, setTimeLeft] = useState<number>(0)

  // Reset state when modal opens
  useEffect(() => {
    if (isOpen) {
      setStep('loading')
      setPinData(null)
      setTimeLeft(0)
      initializeAuth()
    }
  }, [isOpen])

  // Initialize authentication flow
  const initializeAuth = async () => {
    try {
      setStep('loading')
      const data = await startAuth()
      setPinData(data)
      setStep('pin')
      
      // Calculate time left
      const expiresAt = new Date(data.expiresAt).getTime()
      const now = Date.now()
      setTimeLeft(Math.max(0, Math.floor((expiresAt - now) / 1000)))
      
      // Don't start polling automatically - wait for user to click "I've entered the PIN"
    } catch (err) {
      setStep('error')
    }
  }

  // Start polling for PIN authorization
  const startPolling = async (pinId: number) => {
    try {
      setStep('waiting')
      await pollForAuth(pinId)
      setStep('success')
      setTimeout(() => {
        onSuccess?.()
        onClose()
      }, 2000)
    } catch (err) {
      setStep('error')
    }
  }

  // Countdown timer
  useEffect(() => {
    if (timeLeft > 0) {
      const timer = setTimeout(() => {
        setTimeLeft(timeLeft - 1)
      }, 1000)
      return () => clearTimeout(timer)
    } else if (timeLeft === 0 && step === 'waiting') {
      setStep('error')
    }
  }, [timeLeft, step])

  const formatTime = (seconds: number): string => {
    const mins = Math.floor(seconds / 60)
    const secs = seconds % 60
    return `${mins}:${secs.toString().padStart(2, '0')}`
  }

  const handleOpenPlexLink = () => {
    window.open('https://plex.tv/link', '_blank', 'noopener,noreferrer')
  }

  if (!isOpen) return null

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4">
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow-xl max-w-md w-full max-h-[90vh] overflow-y-auto">
        {/* Header */}
        <div className="flex items-center justify-between p-6 border-b border-gray-200 dark:border-gray-700">
          <h2 className="text-xl font-semibold text-gray-900 dark:text-white">
            Connect to Plex
          </h2>
          <button
            onClick={onClose}
            className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 transition-colors"
          >
            <X size={24} />
          </button>
        </div>

        {/* Content */}
        <div className="p-6">
          {step === 'loading' && (
            <div className="text-center">
              <Loader2 className="mx-auto mb-4 animate-spin" size={48} />
              <p className="text-gray-600 dark:text-gray-400">
                Preparing your Plex connection...
              </p>
            </div>
          )}

          {step === 'pin' && pinData && (
            <div className="text-center space-y-4">
              <div className="bg-orange-500 text-white p-6 rounded-lg">
                <div className="text-sm font-medium mb-2">Enter this PIN at Plex:</div>
                <div className="text-4xl font-bold tracking-wider">{pinData.pinCode}</div>
              </div>
              
              <button
                onClick={handleOpenPlexLink}
                className="inline-flex items-center px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white font-medium rounded-lg transition-colors"
              >
                <ExternalLink size={16} className="mr-2" />
                Open plex.tv/link
              </button>
              
              <p className="text-sm text-gray-600 dark:text-gray-400">
                Click the button above to open Plex in a new tab, then enter the PIN.
              </p>
              
              <button
                onClick={() => startPolling(pinData.pinId)}
                className="w-full mt-4 px-4 py-2 bg-green-600 hover:bg-green-700 text-white font-medium rounded-lg transition-colors"
              >
                I've entered the PIN
              </button>
            </div>
          )}

          {step === 'waiting' && (
            <div className="text-center space-y-4">
              <Loader2 className="mx-auto animate-spin" size={48} />
              <div>
                <p className="text-lg font-medium text-gray-900 dark:text-white mb-2">
                  Waiting for authorization...
                </p>
                <p className="text-gray-600 dark:text-gray-400 mb-4">
                  Please authorize the connection in your Plex account
                </p>
                {timeLeft > 0 && (
                  <div className="text-sm text-gray-500 dark:text-gray-400">
                    Time remaining: {formatTime(timeLeft)}
                  </div>
                )}
              </div>
            </div>
          )}

          {step === 'success' && (
            <div className="text-center space-y-4">
              <CheckCircle className="mx-auto text-green-500" size={48} />
              <div>
                <p className="text-lg font-medium text-gray-900 dark:text-white mb-2">
                  Successfully connected!
                </p>
                <p className="text-gray-600 dark:text-gray-400">
                  Your Plex account is now linked to MovieDB
                </p>
              </div>
            </div>
          )}

          {step === 'error' && (
            <div className="text-center space-y-4">
              <AlertCircle className="mx-auto text-red-500" size={48} />
              <div>
                <p className="text-lg font-medium text-gray-900 dark:text-white mb-2">
                  Connection failed
                </p>
                <p className="text-gray-600 dark:text-gray-400 mb-4">
                  {error || 'Failed to connect to Plex. Please try again.'}
                </p>
                <button
                  onClick={initializeAuth}
                  className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white font-medium rounded-lg transition-colors"
                >
                  Try Again
                </button>
              </div>
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="px-6 py-4 bg-gray-50 dark:bg-gray-700 rounded-b-lg">
          <p className="text-xs text-gray-500 dark:text-gray-400 text-center">
            This will allow MovieDB to sync with your Plex library and watchlists
          </p>
        </div>
      </div>
    </div>
  )
}