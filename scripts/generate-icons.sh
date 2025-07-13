#!/bin/bash

# PWA Icon Generator Script
# Converts an SVG logo to all required PWA icon sizes

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
SVG_FILE="$1"
OUTPUT_DIR="web/public/icons"
BACKGROUND_COLOR="${2:-transparent}"  # Default to transparent, or pass a color like "#ffffff"

# Required icon sizes
SIZES=(72 96 128 144 152 192 384 512)

# Help function
show_help() {
    echo "PWA Icon Generator"
    echo "Converts SVG logo to all required PWA icon sizes"
    echo ""
    echo "Usage: $0 <svg-file> [background-color]"
    echo ""
    echo "Arguments:"
    echo "  svg-file         Path to your SVG logo file"
    echo "  background-color Optional background color (default: transparent)"
    echo "                   Examples: transparent, #ffffff, #1f2937"
    echo ""
    echo "Examples:"
    echo "  $0 logo.svg"
    echo "  $0 logo.svg transparent"
    echo "  $0 logo.svg \"#ffffff\""
    echo ""
    echo "Requirements:"
    echo "  - ImageMagick (brew install imagemagick / apt install imagemagick)"
    echo "  - SVG file should be square or will be centered in square"
}

# Check if help requested
if [[ "$1" == "-h" || "$1" == "--help" || -z "$1" ]]; then
    show_help
    exit 0
fi

# Check if SVG file exists
if [[ ! -f "$SVG_FILE" ]]; then
    echo -e "${RED}Error: SVG file '$SVG_FILE' not found${NC}"
    exit 1
fi

# Check if ImageMagick is installed
if ! command -v convert &> /dev/null; then
    echo -e "${RED}Error: ImageMagick is not installed${NC}"
    echo "Install with:"
    echo "  macOS: brew install imagemagick"
    echo "  Ubuntu/Debian: sudo apt install imagemagick"
    echo "  CentOS/RHEL: sudo yum install ImageMagick"
    exit 1
fi

# Create output directory
mkdir -p "$OUTPUT_DIR"

echo -e "${GREEN}🎨 Generating PWA icons from SVG...${NC}"
echo "Source: $SVG_FILE"
echo "Output: $OUTPUT_DIR"
echo "Background: $BACKGROUND_COLOR"
echo ""

# Generate each icon size
for size in "${SIZES[@]}"; do
    output_file="$OUTPUT_DIR/icon-${size}x${size}.png"
    
    echo -e "${YELLOW}Generating ${size}x${size}...${NC}"
    
    # Convert SVG to PNG with specified size and background
    if [[ "$BACKGROUND_COLOR" == "transparent" ]]; then
        # Transparent background
        convert -background transparent \
                -density 300 \
                "$SVG_FILE" \
                -resize "${size}x${size}" \
                -extent "${size}x${size}" \
                -gravity center \
                "$output_file"
    else
        # Solid background color
        convert -background "$BACKGROUND_COLOR" \
                -density 300 \
                "$SVG_FILE" \
                -resize "${size}x${size}" \
                -extent "${size}x${size}" \
                -gravity center \
                "$output_file"
    fi
    
    if [[ -f "$output_file" ]]; then
        file_size=$(ls -lh "$output_file" | awk '{print $5}')
        echo -e "${GREEN}✓ Generated: $output_file ($file_size)${NC}"
    else
        echo -e "${RED}✗ Failed to generate: $output_file${NC}"
    fi
done

echo ""
echo -e "${GREEN}🎉 Icon generation complete!${NC}"
echo ""
echo "Generated icons:"
ls -la "$OUTPUT_DIR"

echo ""
echo -e "${YELLOW}Next steps:${NC}"
echo "1. Review the generated icons to ensure they look good at small sizes"
echo "2. Consider adding padding/margins if the logo looks cramped"
echo "3. Test the PWA installation after building: make build-prod"
echo "4. The icons will be automatically included in your PWA"

echo ""
echo -e "${GREEN}PWA Icon Checklist:${NC}"
for size in "${SIZES[@]}"; do
    icon_file="$OUTPUT_DIR/icon-${size}x${size}.png"
    if [[ -f "$icon_file" ]]; then
        echo -e "✅ ${size}x${size}"
    else
        echo -e "❌ ${size}x${size}"
    fi
done