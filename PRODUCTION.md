# Production Deployment Guide (Digital Ocean)

Complete guide for deploying the Cleaning Scheduler on a Digital Ocean droplet with SSL/HTTPS.

---

## Prerequisites

- Digital Ocean droplet (Ubuntu 22.04 recommended)
- Domain name pointed to your droplet's IP address
- SSH access to your droplet

---

## 1. Initial Server Setup

### SSH into your droplet
```bash
ssh root@your-droplet-ip
```

### Update system
```bash
apt update && apt upgrade -y
```

### Install Docker
```bash
curl -fsSL https://get.docker.com -o get-docker.sh
sh get-docker.sh
```

### Install Docker Compose
```bash
apt install docker-compose-plugin -y
```

### Create a non-root user (optional but recommended)
```bash
adduser deployer
usermod -aG docker deployer
su - deployer
```

---

## 2. DNS Configuration

Before proceeding, ensure your domain is pointed to your droplet:

1. Log into your domain registrar
2. Add an **A record**:
   - **Name**: `@` (or your subdomain like `cleaning`)
   - **Value**: Your droplet's IP address
   - **TTL**: 300 (or default)

3. Optional: Add **www** subdomain:
   - **Name**: `www`
   - **Value**: Your droplet's IP address

Wait 5-10 minutes for DNS to propagate. Test with:
```bash
dig your-domain.com
```

---

## 3. Deploy Application

### Create deployment directory
```bash
mkdir -p ~/cleaning-scheduler
cd ~/cleaning-scheduler
```

---

## Option A: Caddy (Recommended - Easiest Setup)

### 1. Download configuration files
```bash
# Download docker-compose file
curl -O https://raw.githubusercontent.com/druarnfield/cleaning-scheduler/main/docker-compose.prod.yml

# Download Caddyfile
curl -O https://raw.githubusercontent.com/druarnfield/cleaning-scheduler/main/Caddyfile
```

### 2. Edit Caddyfile
```bash
nano Caddyfile
```

Replace `your-domain.com` with your actual domain (e.g., `cleaning.example.com`)

### 3. Start the application
```bash
docker compose -f docker-compose.prod.yml up -d
```

**That's it!** Caddy will automatically:
- Obtain SSL certificates from Let's Encrypt
- Renew certificates automatically
- Redirect HTTP to HTTPS

### 4. Verify
Visit `https://your-domain.com` - you should see a valid SSL certificate!

### View logs
```bash
docker compose -f docker-compose.prod.yml logs -f
```

---

## Option B: Nginx + Certbot (Traditional Setup)

### 1. Download configuration files
```bash
curl -O https://raw.githubusercontent.com/druarnfield/cleaning-scheduler/main/docker-compose.nginx.yml
curl -O https://raw.githubusercontent.com/druarnfield/cleaning-scheduler/main/nginx.conf
```

### 2. Edit nginx.conf
```bash
nano nginx.conf
```

Replace all instances of:
- `your-domain.com` with your actual domain
- `your-email@example.com` with your email

### 3. Initial nginx start (without SSL)
First, comment out the HTTPS server block in `nginx.conf` to start nginx without SSL certificates.

```bash
# Start nginx first
docker compose -f docker-compose.nginx.yml up -d nginx
```

### 4. Obtain SSL certificates
```bash
# Get certificates
docker compose -f docker-compose.nginx.yml run --rm certbot certonly \
  --webroot \
  --webroot-path=/var/www/certbot \
  --email your-email@example.com \
  --agree-tos \
  --no-eff-email \
  -d your-domain.com \
  -d www.your-domain.com
```

### 5. Uncomment SSL configuration
Edit `nginx.conf` and uncomment the HTTPS server blocks.

### 6. Restart nginx
```bash
docker compose -f docker-compose.nginx.yml restart nginx
```

### 7. Set up auto-renewal
Create a renewal cron job:
```bash
crontab -e
```

Add this line to renew certificates daily:
```
0 0 * * * cd ~/cleaning-scheduler && docker compose -f docker-compose.nginx.yml run --rm certbot renew && docker compose -f docker-compose.nginx.yml restart nginx
```

---

## 4. Firewall Configuration

### Configure UFW (Ubuntu Firewall)
```bash
# Allow SSH
sudo ufw allow OpenSSH

# Allow HTTP and HTTPS
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp

# Enable firewall
sudo ufw enable

# Check status
sudo ufw status
```

---

## 5. First-Time Application Setup

1. Visit `https://your-domain.com`
2. Login as **dru** or **josie**
3. Set your password on first login
4. Import tasks or create them manually

---

## 6. Monitoring & Maintenance

### View application logs
```bash
# Caddy
docker compose -f docker-compose.prod.yml logs -f cleaning-scheduler

# Nginx
docker compose -f docker-compose.nginx.yml logs -f cleaning-scheduler
```

