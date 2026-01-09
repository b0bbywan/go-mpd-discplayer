#!/bin/bash
set -e

# Extract package name and version from cmd/player.go
PACKAGE_NAME=$(grep -oP 'AppName\s*=\s*"\K[^"]+' cmd/player.go 2>/dev/null || echo "mpd-discplayer")
VERSION=$(grep -oP 'AppVersion\s*=\s*"\K[^"]+' cmd/player.go 2>/dev/null || echo "0.7")

# Auto-detect architecture
ARCH=$(dpkg --print-architecture)
MAINTAINER="Mathieu Réquillart <mathieu.requillart@gmail.com>"
DESCRIPTION="MPD disc player daemon for CD and USB automation"

# Build directories
BUILD_DIR="build"
PKG_DIR="${BUILD_DIR}/${PACKAGE_NAME}_${VERSION}_${ARCH}"

echo "📦 Building package ${PACKAGE_NAME} v${VERSION} for ${ARCH}"

# Cleanup
rm -rf ${BUILD_DIR}
mkdir -p ${BUILD_DIR}

# Native Go binary compilation
echo "🔨 Compiling binary..."
go build -o ${PACKAGE_NAME} .

# Create package structure
echo "📁 Creating package structure..."
mkdir -p ${PKG_DIR}/DEBIAN
mkdir -p ${PKG_DIR}/usr/bin
mkdir -p ${PKG_DIR}/usr/lib/systemd/user
mkdir -p ${PKG_DIR}/usr/share/${PACKAGE_NAME}

# Copy binary
cp ${PACKAGE_NAME} ${PKG_DIR}/usr/bin/
chmod 755 ${PKG_DIR}/usr/bin/${PACKAGE_NAME}

# Copy example configuration file
if [ -f "share/config.yaml" ]; then
  cp share/config.yaml ${PKG_DIR}/usr/share/${PACKAGE_NAME}/config.yml.example
else
  echo "⚠️  Warning: share/config.yaml not found, config file skipped"
fi

# Copy systemd user service
if [ -f "share/${PACKAGE_NAME}.service" ]; then
  cp "share/${PACKAGE_NAME}.service" "${PKG_DIR}/usr/lib/systemd/user/${PACKAGE_NAME}.service"
else
  echo "⚠️  Warning: share/${PACKAGE_NAME}.service not found, systemd unit file skipped"
fi

# Create control file
cat > ${PKG_DIR}/DEBIAN/control << EOF
Package: ${PACKAGE_NAME}
Version: ${VERSION}
Section: sound
Priority: optional
Architecture: ${ARCH}
Maintainer: ${MAINTAINER}
Description: ${DESCRIPTION}
 A daemon that monitors CD drive status and automatically
 plays inserted discs via MPD (Music Player Daemon).
Depends: mpd, libcdio-paranoia2t64, libdiscid0, libgudev-1.0-0
EOF

# Post-installation script
cat > ${PKG_DIR}/DEBIAN/postinst << 'EOF'
#!/bin/bash
set -e

echo "✓ mpd-discplayer installed"
echo ""
echo "Example configuration file available at:"
echo "  /usr/share/mpd-discplayer/config.yml.example"
echo ""
echo "To configure:"
echo "  mkdir -p ~/.config/mpd-discplayer"
echo "  cp /usr/share/mpd-discplayer/config.yml.example ~/.config/mpd-discplayer/config.yml"
echo "  # Edit ~/.config/mpd-discplayer/config.yml as needed"
echo ""
echo "To enable the user service:"
echo "  systemctl --user enable mpd-discplayer.service"
echo "  systemctl --user start mpd-discplayer.service"
echo ""
echo "To check status:"
echo "  systemctl --user status mpd-discplayer.service"

exit 0
EOF

chmod 755 ${PKG_DIR}/DEBIAN/postinst

# Pre-removal script
cat > ${PKG_DIR}/DEBIAN/prerm << 'EOF'
#!/bin/bash
set -e

# Stop user service if running (best effort)
systemctl --user stop mpd-discplayer.service 2>/dev/null || true
systemctl --user disable mpd-discplayer.service 2>/dev/null || true

exit 0
EOF

chmod 755 ${PKG_DIR}/DEBIAN/prerm

# Build .deb package
echo "🔧 Building .deb file..."
dpkg-deb --build --root-owner-group ${PKG_DIR}

# Move .deb to current directory
mv ${BUILD_DIR}/${PACKAGE_NAME}_${VERSION}_${ARCH}.deb .

echo "✅ Package created: ${PACKAGE_NAME}_${VERSION}_${ARCH}.deb"
echo ""
echo "To install: sudo dpkg -i ${PACKAGE_NAME}_${VERSION}_${ARCH}.deb"
echo "To remove: sudo apt remove ${PACKAGE_NAME}"

# Cleanup
rm -rf ${BUILD_DIR}
rm ${PACKAGE_NAME}
