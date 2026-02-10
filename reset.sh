
#!/bin/sh

# 1. Create directories
mkdir -p cmd/vac-format
mkdir -p internal/crypto
mkdir -p internal/player
mkdir -p internal/metadata
mkdir -p internal/utils
mkdir -p pkg/vacfile
mkdir -p testdata
mkdir -p bin

# 2. Move main.go to cmd
mv main.go cmd/vac-format/

# 3. Move crypto-related files
mv crypto.go encode.go internal/crypto/ 2>/dev/null

# 4. Move player-related file
mv player.go internal/player/ 2>/dev/null

# 5. Move metadata files
mv metadata.go info.go internal/metadata/ 2>/dev/null

# 6. Move utils
mv utils.go internal/utils/ 2>/dev/null

# 7. Move vacfile.go to pkg
mv vacfile.go pkg/vacfile/ 2>/dev/null

# 8. Move test files to testdata
mv test.flac test-8.flac test.vac testdata/ 2>/dev/null

echo "Reorganization complete!"
