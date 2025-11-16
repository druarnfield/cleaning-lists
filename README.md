# Cleaning Scheduler

A household cleaning task scheduler web application that automatically distributes recurring tasks fairly between two people based on time commitment.

![Docker Image Size](https://img.shields.io/docker/image-size/druarnfield/cleaning-scheduler/latest)
![Docker Pulls](https://img.shields.io/docker/pulls/druarnfield/cleaning-scheduler)

## Features

- 📅 **Automatic Task Scheduling** - Generate recurring task instances for the next 4 weeks
- ⚖️ **Fair Distribution** - Automatically balance tasks between two people based on estimated time
- ✅ **Task Completion Tracking** - Mark tasks as complete with one click
- 📊 **Dashboard Analytics** - View completion rates and statistics
- 🔄 **Bring Forward Tasks** - Reschedule future tasks to earlier dates
- 📱 **Mobile Optimized** - Responsive design with touch-friendly controls
- 📂 **CSV Import** - Bulk import tasks from CSV files

## Quick Start with Docker

### Using Docker Compose (Recommended)

1. Create a `docker-compose.yml` file:

```yaml
services:
  cleaning-scheduler:
    image: druarnfield/cleaning-scheduler:latest
    container_name: cleaning-scheduler
    restart: unless-stopped
    ports:
      - "8080:8080"
    volumes:
      - cleaning_data:/data
    environment:
      - PORT=8080
      - DB_PATH=/data/cleaning.db
      - TZ=Australia/Brisbane  # Adjust to your timezone

volumes:
  cleaning_data:
```

2. Start the application:

```bash
docker-compose up -d
```

3. Visit `http://localhost:8080`

### Using Docker Run

```bash
docker run -d \
  --name cleaning-scheduler \
  -p 8080:8080 \
  -v cleaning_data:/data \
  -e TZ=Australia/Brisbane \
  --restart unless-stopped \
  druarnfield/cleaning-scheduler:latest
```

## First-Time Setup

1. Navigate to `http://localhost:8080/login`
2. Login as either **dru** or **josie** (default users)
3. Set your password on first login
4. Start adding tasks or import from CSV

## Technology Stack

- **Backend**: Go 1.24 with Chi router
- **Database**: SQLite 3 with sqlc for type-safe queries
- **Frontend**: HTMX + Tailwind CSS + DaisyUI (server-side rendering)
- **Templates**: templ for type-safe HTML templates
- **Scheduler**: gocron for background jobs

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP port to listen on |
| `DB_PATH` | `/data/cleaning.db` | Path to SQLite database file |
| `TZ` | System timezone | Timezone for scheduling (e.g., `America/New_York`) |

## Data Persistence

The application stores all data in an SQLite database. When using Docker, mount a volume to `/data` to persist your data:

```bash
-v cleaning_data:/data
```

### Backup Your Data

```bash
# Backup
docker run --rm -v cleaning_data:/data -v $(pwd):/backup alpine \
  cp /data/cleaning.db /backup/cleaning.db.backup

# Restore
docker run --rm -v cleaning_data:/data -v $(pwd):/backup alpine \
  cp /backup/cleaning.db.backup /data/cleaning.db
```

## CSV Import Format

Tasks can be imported via CSV with the following columns:

```csv
Task,Category,Frequency,Assigned To,Estimated Mins
Vacuum Living Room,Cleaning,Weekly,dru,15
Mop Floors,Cleaning,Fortnightly,josie,30
Clean Bathroom,Cleaning,Weekly,alternate,20
```

**Frequency options**: Daily, Weekly, Fortnightly, Monthly, "N Weekly" (e.g., "2 Weekly"), "Nx/week" (e.g., "2x/week")

**Assigned To options**: dru, josie, both, alternate (or leave empty for auto-distribution)

## Building from Source

### Prerequisites

- Go 1.24+
- Node.js (for TailwindCSS)
- Make

### Build Steps

```bash
# Clone repository
git clone https://github.com/druarnfield/cleaning-scheduler.git
cd cleaning-scheduler

# Install dependencies
go mod download

# Generate code
make generate

# Build
make build

# Run
make run
```

### Build Docker Image

```bash
make docker-build
```

## Development

```bash
# Install development tools
make install

# Run with hot reload
make dev

# Run tests
make test
```

## Project Structure

```
.
├── cmd/server/          # Application entry point
├── internal/
│   ├── auth/           # Authentication logic
│   ├── database/       # SQLite schema and queries
│   ├── handlers/       # HTTP handlers
│   ├── scheduler/      # Task scheduling and distribution logic
│   └── templates/      # templ templates
└── web/
    └── static/         # Static assets
```

## License

MIT License - see LICENSE file for details

## Notes

This is a personal household application designed for two specific users. While functional and deployable, it's primarily a showcase project. The application has hardcoded user logic for "dru" and "josie".

## Support

For issues or questions, please open an issue on [GitHub](https://github.com/druarnfield/cleaning-scheduler/issues).

## Docker Hub

Official image: https://hub.docker.com/r/druarnfield/cleaning-scheduler
