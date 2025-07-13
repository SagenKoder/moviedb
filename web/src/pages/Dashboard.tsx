import { useState } from 'react'
import { Link } from 'react-router-dom'
import { useAuth0 } from '@auth0/auth0-react'
import { useLists } from '../hooks/useLists'
import { CreateListModal } from '../components/lists/CreateListModal'
import { Loader2, Plus } from 'lucide-react'

export function Dashboard() {
  const { user } = useAuth0()
  const { lists, loading, error, fetchLists } = useLists()
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false)

  const totalMovies = lists.reduce((sum, list) => sum + list.movie_count, 0)

  return (
    <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      <div className="mb-8">
        <h1 className="text-3xl font-bold text-gray-900 dark:text-white mb-2">
          Welcome back, {user?.given_name || user?.name || 'there'}!
        </h1>
        <p className="text-gray-600 dark:text-gray-300">
          Ready to discover and track your favorite movies?
        </p>
      </div>

      {/* List Stats */}
      <div className="mb-8">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-xl font-semibold text-gray-900 dark:text-white">Your Lists</h2>
          <button 
            onClick={() => setIsCreateModalOpen(true)}
            className="flex items-center gap-2 bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg font-medium transition-colors"
          >
            <Plus size={16} />
            Create List
          </button>
        </div>

        {loading ? (
          <div className="flex items-center justify-center py-8">
            <Loader2 className="animate-spin" size={32} />
          </div>
        ) : error ? (
          <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-4">
            <p className="text-red-800 dark:text-red-200">Failed to load lists: {error}</p>
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
            {/* Total Movies */}
            <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6 transition-colors duration-200">
              <Link 
                to={`/profile/${user?.sub}`}
                className="block hover:bg-gray-50 dark:hover:bg-gray-700/50 -m-6 p-6 rounded-lg transition-colors"
              >
                <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-2 hover:text-blue-600 dark:hover:text-blue-400 transition-colors">Total Movies</h3>
                <p className="text-3xl font-bold text-blue-600">{totalMovies}</p>
                <p className="text-gray-500 dark:text-gray-400 text-sm">Across all lists</p>
              </Link>
            </div>

            {/* Individual Lists */}
            {lists.slice(0, 3).map((list) => (
              <div key={list.id} className="bg-white dark:bg-gray-800 rounded-lg shadow p-6 transition-colors duration-200">
                <Link 
                  to={`/profile/${user?.sub}?list=${list.id}`}
                  className="block hover:bg-gray-50 dark:hover:bg-gray-700/50 -m-6 p-6 rounded-lg transition-colors"
                >
                  <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-2 truncate hover:text-blue-600 dark:hover:text-blue-400 transition-colors" title={list.name}>
                    {list.name}
                  </h3>
                  <p className="text-3xl font-bold text-green-600">{list.movie_count}</p>
                  <p className="text-gray-500 dark:text-gray-400 text-sm">
                    {list.movie_count === 1 ? 'movie' : 'movies'}
                  </p>
                </Link>
              </div>
            ))}

            {/* Show "More Lists" card if there are more than 3 */}
            {lists.length > 3 && (
              <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6 transition-colors duration-200">
                <Link 
                  to={`/profile/${user?.sub}`}
                  className="block hover:bg-gray-50 dark:hover:bg-gray-700/50 -m-6 p-6 rounded-lg transition-colors"
                >
                  <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-2 hover:text-blue-600 dark:hover:text-blue-400 transition-colors">More Lists</h3>
                  <p className="text-3xl font-bold text-purple-600">+{lists.length - 3}</p>
                  <p className="text-gray-500 dark:text-gray-400 text-sm">View all lists</p>
                </Link>
              </div>
            )}

            {/* Empty state for first-time users */}
            {lists.length === 0 && (
              <div className="col-span-full bg-gray-50 dark:bg-gray-800/50 rounded-lg border-2 border-dashed border-gray-300 dark:border-gray-600 p-8 text-center">
                <div className="text-gray-400 dark:text-gray-500 mb-4">
                  <div className="text-6xl mb-4">📋</div>
                </div>
                <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-2">No lists yet</h3>
                <p className="text-gray-500 dark:text-gray-400 mb-4">
                  Create your first list to start organizing your movies
                </p>
                <button 
                  onClick={() => setIsCreateModalOpen(true)}
                  className="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg font-medium transition-colors"
                >
                  Create Your First List
                </button>
              </div>
            )}
          </div>
        )}
      </div>

      {/* Recent Activity */}
      <div className="mt-8 bg-white dark:bg-gray-800 rounded-lg shadow transition-colors duration-200">
        <div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700">
          <h2 className="text-lg font-semibold text-gray-900 dark:text-white">Recent Activity</h2>
        </div>
        <div className="p-6">
          <p className="text-gray-500 dark:text-gray-400 text-center py-8">
            No activity yet. Start by searching for movies!
          </p>
        </div>
      </div>

      {/* Create List Modal */}
      <CreateListModal
        isOpen={isCreateModalOpen}
        onClose={() => setIsCreateModalOpen(false)}
        onSuccess={() => fetchLists()}
      />
    </div>
  )
}