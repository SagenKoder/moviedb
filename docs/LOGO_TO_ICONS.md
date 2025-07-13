# Converting Your Logo to PWA Icons

This guide shows you how to convert your SVG logo into all the required PWA icon sizes.

## Quick Method (Recommended)

### 1. Use the Provided Script

```bash
# Make sure you're in the project root
cd /path/to/movie-db

# Run the icon generator script
./scripts/generate-icons.sh path/to/your/logo.svg

# With a background color (optional)
./scripts/generate-icons.sh path/to/your/logo.svg "#1f2937"
```

The script will automatically:
- Generate all 8 required icon sizes
- Place them in the correct directory (`web/public/icons/`)
- Handle transparent or colored backgrounds
- Provide a checklist of generated files

## Manual Methods

### Method 1: ImageMagick (Command Line)

First, install ImageMagick:
```bash
# macOS
brew install imagemagick

# Ubuntu/Debian
sudo apt install imagemagick

# Windows (via Chocolatey)
choco install imagemagick
```

Then generate icons:
```bash
cd web/public/icons/

# Convert SVG to PNG at different sizes
convert -background transparent -density 300 ../../../your-logo.svg -resize 512x512 -extent 512x512 -gravity center icon-512x512.png
convert -background transparent -density 300 ../../../your-logo.svg -resize 384x384 -extent 384x384 -gravity center icon-384x384.png
convert -background transparent -density 300 ../../../your-logo.svg -resize 192x192 -extent 192x192 -gravity center icon-192x192.png
convert -background transparent -density 300 ../../../your-logo.svg -resize 152x152 -extent 152x152 -gravity center icon-152x152.png
convert -background transparent -density 300 ../../../your-logo.svg -resize 144x144 -extent 144x144 -gravity center icon-144x144.png
convert -background transparent -density 300 ../../../your-logo.svg -resize 128x128 -extent 128x128 -gravity center icon-128x128.png
convert -background transparent -density 300 ../../../your-logo.svg -resize 96x96 -extent 96x96 -gravity center icon-96x96.png
convert -background transparent -density 300 ../../../your-logo.svg -resize 72x72 -extent 72x72 -gravity center icon-72x72.png
```

### Method 2: Online SVG to PNG Converter

1. **Upload your SVG** to one of these services:
   - [SVG to PNG Converter](https://svgtopng.com/)
   - [CloudConvert](https://cloudconvert.com/svg-to-png)
   - [Convertio](https://convertio.co/svg-png/)

2. **Generate each size** individually:
   - 512x512, 384x384, 192x192, 152x152, 144x144, 128x128, 96x96, 72x72

3. **Download and rename** files to match the required naming pattern:
   - `icon-72x72.png`, `icon-96x96.png`, etc.

4. **Place in directory**: `web/public/icons/`

### Method 3: Design Tools (Figma, Sketch, Photoshop)

1. **Import your SVG** into your design tool
2. **Create artboards** for each size:
   - 72×72, 96×96, 128×128, 144×144, 152×152, 192×192, 384×384, 512×512
3. **Center your logo** on each artboard
4. **Add padding** if needed (10-20% margin recommended)
5. **Export as PNG** with the correct names
6. **Place in** `web/public/icons/` directory

## Icon Design Considerations

### Size Guidelines
- **512×512**: Main icon, splash screens
- **192×192**: Standard home screen icon
- **152×152**: iOS home screen
- **144×144**: Windows tiles
- **128×128**: Large displays
- **96×96**: Medium displays  
- **72×72**: Small displays, notifications

### Design Tips

#### ✅ Do:
- **Keep it simple** - Should be recognizable at 72×72
- **Use high contrast** - Works on light and dark backgrounds
- **Add padding** - 10-20% margin for breathing room
- **Test at small sizes** - Zoom out to check readability
- **Use your brand colors** - Match your app's theme

#### ❌ Don't:
- **Include fine details** - Will disappear at small sizes
- **Use text** - Hard to read at icon sizes
- **Make it too complex** - Simple shapes work better
- **Forget about iOS** - Apple rounds corners automatically

### Background Options

#### Transparent Background (Recommended)
```bash
./scripts/generate-icons.sh logo.svg transparent
```
- Adapts to different home screen backgrounds
- Looks clean and modern
- Works well if your logo has good contrast

#### Solid Color Background
```bash
./scripts/generate-icons.sh logo.svg "#1f2937"
```
- Use your app's primary color
- Good if logo needs contrast
- Ensures consistent appearance

#### Gradient Background (Advanced)
For gradient backgrounds, you'll need to manually edit in a design tool.

## Verifying Your Icons

### Visual Check
```bash
# List generated icons
ls -la web/public/icons/

# Check file sizes (should be reasonable, not too large)
du -h web/public/icons/*
```

### Test in Browser
1. **Build your app**: `make build-prod`
2. **Run locally**: `./bin/moviedb`
3. **Open Chrome DevTools** → Application → Manifest
4. **Check icons** are loaded correctly

### Test Installation
1. **Deploy to a server** with HTTPS
2. **Open on iPhone** in Safari
3. **Add to Home Screen**
4. **Check icon** appears correctly

## Common Issues & Solutions

### Issue: Icons look blurry
**Solution**: Increase `-density` parameter in ImageMagick or use higher resolution source

### Issue: Logo is too small/cramped
**Solution**: Your SVG might have extra whitespace. Crop it or add padding in the conversion

### Issue: Background is wrong color
**Solution**: Check the `background` parameter in your conversion command

### Issue: Some icons missing
**Solution**: Run the script again or check for errors in the terminal output

## File Structure

After generation, you should have:
```
web/public/icons/
├── icon-72x72.png
├── icon-96x96.png
├── icon-128x128.png
├── icon-144x144.png
├── icon-152x152.png
├── icon-192x192.png
├── icon-384x384.png
└── icon-512x512.png
```

## Next Steps

1. **Generate your icons** using one of the methods above
2. **Build your app**: `make build-prod`
3. **Deploy with HTTPS**
4. **Test PWA installation** on mobile devices

Your Movie Database app will now have a professional icon set ready for PWA installation! 🎨📱