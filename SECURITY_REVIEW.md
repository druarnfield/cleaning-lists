# Security Review for Public Container Registry

**Review Date**: 2025-11-16
**Image**: cleaning-scheduler:latest
**Status**: ⚠️ PRIVACY CONCERNS - Review before publishing

---

## 🔴 Privacy Issues

### 1. Hardcoded Personal Names
**Severity**: Medium
**Location**: Throughout codebase

The application is designed for specific users ("dru" and "josie") with names hardcoded in:
- Database schema (`internal/database/migrations/001_initial_schema.sql`)
- Application logic (handlers, templates, balancer)
- Go module path (`github.com/druarnfield/cleaning-scheduler`)

**Impact**:
- Exposes personal identity
- Application is not generic/reusable by others without modification

**Recommendation**:
If publishing publicly, consider:
1. Document this is a personal household app (not production software)
2. Create a generic fork with configurable usernames
3. Accept that it's a personal project showcase

### 2. Module Import Path
**Severity**: Low
**Location**: `go.mod`

```go
module github.com/druarnfield/cleaning-scheduler
```

Exposes GitHub username.

**Recommendation**:
- This is standard for Go modules, generally acceptable
- If privacy is critical, fork to a neutral organization account

---

## ✅ Security Strengths

### 1. No Hardcoded Secrets
- Session secrets generated at runtime (random 32 bytes)
- No API keys or tokens in code
- No default passwords

### 2. Proper File Exclusion
**Fixed**: CSV files now excluded from image via `.dockerignore`

Excluded from image:
- Development databases (*.db, *.db-shm, *.db-wal)
- Personal CSV data (*.csv)
- IDE configurations
- Build artifacts
- Git history

### 3. Non-Root User
Container runs as `appuser` (UID 1000), not root.

### 4. Health Checks
Built-in health monitoring for container orchestration.

### 5. No Data Baked In
- Database created at runtime in mounted volume
- No personal data in image layers
- Clean separation of code and data

### 6. Minimal Attack Surface
- Small image size (43.6MB)
- Only necessary runtime dependencies
- No unnecessary tools or packages

---

## 📋 Pre-Publication Checklist

Before pushing to a public registry:

- [x] CSV files excluded from image
- [x] No database files in image
- [x] No secrets or credentials in code
- [x] Non-root user configured
- [x] Health checks implemented
- [ ] **Decision**: Accept personal names are visible?
- [ ] Add LICENSE file
- [ ] Add comprehensive README
- [ ] Consider privacy disclaimer
- [ ] Tag with appropriate version

---

## 🔍 Vulnerability Scan Recommendations

Before publishing, run:

```bash
# Scan for vulnerabilities
docker scan cleaning-scheduler:latest

# Or use Trivy
trivy image cleaning-scheduler:latest

# Check for secrets (just in case)
docker run --rm -v /var/run/docker.sock:/var/run/docker.sock \
  trufflesecurity/trufflehog:latest \
  docker --image cleaning-scheduler:latest
```

---

## 📝 License Considerations

**Current Status**: No LICENSE file in repository

**Dependencies**:
All dependencies are open source with permissive licenses:
- Go (BSD)
- Chi router (MIT)
- SQLite (Public Domain)
- HTMX (BSD)
- DaisyUI (MIT)
- templ (MIT)

**Recommendation**:
Add an MIT or Apache 2.0 license before publishing.

---

## 🎯 Publishing Recommendations

### Option 1: Public as Personal Project
- Add disclaimer: "Personal household cleaning scheduler for two people"
- Accept that names are visible
- Add MIT license
- Tag as `latest` and version tags (e.g., `v1.0.0`)

### Option 2: Generalize First
- Refactor to use environment variables for usernames
- Make it a generic "2-person task scheduler"
- Remove hardcoded names from schema
- Then publish

### Option 3: Private Registry
- Use GitHub Container Registry with private visibility
- Or self-hosted private registry
- No privacy concerns

---

## 🚀 Recommended Publishing Process

If proceeding with public publication:

```bash
# 1. Rebuild to ensure latest .dockerignore is applied
docker build --no-cache -t cleaning-scheduler:latest .

# 2. Verify no personal data in image
docker run --rm cleaning-scheduler:latest ls -la

# 3. Tag for your registry
docker tag cleaning-scheduler:latest ghcr.io/yourusername/cleaning-scheduler:v1.0.0
docker tag cleaning-scheduler:latest ghcr.io/yourusername/cleaning-scheduler:latest

# 4. Push
docker push ghcr.io/yourusername/cleaning-scheduler:v1.0.0
docker push ghcr.io/yourusername/cleaning-scheduler:latest
```

---

## Summary

**Safe to publish?** ⚠️ **Yes, with caveats**

The image itself is **technically secure** with no secrets or personal data. However, the **source code** contains hardcoded personal names by design.

If you're comfortable with:
- Your first name being visible in the code
- The app being specific to two users
- It being clearly a personal project

Then it's **safe to publish** as a personal project showcase.

**No blockers found that would expose sensitive data or create security vulnerabilities.**
