#!/bin/bash

# Cleaning Scheduler - Digital Ocean Setup Script
# Run this on your Ubuntu droplet to quickly deploy with SSL

set -e

echo "=================================="
echo "Cleaning Scheduler Setup for Digital Ocean"
echo "=================================="
echo ""

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo "Please run as root (use: sudo bash setup-digitalocean.sh)"
    exit 1
fi

# Get domain from user
read -p "Enter your domain name (e.g., cleaning.example.com): " DOMAIN
if [ -z "$DOMAIN" ]; then
    echo "Domain is required!"
    exit 1
fi

# Get email for Let's Encrypt
read -p "Enter your email for Let's Encrypt notifications: " EMAIL
if [ -z "$EMAIL" ]; then
    echo "Email is required!"
    exit 1
fi

# Get timezone (optional)
read -p "Enter your timezone (default: Australia/Brisbane): " TZ
TZ=${TZ:-Australia/Brisbane}

echo ""
echo "Configuration:"
echo "  Domain: $DOMAIN"
echo "  Email: $EMAIL"
echo "  Timezone: $TZ"
echo ""
read -p "Continue? (y/n) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    exit 1
fi

echo ""
echo "Step 1: Updating system..."
apt update && apt upgrade -y

echo ""
echo "Step 2: Installing Docker..."
if ! command -v docker &> /dev/null; then
    curl -fsSL https://get.docker.com -o get-docker.sh
    sh get-docker.sh
    rm get-docker.sh
else
    echo "Docker already installed"
fi

echo ""
echo "Step 3: Installing Docker Compose..."
if ! command -v docker-compose &> /dev/null; then
    apt install docker-compose-plugin -y
else
    echo "Docker Compose already installed"
fi

echo ""
echo "Step 4: Configuring firewall..."
ufw --force enable
ufw allow OpenSSH
ufw allow 80/tcp
ufw allow 443/tcp
ufw reload

echo ""
echo "Step 5: Creating deployment directory..."
mkdir -p /opt/cleaning-scheduler
cd /opt/cleaning-scheduler

echo ""
echo "Step 6: Creating docker-compose.yml..."
cat > docker-compose.yml <<EOF
services:
  cleaning-scheduler:
    image: druarnfield/cleaning-scheduler:latest
    container_name: cleaning-scheduler
    restart: unless-stopped
    expose:
      - "8080"
    volumes:
      - cleaning_data:/data
    environment:
      - PORT=8080
      - DB_PATH=/data/cleaning.db
      - TZ=$TZ
    networks:
      - app-network
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:8080/login"]
      interval: 30s
      timeout: 3s
      start_period: 5s
      retries: 3

  caddy:
    image: caddy:2-alpine
    container_name: caddy
    restart: unless-stopped
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./Caddyfile:/etc/caddy/Caddyfile
      - caddy_data:/data
      - caddy_config:/config
    networks:
      - app-network
    depends_on:
      - cleaning-scheduler

volumes:
  cleaning_data:
    driver: local
  caddy_data:
    driver: local
  caddy_config:
    driver: local

networks:
  app-network:
    driver: bridge
EOF

echo ""
echo "Step 7: Creating Caddyfile..."
cat > Caddyfile <<EOF
$DOMAIN {
    reverse_proxy cleaning-scheduler:8080

    header {
        Strict-Transport-Security "max-age=31536000; includeSubDomains; preload"
        X-Frame-Options "SAMEORIGIN"
        X-Content-Type-Options "nosniff"
        X-XSS-Protection "1; mode=block"
        Referrer-Policy "strict-origin-when-cross-origin"
    }

    log {
        output file /var/log/caddy/access.log
        format json
    }
}
EOF

echo ""
echo "Step 8: Pulling Docker images..."
docker pull druarnfield/cleaning-scheduler:latest
docker pull caddy:2-alpine

echo ""
echo "Step 9: Starting services..."
docker compose up -d

echo ""
echo "Step 10: Setting up automatic backups..."
cat > /opt/cleaning-scheduler/backup.sh <<'EOF'
#!/bin/bash
DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR=/opt/cleaning-scheduler/backups
mkdir -p $BACKUP_DIR

docker run --rm \
  -v cleaning-scheduler_cleaning_data:/data \
  -v $BACKUP_DIR:/backup \
  alpine cp /data/cleaning.db /backup/cleaning-$DATE.db

# Keep only last 30 days
find $BACKUP_DIR -name "cleaning-*.db" -mtime +30 -delete

echo "Backup completed: cleaning-$DATE.db"
EOF

chmod +x /opt/cleaning-scheduler/backup.sh

# Add to crontab
(crontab -l 2>/dev/null; echo "0 2 * * * /opt/cleaning-scheduler/backup.sh >> /var/log/cleaning-backup.log 2>&1") | crontab -

echo ""
echo "Step 11: Setting up automatic security updates..."
apt install unattended-upgrades -y
dpkg-reconfigure -plow unattended-upgrades

echo ""
echo "=================================="
echo "✅ Setup Complete!"
echo "=================================="
echo ""
echo "Your application should now be running at:"
echo "  https://$DOMAIN"
echo ""
echo "Important next steps:"
echo "  1. Wait 2-3 minutes for SSL certificates to be obtained"
echo "  2. Visit https://$DOMAIN and login as 'dru' or 'josie'"
echo "  3. Set your password on first login"
echo ""
echo "Useful commands:"
echo "  View logs:        cd /opt/cleaning-scheduler && docker compose logs -f"
echo "  Restart:          cd /opt/cleaning-scheduler && docker compose restart"
echo "  Stop:             cd /opt/cleaning-scheduler && docker compose down"
echo "  Start:            cd /opt/cleaning-scheduler && docker compose up -d"
echo "  Update:           cd /opt/cleaning-scheduler && docker pull druarnfield/cleaning-scheduler:latest && docker compose up -d"
echo "  Backup database:  /opt/cleaning-scheduler/backup.sh"
echo ""
echo "Backups are automatically created daily at 2 AM in /opt/cleaning-scheduler/backups/"
echo ""
echo "DNS Check:"
echo "  Make sure $DOMAIN points to this server's IP address"
echo "  Current IP: $(curl -s ifconfig.me)"
echo ""
