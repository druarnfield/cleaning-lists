# Deployment Guide

## Docker Deployment

This application is available as a pre-built Docker image on Docker Hub for easy deployment.

### Quick Start (Using Pre-built Image)

```bash
# Pull and run with docker-compose
docker-compose up -d
```

The application will be available at `http://localhost:8080`

### Building from Source (Development)

```bash
# Build locally
make docker-build

# Or manually
docker build -t cleaning-scheduler:latest .
```

### Docker Commands

```bash
# Pull the latest image
docker pull druarnfield/cleaning-scheduler:latest

# Start the application
make docker-run
# or
docker-compose up -d

# Build locally (for development)
make docker-build
# or
docker build -t cleaning-scheduler:latest .

# View logs
make docker-logs
# or
docker-compose logs -f

# Stop the application
make docker-stop
# or
docker-compose down

# Clean up (removes containers, volumes, and image)
make docker-clean
```

### Environment Variables

Configure the application using environment variables in `docker-compose.yml`:

- `PORT` - HTTP port (default: 8080)
- `DB_PATH` - Database file path (default: /data/cleaning.db)
- `TZ` - Timezone (default: Australia/Brisbane)

### Data Persistence

The database is persisted in a Docker volume named `cleaning_data`. This ensures your data survives container restarts and updates.

To backup the database:
```bash
docker run --rm -v cleaning_data:/data -v $(pwd):/backup alpine cp /data/cleaning.db /backup/cleaning.db.backup
```

To restore from backup:
```bash
docker run --rm -v cleaning_data:/data -v $(pwd):/backup alpine cp /backup/cleaning.db.backup /data/cleaning.db
```

### Production Deployment

For production deployment:

1. Update the timezone in `docker-compose.yml` to match your location
2. Consider using a reverse proxy (nginx, Traefik) for HTTPS
3. Set up automated backups of the database volume
4. Monitor container health using the built-in health check

Example nginx reverse proxy configuration:
```nginx
server {
    listen 80;
    server_name cleaning.example.com;

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### Image Size

The multistage build produces a minimal image:
- Builder stage: ~500MB (includes full Go toolchain)
- Final runtime image: ~20-30MB (Alpine + binary + minimal dependencies)

### First-Time Setup

On first run, the application will:
1. Create the database at `/data/cleaning.db`
2. Run migrations automatically
3. Create default users (dru and josie)
4. Generate initial task instances

You'll need to:
1. Navigate to `http://localhost:8080/login`
2. Login as either "dru" or "josie"
3. Set up your password on first login
4. Import your cleaning tasks CSV (if you have one)

### Updating

To update the application:

**Using pre-built image:**
```bash
# Pull latest image
docker pull druarnfield/cleaning-scheduler:latest

# Restart with new image
docker-compose down
docker-compose up -d
```

**Building from source:**
```bash
# Pull latest code
git pull

# Rebuild and restart
make docker-stop
make docker-build
make docker-run
```

The database will persist across updates thanks to the volume mount.

---

## Public Docker Image

This application is published on Docker Hub:

**Image**: `druarnfield/cleaning-scheduler:latest`

**Quick deploy on any machine:**
```bash
# Create docker-compose.yml or use this one-liner
docker run -d \
  --name cleaning-scheduler \
  -p 8080:8080 \
  -v cleaning_data:/data \
  -e TZ=Australia/Brisbane \
  --restart unless-stopped \
  druarnfield/cleaning-scheduler:latest
```

Visit Docker Hub: https://hub.docker.com/r/druarnfield/cleaning-scheduler
