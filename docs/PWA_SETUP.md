# PWA Setup Guide

This guide covers setting up the Movie Database as a Progressive Web App (PWA) that can be installed on mobile devices, including iPhones.

## What's Included

The PWA setup includes:
- ✅ **Web App Manifest** - App metadata and icon configuration
- ✅ **Service Worker** - Offline functionality and caching
- ✅ **iOS Specific Meta Tags** - iPhone/iPad optimization
- ✅ **App Icons** - Various sizes for different devices
- ✅ **Installable** - "Add to Home Screen" functionality

## Required Files

### 1. Web App Manifest (`/web/public/manifest.json`)
- App name, description, and display settings
- Icon configurations for all sizes
- Theme colors and orientation preferences
- Start URL and display mode

### 2. Service Worker (`/web/public/sw.js`)
- Offline caching strategy
- Network-first for API calls
- Cache-first for static assets
- Background sync capability

### 3. App Icons (`/web/public/icons/`)
You need to create icons in these sizes:
- `icon-72x72.png` - Small devices
- `icon-96x96.png` - Medium devices
- `icon-128x128.png` - Large devices
- `icon-144x144.png` - High DPI devices
- `icon-152x152.png` - iOS home screen
- `icon-192x192.png` - Standard PWA icon
- `icon-384x384.png` - Large displays
- `icon-512x512.png` - Maximum size/splash screens

## Creating App Icons

### Method 1: Using a Design Tool
1. Create a 512x512px icon in Figma, Photoshop, or similar
2. Design should be simple, recognizable, and work at small sizes
3. Export as PNG with transparent background
4. Use an icon generator to create all sizes

### Method 2: Using Online Generator
1. Create/find a 512x512px base icon
2. Use PWA Icon Generator (https://www.pwabuilder.com/imageGenerator)
3. Upload your base icon
4. Download the generated icon set
5. Place icons in `/web/public/icons/` directory

### Method 3: Using ImageMagick (Command Line)
```bash
# Install ImageMagick
brew install imagemagick  # MacOS
sudo apt install imagemagick  # Linux

# Generate all icon sizes from a base 512x512 icon
cd web/public/icons/

# Create all required sizes
convert icon-512x512.png -resize 384x384 icon-384x384.png
convert icon-512x512.png -resize 192x192 icon-192x192.png
convert icon-512x512.png -resize 152x152 icon-152x152.png
convert icon-512x512.png -resize 144x144 icon-144x144.png
convert icon-512x512.png -resize 128x128 icon-128x128.png
convert icon-512x512.png -resize 96x96 icon-96x96.png
convert icon-512x512.png -resize 72x72 icon-72x72.png
```

## Icon Design Guidelines

### Visual Requirements
- **Simple & Recognizable** - Works at 72x72px
- **High Contrast** - Clear against various backgrounds
- **No Text** - Icons should work without text
- **Centered Design** - Account for rounded corners on iOS

### Technical Requirements
- **Format**: PNG with transparency
- **Background**: Transparent or solid color
- **Padding**: 10-20% padding for safe area
- **Colors**: Match your app's theme

### Suggested Design
For the Movie Database app, consider:
- 🎬 Film reel or camera icon
- 🎭 Theater masks
- 📽️ Movie projector
- 🍿 Popcorn icon
- 📱 Device with movie interface

## iOS Installation Process

### How Users Install on iPhone:

1. **Open Safari** (must use Safari, not Chrome/Firefox)
2. **Navigate to your app** (e.g., https://yourdomain.com)
3. **Tap Share button** (square with arrow up)
4. **Scroll down** and tap "Add to Home Screen"
5. **Edit name** if desired (defaults to "Movie DB")
6. **Tap "Add"** to install

### What Users See:
- App icon on home screen
- App opens in full-screen mode (no Safari UI)
- Looks and feels like a native app
- Works offline with cached content

## Testing PWA Installation

### Desktop Testing
1. **Chrome DevTools**:
   - Open DevTools → Application tab
   - Check "Manifest" and "Service Workers" sections
   - Look for installation prompts

2. **Lighthouse Audit**:
   ```bash
   # Install Lighthouse CLI
   npm install -g lighthouse
   
   # Run PWA audit
   lighthouse https://localhost:3000 --view
   ```

### Mobile Testing
1. **iPhone Safari**:
   - Open app in Safari
   - Look for "Add to Home Screen" option
   - Install and test offline functionality

2. **Android Chrome**:
   - Should show "Install App" banner
   - Can install via Chrome menu

## Production Deployment

### 1. Build with PWA Assets
```bash
# Ensure all PWA files are in place
ls web/public/manifest.json
ls web/public/sw.js
ls web/public/icons/

# Build production app
make build-prod
```

### 2. Server Configuration
Ensure your server serves PWA files with correct headers:

**Nginx**:
```nginx
# Serve manifest with correct MIME type
location /manifest.json {
    add_header Content-Type application/manifest+json;
    add_header Cache-Control "public, max-age=86400";
}

# Service worker should not be cached
location /sw.js {
    add_header Content-Type application/javascript;
    add_header Cache-Control "no-cache, no-store, must-revalidate";
}

# Icons with long cache
location /icons/ {
    add_header Cache-Control "public, max-age=31536000";
}
```

### 3. HTTPS Requirement
PWAs require HTTPS in production:
- Service Workers only work over HTTPS
- "Add to Home Screen" requires secure context
- Use Let's Encrypt for free SSL certificates

## Troubleshooting

### Common Issues

| Issue | Solution |
|-------|----------|
| **"Add to Home Screen" missing** | Must use Safari on iOS, ensure HTTPS |
| **Icons not showing** | Check file paths and sizes in manifest.json |
| **Service Worker fails** | Check browser console for errors |
| **App doesn't work offline** | Verify service worker caching strategy |
| **iOS status bar issues** | Adjust `apple-mobile-web-app-status-bar-style` |

### Debug Commands

```bash
# Check if service worker is registered
# In browser console:
navigator.serviceWorker.getRegistrations().then(console.log)

# Check manifest
# In browser console:
console.log(document.querySelector('link[rel="manifest"]'))

# Test offline
# In DevTools: Application → Service Workers → Offline checkbox
```

### Validation Tools

1. **PWA Builder**: https://www.pwabuilder.com/
2. **Web App Manifest Generator**: https://app-manifest.firebaseapp.com/
3. **Lighthouse PWA Audit**: Built into Chrome DevTools

## Advanced Features (Optional)

### Push Notifications
```javascript
// Request permission
Notification.requestPermission().then(permission => {
  if (permission === 'granted') {
    // Subscribe to push notifications
  }
});
```

### Background Sync
```javascript
// Register background sync
navigator.serviceWorker.ready.then(registration => {
  return registration.sync.register('background-sync');
});
```

### Install Prompt
```javascript
// Capture install prompt event
window.addEventListener('beforeinstallprompt', (e) => {
  e.preventDefault();
  // Show custom install button
});
```

## File Checklist

Before deploying, ensure you have:
- ✅ `/web/public/manifest.json` - Web app manifest
- ✅ `/web/public/sw.js` - Service worker
- ✅ `/web/public/icons/icon-*.png` - All icon sizes
- ✅ Updated `/web/index.html` - PWA meta tags
- ✅ Service worker registration in `/web/src/main.tsx`
- ✅ HTTPS enabled in production
- ✅ Proper server headers for PWA files

Your Movie Database app is now ready to be installed as a PWA on iPhones and other mobile devices! 📱🎬