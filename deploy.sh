#!/bin/bash
# Deployment script for DoH server on Oracle

set -e

DOMAIN="doh.lllllyccc.qzz.io"
INSTALL_DIR="/opt/doh-server"

echo "=== DNS-over-HTTPS Deployment ==="

# Install Go if not present
if ! command -v go &> /dev/null; then
    echo "Installing Go..."
    wget -q https://go.dev/dl/go1.22.0.linux-amd64.tar.gz
    sudo tar -C /usr/local -xzf go1.22.0.linux-amd64.tar.gz
    rm go1.22.0.linux-amd64.tar.gz
    echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
    export PATH=$PATH:/usr/local/go/bin
fi

echo "Go version: $(go version)"

# Clone repo
echo "Cloning repository..."
cd $INSTALL_DIR
git clone https://github.com/lllllyccc/Dns-over-Https.git .

# Build
echo "Building..."
go build -o doh-server ./cmd/doh-server

# Copy config
if [ ! -f config.yaml ]; then
    cp config.example.yaml config.yaml
    echo "Created config.yaml - please edit it with your settings"
fi

# Create data directory
mkdir -p data

# Install systemd service
echo "Installing systemd service..."
sudo cp doh-server.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable doh-server
sudo systemctl restart doh-server

echo ""
echo "=== Deployment Complete ==="
echo ""
echo "Next steps:"
echo "1. Edit /opt/doh-server/config.yaml with your settings"
echo "2. Add DNS A record: $DOMAIN -> $(curl -s ifconfig.me)"
echo "3. Get SSL cert: sudo certbot certonly --standalone -d $DOMAIN"
echo "4. Copy nginx-doh.conf to /etc/nginx/sites-available/doh"
echo "5. sudo ln -s /etc/nginx/sites-available/doh /etc/nginx/sites-enabled/"
echo "6. sudo nginx -t && sudo systemctl reload nginx"
echo ""
echo "Check status: sudo systemctl status doh-server"
echo "View logs: sudo journalctl -u doh-server -f"
