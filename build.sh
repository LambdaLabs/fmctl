set -e

echo "Building fmctl CLI tool..."

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed. Please install Go 1.21 or later."
    exit 1
fi

# Build the CLI tool
# Use -tags=dev if FM SDK library is not available
# Remove -tags=dev for production build with FM library
go build -tags=dev -o fmctl ./cmd/fmctl

echo "Build complete! Binary: ./fmctl"
echo ""
echo "Usage: ./fmctl --help"
echo ""
echo "For production build with FM SDK library:"
echo "  go build -o fmctl ./cmd/fmctl"
