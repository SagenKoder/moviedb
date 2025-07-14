import { ExternalLink, Play, ShoppingCart, CreditCard, Tv, Server } from 'lucide-react'
import { useWatchProviders, WatchProvider } from '../../hooks/useWatchProviders'

interface WatchProvidersProps {
  tmdbId: number
  region?: string
}

export function WatchProviders({ tmdbId, region = 'US' }: WatchProvidersProps) {
  const { watchProviders, loading, error } = useWatchProviders(tmdbId, region)

  if (loading) {
    return (
      <div className="space-y-3">
        <h3 className="text-lg font-semibold text-gray-900 dark:text-white">Watch Now</h3>
        <div className="animate-pulse">
          <div className="h-4 bg-gray-300 dark:bg-gray-600 rounded w-3/4 mb-2"></div>
          <div className="h-20 bg-gray-300 dark:bg-gray-600 rounded"></div>
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="space-y-3">
        <h3 className="text-lg font-semibold text-gray-900 dark:text-white">Watch Now</h3>
        <div className="text-sm text-gray-500 dark:text-gray-400">
          Unable to load watch options
        </div>
      </div>
    )
  }

  if (!watchProviders || watchProviders.providers.length === 0) {
    return (
      <div className="space-y-3">
        <h3 className="text-lg font-semibold text-gray-900 dark:text-white">Watch Now</h3>
        <div className="text-sm text-gray-500 dark:text-gray-400">
          No streaming options available in {region}
        </div>
      </div>
    )
  }

  const getProviderIcon = (providerType: WatchProvider['providerType']) => {
    switch (providerType) {
      case 'flatrate':
        return <Play size={16} />
      case 'rent':
        return <ShoppingCart size={16} />
      case 'buy':
        return <CreditCard size={16} />
      case 'free':
        return <Tv size={16} />
      case 'plex':
        return <Server size={16} />
      default:
        return <ExternalLink size={16} />
    }
  }

  const getProviderTypeLabel = (providerType: WatchProvider['providerType']) => {
    switch (providerType) {
      case 'flatrate':
        return 'Stream'
      case 'rent':
        return 'Rent'
      case 'buy':
        return 'Buy'
      case 'free':
        return 'Free'
      case 'plex':
        return 'Your Library'
      default:
        return 'Watch'
    }
  }

  const getProviderTypeColor = (providerType: WatchProvider['providerType']) => {
    switch (providerType) {
      case 'flatrate':
        return 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200'
      case 'rent':
        return 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200'
      case 'buy':
        return 'bg-purple-100 text-purple-800 dark:bg-purple-900 dark:text-purple-200'
      case 'free':
        return 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-200'
      case 'plex':
        return 'bg-orange-100 text-orange-800 dark:bg-orange-900 dark:text-orange-200'
      default:
        return 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-200'
    }
  }

  // Group providers by type
  const groupedProviders = watchProviders.providers.reduce((acc, provider) => {
    if (!acc[provider.providerType]) {
      acc[provider.providerType] = []
    }
    acc[provider.providerType].push(provider)
    return acc
  }, {} as Record<WatchProvider['providerType'], WatchProvider[]>)

  // Order of provider types for display
  const providerTypeOrder: WatchProvider['providerType'][] = ['plex', 'flatrate', 'free', 'rent', 'buy']

  return (
    <div className="space-y-4">
      <h3 className="text-lg font-semibold text-gray-900 dark:text-white">Watch Now</h3>
      
      {providerTypeOrder.map(providerType => {
        const providers = groupedProviders[providerType]
        if (!providers || providers.length === 0) return null

        return (
          <div key={providerType} className="space-y-2">
            <div className="flex items-center gap-2">
              <span className={`inline-flex items-center gap-1 px-2 py-1 rounded-full text-xs font-medium ${getProviderTypeColor(providerType)}`}>
                {getProviderIcon(providerType)}
                {getProviderTypeLabel(providerType)}
              </span>
            </div>
            
            <div className="grid grid-cols-2 sm:grid-cols-3 gap-3">
              {providers.map((provider, index) => (
                <div
                  key={`${providerType}-${index}`}
                  className={`flex items-center gap-3 p-3 rounded-lg border border-gray-200 dark:border-gray-700 hover:border-gray-300 dark:hover:border-gray-600 transition-colors ${
                    provider.link ? 'cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-800' : ''
                  }`}
                  onClick={() => provider.link && window.open(provider.link, '_blank')}
                >
                  {provider.logoPath ? (
                    <img
                      src={provider.logoPath}
                      alt={provider.name}
                      className="w-8 h-8 rounded"
                      onError={(e) => {
                        e.currentTarget.style.display = 'none'
                      }}
                    />
                  ) : (
                    <div className="w-8 h-8 bg-gray-200 dark:bg-gray-700 rounded flex items-center justify-center">
                      {getProviderIcon(providerType)}
                    </div>
                  )}
                  
                  <div className="flex-1 min-w-0">
                    <div className="text-sm font-medium text-gray-900 dark:text-white truncate">
                      {provider.name}
                    </div>
                    {provider.price && (
                      <div className="text-xs text-gray-500 dark:text-gray-400">
                        {provider.price}
                      </div>
                    )}
                    {providerType === 'plex' && (
                      <div className="text-xs text-gray-500 dark:text-gray-400">
                        {provider.libraryName && (
                          <span>{provider.libraryName}</span>
                        )}
                        {provider.libraryName && provider.plexServer && (
                          <span> • </span>
                        )}
                        {provider.plexServer && (
                          <span>{provider.plexServer}</span>
                        )}
                      </div>
                    )}
                  </div>
                  
                  {provider.link && (
                    <ExternalLink size={14} className="text-gray-400 flex-shrink-0" />
                  )}
                </div>
              ))}
            </div>
          </div>
        )
      })}

      {watchProviders.tmdbLink && (
        <div className="pt-2 border-t border-gray-200 dark:border-gray-700">
          <a
            href={watchProviders.tmdbLink}
            target="_blank"
            rel="noopener noreferrer"
            className="inline-flex items-center gap-1 text-sm text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300"
          >
            <ExternalLink size={14} />
            View more options
          </a>
        </div>
      )}

      <div className="text-xs text-gray-500 dark:text-gray-400">
        Data from The Movie Database • Updated {new Date(watchProviders.cachedAt).toLocaleDateString()}
      </div>
    </div>
  )
}