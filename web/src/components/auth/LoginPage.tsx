import { LoginButton } from './LoginButton'

export function LoginPage() {
  return (
    <div className="min-h-screen bg-gradient-to-br from-blue-900 to-purple-900 flex items-center justify-center px-4">
      <div className="max-w-md w-full space-y-8">
        <div className="text-center">
          <h1 className="text-4xl font-bold text-white mb-2">Sagens Movie Database</h1>
          <p className="text-blue-200 text-lg">
            Track your movies, discover new favorites, and connect with friends
          </p>
        </div>
        
        <div className="bg-white rounded-lg shadow-xl p-8 space-y-6">
          <div className="text-center">
            <h2 className="text-2xl font-semibold text-gray-900 mb-2">
              Welcome Back
            </h2>
            <p className="text-gray-600">
              Sign in to access your movie collection
            </p>
          </div>
          
          <div className="flex justify-center">
            <LoginButton />
          </div>
        </div>
        
        <div className="text-center">
          <p className="text-blue-200 text-sm">
            Secure authentication powered by Auth0
          </p>
        </div>
      </div>
    </div>
  )
}