### View reverse proxy logs
```bash
# Caddy
docker compose -f docker-compose.prod.yml logs -f caddy

# Nginx
docker compose -f docker-compose.nginx.yml logs -f nginx
```

### Restart application
```bash
# Caddy
docker compose -f docker-compose.prod.yml restart

# Nginx
docker compose -f docker-compose.nginx.yml restart
```

### Update to latest version
```bash
# Pull latest image
docker pull druarnfield/cleaning-scheduler:latest

# Restart with new image
# Caddy:
docker compose -f docker-compose.prod.yml down
docker compose -f docker-compose.prod.yml up -d

# Nginx:
docker compose -f docker-compose.nginx.yml down
docker compose -f docker-compose.nginx.yml up -d
```

### Backup database
```bash
# Create backup
docker run --rm \
  -v cleaning-scheduler_cleaning_data:/data \
  -v $(pwd):/backup \
  alpine cp /data/cleaning.db /backup/cleaning.db.backup-$(date +%Y%m%d)

# Download to local machine
scp user@your-droplet-ip:~/cleaning-scheduler/cleaning.db.backup-* ./
```

### Restore database
```bash
# Upload backup to droplet
scp cleaning.db.backup-YYYYMMDD user@your-droplet-ip:~/cleaning-scheduler/

# Restore
docker run --rm \
  -v cleaning-scheduler_cleaning_data:/data \
  -v $(pwd):/backup \
  alpine cp /backup/cleaning.db.backup-YYYYMMDD /data/cleaning.db

# Restart application
docker compose -f docker-compose.prod.yml restart
```

---

## 7. Security Best Practices

### Enable automatic security updates
```bash
sudo apt install unattended-upgrades
sudo dpkg-reconfigure -plow unattended-upgrades
```

### Disable root SSH login
```bash
sudo nano /etc/ssh/sshd_config
```

Change:
```
PermitRootLogin no
PasswordAuthentication no  # Use SSH keys only
```

Restart SSH:
```bash
sudo systemctl restart sshd
```

### Set up automated backups
Create a backup script:
```bash
nano ~/backup.sh
```

```bash
#!/bin/bash
DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR=~/backups
mkdir -p $BACKUP_DIR

# Backup database
docker run --rm \
  -v cleaning-scheduler_cleaning_data:/data \
  -v $BACKUP_DIR:/backup \
  alpine cp /data/cleaning.db /backup/cleaning-$DATE.db

# Keep only last 7 days
find $BACKUP_DIR -name "cleaning-*.db" -mtime +7 -delete
```

Make executable and add to cron:
```bash
chmod +x ~/backup.sh
crontab -e
```

Add:
```
0 2 * * * ~/backup.sh
```

---

## 8. Troubleshooting

### Check container status
```bash
docker ps -a
```

### Check container health
```bash
docker inspect cleaning-scheduler | grep -A 10 Health
```

### View all logs
```bash
docker compose -f docker-compose.prod.yml logs --tail=100
```

### Test SSL certificate
```bash
curl -I https://your-domain.com
```

### Check certificate expiry (Caddy)
Caddy handles this automatically, but you can check:
```bash
echo | openssl s_client -connect your-domain.com:443 2>/dev/null | openssl x509 -noout -dates
```

### Common issues

**SSL not working:**
- Verify DNS is pointing to your droplet: `dig your-domain.com`
- Check firewall: `sudo ufw status`
- View reverse proxy logs: `docker logs caddy` or `docker logs nginx`

**Application not starting:**
- Check logs: `docker logs cleaning-scheduler`
- Verify port 8080 is not in use: `netstat -tulpn | grep 8080`

**Can't access application:**
- Verify containers are running: `docker ps`
- Check firewall rules: `sudo ufw status`
- Test from server: `curl http://localhost:8080/login`

---

## 9. Performance Optimization

### Enable Docker log rotation
Create `/etc/docker/daemon.json`:
```json
{
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "10m",
    "max-file": "3"
  }
}
```

Restart Docker:
```bash
sudo systemctl restart docker
```

### Monitor resource usage
```bash
docker stats
```

---

## Quick Reference

### Caddy Deployment
```bash
cd ~/cleaning-scheduler
docker compose -f docker-compose.prod.yml up -d
docker compose -f docker-compose.prod.yml logs -f
```

### Nginx Deployment
```bash
cd ~/cleaning-scheduler
docker compose -f docker-compose.nginx.yml up -d
docker compose -f docker-compose.nginx.yml logs -f
```

### Update Application
```bash
docker pull druarnfield/cleaning-scheduler:latest
docker compose -f docker-compose.prod.yml down && docker compose -f docker-compose.prod.yml up -d
```

### Backup Database
```bash
docker run --rm -v cleaning-scheduler_cleaning_data:/data -v $(pwd):/backup alpine cp /data/cleaning.db /backup/backup-$(date +%Y%m%d).db
```

---

**Need help?** Check logs first: `docker compose logs -f`
