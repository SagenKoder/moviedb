import { BrowserRouter as Router, Routes, Route } from 'react-router-dom'
import { Auth0Provider } from '@auth0/auth0-react'
import { ProtectedRoute } from './components/auth/ProtectedRoute'
import { Header } from './components/layout/Header'
import { Footer } from './components/layout/Footer'
import { Dashboard } from './pages/Dashboard'
import { Movies } from './pages/Movies'
import { Community } from './pages/Community'
import { Profile } from './pages/Profile'
import { DarkModeProvider } from './contexts/DarkModeContext'

function App() {
  return (
    <Auth0Provider
      domain={import.meta.env.VITE_AUTH0_DOMAIN}
      clientId={import.meta.env.VITE_AUTH0_CLIENT_ID}
      authorizationParams={{
        redirect_uri: window.location.origin,
        audience: import.meta.env.VITE_AUTH0_AUDIENCE
      }}
    >
      <DarkModeProvider>
        <Router>
          <ProtectedRoute>
            <div className="min-h-screen bg-gray-100 dark:bg-gray-900 transition-colors duration-200 flex flex-col">
              <Header />
              <main className="flex-1">
                <Routes>
                  <Route path="/" element={<Dashboard />} />
                  <Route path="/dashboard" element={<Dashboard />} />
                  <Route path="/movies" element={<Movies />} />
                  <Route path="/community" element={<Community />} />
                  <Route path="/profile" element={<Profile />} />
                  <Route path="/profile/:userId" element={<Profile />} />
                </Routes>
              </main>
              <Footer />
            </div>
          </ProtectedRoute>
        </Router>
      </DarkModeProvider>
    </Auth0Provider>
  )
}

export default App