import { useState, useEffect } from 'react'
import { X, Trash2, Loader2 } from 'lucide-react'
import { useLists, List } from '../../hooks/useLists'

interface EditListModalProps {
  list: List | null
  isOpen: boolean
  onClose: () => void
  onUpdate: () => void
}

export function EditListModal({ list, isOpen, onClose, onUpdate }: EditListModalProps) {
  const { updateList, deleteList, loading } = useLists()
  const [formData, setFormData] = useState({
    name: '',
    description: '',
    is_public: true
  })
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)
  const [deleting, setDeleting] = useState(false)

  // Reset form when list changes
  useEffect(() => {
    if (list) {
      setFormData({
        name: list.name,
        description: list.description || '',
        is_public: list.is_public
      })
    }
    setShowDeleteConfirm(false)
  }, [list])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!list) return

    try {
      await updateList(list.id, formData)
      onUpdate()
      onClose()
    } catch (error) {
      console.error('Failed to update list:', error)
      // Error is handled by the hook
    }
  }

  const handleDelete = async () => {
    if (!list) return

    setDeleting(true)
    try {
      await deleteList(list.id)
      onUpdate()
      onClose()
    } catch (error) {
      console.error('Failed to delete list:', error)
      // Error is handled by the hook
    } finally {
      setDeleting(false)
    }
  }

  if (!isOpen || !list) return null

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50">
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow-xl w-full max-w-md transition-colors duration-200">
        {/* Header */}
        <div className="flex items-center justify-between p-6 border-b border-gray-200 dark:border-gray-700">
          <h2 className="text-xl font-semibold text-gray-900 dark:text-white">
            Edit List
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
          {!showDeleteConfirm ? (
            <form onSubmit={handleSubmit} className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                  List Name
                </label>
                <input
                  type="text"
                  value={formData.name}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  placeholder="Enter list name"
                  required
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                  Description
                </label>
                <textarea
                  value={formData.description}
                  onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  placeholder="Enter list description (optional)"
                  rows={3}
                />
              </div>

              <div>
                <label className="flex items-center space-x-2">
                  <input
                    type="checkbox"
                    checked={formData.is_public}
                    onChange={(e) => setFormData({ ...formData, is_public: e.target.checked })}
                    className="rounded border-gray-300 dark:border-gray-600 text-blue-600 focus:ring-blue-500"
                  />
                  <span className="text-sm text-gray-700 dark:text-gray-300">
                    Make this list public
                  </span>
                </label>
              </div>

              <div className="flex items-center justify-between pt-4">
                <button
                  type="button"
                  onClick={() => setShowDeleteConfirm(true)}
                  className="flex items-center space-x-2 px-4 py-2 text-red-600 hover:text-red-700 dark:text-red-400 dark:hover:text-red-300 transition-colors"
                >
                  <Trash2 size={16} />
                  <span>Delete List</span>
                </button>

                <div className="flex space-x-3">
                  <button
                    type="button"
                    onClick={onClose}
                    className="px-4 py-2 text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200 transition-colors"
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    disabled={loading || !formData.name.trim()}
                    className="flex items-center space-x-2 bg-blue-600 hover:bg-blue-700 disabled:bg-blue-400 text-white px-6 py-2 rounded-lg font-medium transition-colors"
                  >
                    {loading ? (
                      <>
                        <Loader2 size={16} className="animate-spin" />
                        <span>Updating...</span>
                      </>
                    ) : (
                      'Update List'
                    )}
                  </button>
                </div>
              </div>
            </form>
          ) : (
            /* Delete Confirmation */
            <div className="text-center">
              <div className="mb-4">
                <Trash2 size={48} className="mx-auto text-red-500 mb-4" />
                <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-2">
                  Delete "{list.name}"?
                </h3>
                <p className="text-gray-600 dark:text-gray-400">
                  This action cannot be undone. All movies in this list will be removed.
                </p>
              </div>

              <div className="flex items-center justify-center space-x-3">
                <button
                  onClick={() => setShowDeleteConfirm(false)}
                  className="px-4 py-2 text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200 transition-colors"
                >
                  Cancel
                </button>
                <button
                  onClick={handleDelete}
                  disabled={deleting}
                  className="flex items-center space-x-2 bg-red-600 hover:bg-red-700 disabled:bg-red-400 text-white px-6 py-2 rounded-lg font-medium transition-colors"
                >
                  {deleting ? (
                    <>
                      <Loader2 size={16} className="animate-spin" />
                      <span>Deleting...</span>
                    </>
                  ) : (
                    <>
                      <Trash2 size={16} />
                      <span>Delete List</span>
                    </>
                  )}
                </button>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}