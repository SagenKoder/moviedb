# PWA Icons Directory

This directory should contain the app icons for the Progressive Web App (PWA) functionality.

## Required Icon Sizes

Create PNG icons in the following sizes:

- `icon-72x72.png` - Small devices, notification badges
- `icon-96x96.png` - Medium devices 
- `icon-128x128.png` - Large devices
- `icon-144x144.png` - High DPI devices, Windows tiles
- `icon-152x152.png` - iOS home screen icon
- `icon-192x192.png` - Standard PWA icon, Android home screen
- `icon-384x384.png` - Large displays, splash screens
- `icon-512x512.png` - Maximum size, splash screens, maskable

## Quick Icon Generation

### Using ImageMagick (from a 512x512 base icon):

```bash
# Navigate to this directory
cd web/public/icons/

# Place your base icon as icon-512x512.png, then run:
convert icon-512x512.png -resize 384x384 icon-384x384.png
convert icon-512x512.png -resize 192x192 icon-192x192.png
convert icon-512x512.png -resize 152x152 icon-152x152.png
convert icon-512x512.png -resize 144x144 icon-144x144.png
convert icon-512x512.png -resize 128x128 icon-128x128.png
convert icon-512x512.png -resize 96x96 icon-96x96.png
convert icon-512x512.png -resize 72x72 icon-72x72.png
```

### Online Generators:
- [PWA Builder Image Generator](https://www.pwabuilder.com/imageGenerator)
- [App Icon Generator](https://appicon.co/)
- [Favicon Generator](https://favicon.io/)

## Icon Design Tips

- Use a simple, recognizable design
- Ensure it works at 72x72px (smallest size)
- Consider a movie-themed icon (🎬📽️🎭🍿)
- Use your app's primary colors
- Include some padding for rounded corners on iOS
- Test on both light and dark backgrounds