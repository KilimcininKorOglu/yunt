# Yunt Backup and Restore Guide

This guide covers backup strategies, procedures, and disaster recovery for Yunt mail server.

## Table of Contents

1. [Backup Strategy](#backup-strategy)
2. [Database Backup](#database-backup)
3. [File System Backup](#file-system-backup)
4. [Automated Backups](#automated-backups)
5. [Restore Procedures](#restore-procedures)
6. [Disaster Recovery](#disaster-recovery)
7. [Backup Verification](#backup-verification)

## Backup Strategy

### What to Back Up

| Component       | Location                       | Priority | Frequency     |
|-----------------|--------------------------------|----------|---------------|
| Database        | SQLite file or external DB     | Critical | Daily         |
| Configuration   | `/etc/yunt/yunt.yaml`          | High     | On change     |
| TLS Certificates| `/etc/letsencrypt/`            | High     | On renewal    |
| Attachments     | `/var/lib/yunt/attachments/`   | Critical | Daily         |
| Docker Volumes  | Named volumes                  | Critical | Daily         |

### Backup Types

| Type        | Description                      | Use Case                    |
|-------------|----------------------------------|-----------------------------|
| Full        | Complete backup of all data      | Weekly baseline             |
| Incremental | Changes since last backup        | Daily efficiency            |
| Differential| Changes since last full backup   | Balance of speed/simplicity |
| Snapshot    | Point-in-time copy               | Pre-maintenance             |

### Retention Policy

Recommended retention schedule:

| Period    | Retention     | Purpose                    |
|-----------|---------------|----------------------------|
| Daily     | 7 days        | Quick recovery             |
| Weekly    | 4 weeks       | Short-term history         |
| Monthly   | 12 months     | Compliance/audit           |
| Yearly    | 7 years       | Legal requirements         |

## Database Backup

### SQLite Backup

**Simple file copy (requires service stop):**

```bash
#!/bin/bash
# sqlite-backup-offline.sh

BACKUP_DIR="/backup/yunt/sqlite"
DATA_DIR="/var/lib/yunt"
DATE=$(date +%Y%m%d_%H%M%S)

# Stop Yunt
docker stop yunt

# Create backup
mkdir -p "$BACKUP_DIR"
cp "$DATA_DIR/yunt.db" "$BACKUP_DIR/yunt_${DATE}.db"

# Start Yunt
docker start yunt

# Compress backup
gzip "$BACKUP_DIR/yunt_${DATE}.db"

echo "Backup completed: $BACKUP_DIR/yunt_${DATE}.db.gz"
```

**Online backup using SQLite backup API:**

```bash
#!/bin/bash
# sqlite-backup-online.sh

BACKUP_DIR="/backup/yunt/sqlite"
DATA_DIR="/var/lib/yunt"
DATE=$(date +%Y%m%d_%H%M%S)

mkdir -p "$BACKUP_DIR"

# Use SQLite's backup command (no downtime)
docker exec yunt sqlite3 "$DATA_DIR/yunt.db" ".backup '/tmp/yunt_backup.db'"
docker cp yunt:/tmp/yunt_backup.db "$BACKUP_DIR/yunt_${DATE}.db"
docker exec yunt rm /tmp/yunt_backup.db

# Compress
gzip "$BACKUP_DIR/yunt_${DATE}.db"

echo "Online backup completed: $BACKUP_DIR/yunt_${DATE}.db.gz"
```

### PostgreSQL Backup

**Using pg_dump:**

```bash
#!/bin/bash
# postgres-backup.sh

BACKUP_DIR="/backup/yunt/postgres"
DATE=$(date +%Y%m%d_%H%M%S)
DB_HOST="postgres"
DB_NAME="yunt"
DB_USER="yunt"
PGPASSWORD="${POSTGRES_PASSWORD}"

mkdir -p "$BACKUP_DIR"

# Create backup
export PGPASSWORD
pg_dump -h "$DB_HOST" -U "$DB_USER" -d "$DB_NAME" \
    --format=custom \
    --compress=9 \
    --file="$BACKUP_DIR/yunt_${DATE}.dump"

# Verify backup
pg_restore --list "$BACKUP_DIR/yunt_${DATE}.dump" > /dev/null 2>&1
if [ $? -eq 0 ]; then
    echo "Backup verified: $BACKUP_DIR/yunt_${DATE}.dump"
else
    echo "ERROR: Backup verification failed!"
    exit 1
fi
```

**Using Docker:**

```bash
#!/bin/bash
# postgres-backup-docker.sh

BACKUP_DIR="/backup/yunt/postgres"
DATE=$(date +%Y%m%d_%H%M%S)

mkdir -p "$BACKUP_DIR"

docker exec yunt-postgres pg_dump -U yunt -d yunt \
    --format=custom --compress=9 \
    > "$BACKUP_DIR/yunt_${DATE}.dump"

echo "Backup completed: $BACKUP_DIR/yunt_${DATE}.dump"
```

**Point-in-time recovery with WAL archiving:**

```ini
# postgresql.conf
wal_level = replica
archive_mode = on
archive_command = 'cp %p /backup/yunt/postgres/wal/%f'
```

```bash
# Restore to specific point
pg_restore --dbname=yunt_recovery \
    --target-time='2024-01-15 14:30:00' \
    /backup/yunt/postgres/yunt_latest.dump
```

### MySQL Backup

**Using mysqldump:**

```bash
#!/bin/bash
# mysql-backup.sh

BACKUP_DIR="/backup/yunt/mysql"
DATE=$(date +%Y%m%d_%H%M%S)
DB_HOST="mysql"
DB_NAME="yunt"
DB_USER="yunt"
DB_PASS="${MYSQL_PASSWORD}"

mkdir -p "$BACKUP_DIR"

# Create backup
mysqldump -h "$DB_HOST" -u "$DB_USER" -p"$DB_PASS" \
    --single-transaction \
    --routines \
    --triggers \
    --databases "$DB_NAME" \
    | gzip > "$BACKUP_DIR/yunt_${DATE}.sql.gz"

echo "Backup completed: $BACKUP_DIR/yunt_${DATE}.sql.gz"
```

**Using Docker:**

```bash
#!/bin/bash
# mysql-backup-docker.sh

BACKUP_DIR="/backup/yunt/mysql"
DATE=$(date +%Y%m%d_%H%M%S)

mkdir -p "$BACKUP_DIR"

docker exec yunt-mysql mysqldump -u yunt -p"${MYSQL_PASSWORD}" \
    --single-transaction \
    --routines \
    --triggers \
    yunt \
    | gzip > "$BACKUP_DIR/yunt_${DATE}.sql.gz"

echo "Backup completed: $BACKUP_DIR/yunt_${DATE}.sql.gz"
```

### MongoDB Backup

**Using mongodump:**

```bash
#!/bin/bash
# mongodb-backup.sh

BACKUP_DIR="/backup/yunt/mongodb"
DATE=$(date +%Y%m%d_%H%M%S)
MONGO_URI="mongodb://yunt:${MONGO_PASSWORD}@mongodb:27017/yunt"

mkdir -p "$BACKUP_DIR"

# Create backup
mongodump --uri="$MONGO_URI" \
    --gzip \
    --archive="$BACKUP_DIR/yunt_${DATE}.archive.gz"

echo "Backup completed: $BACKUP_DIR/yunt_${DATE}.archive.gz"
```

**Using Docker:**

```bash
#!/bin/bash
# mongodb-backup-docker.sh

BACKUP_DIR="/backup/yunt/mongodb"
DATE=$(date +%Y%m%d_%H%M%S)

mkdir -p "$BACKUP_DIR"

docker exec yunt-mongodb mongodump \
    --db=yunt \
    --gzip \
    --archive \
    > "$BACKUP_DIR/yunt_${DATE}.archive.gz"

echo "Backup completed: $BACKUP_DIR/yunt_${DATE}.archive.gz"
```

## File System Backup

### Docker Volume Backup

**Backup named volumes:**

```bash
#!/bin/bash
# volume-backup.sh

BACKUP_DIR="/backup/yunt/volumes"
DATE=$(date +%Y%m%d_%H%M%S)
VOLUME_NAME="yunt-data"

mkdir -p "$BACKUP_DIR"

# Create backup using temporary container
docker run --rm \
    -v "${VOLUME_NAME}:/source:ro" \
    -v "${BACKUP_DIR}:/backup" \
    alpine tar czf "/backup/${VOLUME_NAME}_${DATE}.tar.gz" -C /source .

echo "Volume backup completed: $BACKUP_DIR/${VOLUME_NAME}_${DATE}.tar.gz"
```

**Backup all Yunt volumes:**

```bash
#!/bin/bash
# backup-all-volumes.sh

BACKUP_DIR="/backup/yunt/volumes"
DATE=$(date +%Y%m%d_%H%M%S)

mkdir -p "$BACKUP_DIR"

for volume in yunt-data yunt-postgres-data; do
    if docker volume inspect "$volume" > /dev/null 2>&1; then
        docker run --rm \
            -v "${volume}:/source:ro" \
            -v "${BACKUP_DIR}:/backup" \
            alpine tar czf "/backup/${volume}_${DATE}.tar.gz" -C /source .
        echo "Backed up: $volume"
    fi
done

echo "All volume backups completed"
```

### Configuration Backup

```bash
#!/bin/bash
# config-backup.sh

BACKUP_DIR="/backup/yunt/config"
DATE=$(date +%Y%m%d_%H%M%S)

mkdir -p "$BACKUP_DIR"

# Backup configuration files
tar czf "$BACKUP_DIR/config_${DATE}.tar.gz" \
    /etc/yunt/ \
    /opt/yunt/docker-compose.yml \
    /opt/yunt/.env \
    2>/dev/null

echo "Configuration backup completed: $BACKUP_DIR/config_${DATE}.tar.gz"
```

### TLS Certificate Backup

```bash
#!/bin/bash
# cert-backup.sh

BACKUP_DIR="/backup/yunt/certs"
DATE=$(date +%Y%m%d_%H%M%S)

mkdir -p "$BACKUP_DIR"

# Backup Let's Encrypt certificates
tar czf "$BACKUP_DIR/certs_${DATE}.tar.gz" \
    /etc/letsencrypt/live/ \
    /etc/letsencrypt/archive/ \
    /etc/letsencrypt/renewal/

echo "Certificate backup completed: $BACKUP_DIR/certs_${DATE}.tar.gz"
```

## Automated Backups

### Complete Backup Script

Create `/opt/yunt/scripts/backup.sh`:

```bash
#!/bin/bash
# =============================================================================
# Yunt Complete Backup Script
# =============================================================================

set -euo pipefail

# Configuration
BACKUP_ROOT="/backup/yunt"
DATE=$(date +%Y%m%d_%H%M%S)
RETENTION_DAYS=7
DB_DRIVER="${YUNT_DATABASE_DRIVER:-sqlite}"
LOG_FILE="/var/log/yunt/backup.log"

# Ensure backup directory exists
mkdir -p "$BACKUP_ROOT"/{db,volumes,config}

# Logging function
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

# Error handler
error_exit() {
    log "ERROR: $1"
    exit 1
}

# Cleanup old backups
cleanup_old_backups() {
    log "Cleaning up backups older than $RETENTION_DAYS days..."
    find "$BACKUP_ROOT" -type f -mtime +"$RETENTION_DAYS" -delete
    log "Cleanup completed"
}

# Database backup
backup_database() {
    log "Starting database backup (driver: $DB_DRIVER)..."
    
    case "$DB_DRIVER" in
        sqlite)
            docker exec yunt sqlite3 /var/lib/yunt/yunt.db ".backup '/tmp/backup.db'"
            docker cp yunt:/tmp/backup.db "$BACKUP_ROOT/db/yunt_${DATE}.db"
            docker exec yunt rm /tmp/backup.db
            gzip "$BACKUP_ROOT/db/yunt_${DATE}.db"
            ;;
        postgres)
            docker exec yunt-postgres pg_dump -U yunt -d yunt \
                --format=custom --compress=9 \
                > "$BACKUP_ROOT/db/yunt_${DATE}.dump"
            ;;
        mysql)
            docker exec yunt-mysql mysqldump -u yunt -p"${MYSQL_PASSWORD}" \
                --single-transaction yunt \
                | gzip > "$BACKUP_ROOT/db/yunt_${DATE}.sql.gz"
            ;;
        mongodb)
            docker exec yunt-mongodb mongodump --db=yunt --gzip --archive \
                > "$BACKUP_ROOT/db/yunt_${DATE}.archive.gz"
            ;;
        *)
            error_exit "Unknown database driver: $DB_DRIVER"
            ;;
    esac
    
    log "Database backup completed"
}

# Volume backup
backup_volumes() {
    log "Starting volume backup..."
    
    for volume in yunt-data; do
        if docker volume inspect "$volume" > /dev/null 2>&1; then
            docker run --rm \
                -v "${volume}:/source:ro" \
                -v "$BACKUP_ROOT/volumes:/backup" \
                alpine tar czf "/backup/${volume}_${DATE}.tar.gz" -C /source .
            log "Backed up volume: $volume"
        fi
    done
    
    log "Volume backup completed"
}

# Configuration backup
backup_config() {
    log "Starting configuration backup..."
    
    tar czf "$BACKUP_ROOT/config/config_${DATE}.tar.gz" \
        /opt/yunt/docker-compose.yml \
        /opt/yunt/.env \
        /etc/yunt/ \
        2>/dev/null || true
    
    log "Configuration backup completed"
}

# Verify backup
verify_backup() {
    log "Verifying backups..."
    
    # Check database backup exists and has size > 0
    local db_backup=$(ls -t "$BACKUP_ROOT/db/"* 2>/dev/null | head -1)
    if [ -z "$db_backup" ] || [ ! -s "$db_backup" ]; then
        error_exit "Database backup verification failed"
    fi
    
    log "Backup verification passed"
}

# Main execution
main() {
    log "=========================================="
    log "Starting Yunt backup..."
    log "=========================================="
    
    backup_database
    backup_volumes
    backup_config
    verify_backup
    cleanup_old_backups
    
    log "=========================================="
    log "Backup completed successfully"
    log "=========================================="
}

main "$@"
```

Make executable:

```bash
chmod +x /opt/yunt/scripts/backup.sh
```

### Cron Configuration

```bash
# Edit crontab
crontab -e

# Add daily backup at 2 AM
0 2 * * * /opt/yunt/scripts/backup.sh >> /var/log/yunt/backup.log 2>&1

# Weekly full backup on Sunday
0 3 * * 0 /opt/yunt/scripts/backup.sh --full >> /var/log/yunt/backup.log 2>&1
```

### Systemd Timer (Alternative)

Create `/etc/systemd/system/yunt-backup.service`:

```ini
[Unit]
Description=Yunt Backup Service
After=docker.service

[Service]
Type=oneshot
ExecStart=/opt/yunt/scripts/backup.sh
User=root

[Install]
WantedBy=multi-user.target
```

Create `/etc/systemd/system/yunt-backup.timer`:

```ini
[Unit]
Description=Daily Yunt Backup

[Timer]
OnCalendar=*-*-* 02:00:00
Persistent=true

[Install]
WantedBy=timers.target
```

Enable the timer:

```bash
sudo systemctl daemon-reload
sudo systemctl enable yunt-backup.timer
sudo systemctl start yunt-backup.timer
```

### Remote Backup

**Using rsync:**

```bash
#!/bin/bash
# remote-backup.sh

BACKUP_DIR="/backup/yunt"
REMOTE_HOST="backup-server.example.com"
REMOTE_PATH="/backups/yunt"

# Sync to remote server
rsync -avz --delete \
    -e "ssh -i /root/.ssh/backup_key" \
    "$BACKUP_DIR/" \
    "backup@${REMOTE_HOST}:${REMOTE_PATH}/"
```

**Using rclone (cloud storage):**

```bash
#!/bin/bash
# cloud-backup.sh

BACKUP_DIR="/backup/yunt"
REMOTE="s3:yunt-backups"
DATE=$(date +%Y%m%d)

# Sync to S3
rclone sync "$BACKUP_DIR" "$REMOTE/$DATE" \
    --transfers=4 \
    --checkers=8 \
    --log-file=/var/log/yunt/rclone.log
```

## Restore Procedures

### SQLite Restore

```bash
#!/bin/bash
# sqlite-restore.sh

BACKUP_FILE="$1"
DATA_DIR="/var/lib/yunt"

if [ -z "$BACKUP_FILE" ]; then
    echo "Usage: $0 <backup_file.db.gz>"
    exit 1
fi

# Stop Yunt
docker stop yunt

# Backup current database
mv "$DATA_DIR/yunt.db" "$DATA_DIR/yunt.db.pre-restore"

# Restore from backup
if [[ "$BACKUP_FILE" == *.gz ]]; then
    gunzip -c "$BACKUP_FILE" > "$DATA_DIR/yunt.db"
else
    cp "$BACKUP_FILE" "$DATA_DIR/yunt.db"
fi

# Set permissions
chown yunt:yunt "$DATA_DIR/yunt.db"

# Start Yunt
docker start yunt

echo "Restore completed from: $BACKUP_FILE"
```

### PostgreSQL Restore

```bash
#!/bin/bash
# postgres-restore.sh

BACKUP_FILE="$1"
DB_HOST="postgres"
DB_NAME="yunt"
DB_USER="yunt"

if [ -z "$BACKUP_FILE" ]; then
    echo "Usage: $0 <backup_file.dump>"
    exit 1
fi

# Stop Yunt
docker stop yunt

# Drop and recreate database
docker exec yunt-postgres psql -U "$DB_USER" -c "DROP DATABASE IF EXISTS ${DB_NAME};"
docker exec yunt-postgres psql -U "$DB_USER" -c "CREATE DATABASE ${DB_NAME};"

# Restore
pg_restore -h "$DB_HOST" -U "$DB_USER" -d "$DB_NAME" "$BACKUP_FILE"

# Start Yunt
docker start yunt

echo "Restore completed from: $BACKUP_FILE"
```

### MySQL Restore

```bash
#!/bin/bash
# mysql-restore.sh

BACKUP_FILE="$1"

if [ -z "$BACKUP_FILE" ]; then
    echo "Usage: $0 <backup_file.sql.gz>"
    exit 1
fi

# Stop Yunt
docker stop yunt

# Restore
if [[ "$BACKUP_FILE" == *.gz ]]; then
    gunzip -c "$BACKUP_FILE" | docker exec -i yunt-mysql mysql -u yunt -p"${MYSQL_PASSWORD}" yunt
else
    docker exec -i yunt-mysql mysql -u yunt -p"${MYSQL_PASSWORD}" yunt < "$BACKUP_FILE"
fi

# Start Yunt
docker start yunt

echo "Restore completed from: $BACKUP_FILE"
```

### Volume Restore

```bash
#!/bin/bash
# volume-restore.sh

BACKUP_FILE="$1"
VOLUME_NAME="yunt-data"

if [ -z "$BACKUP_FILE" ]; then
    echo "Usage: $0 <volume_backup.tar.gz>"
    exit 1
fi

# Stop containers
docker stop yunt

# Remove existing volume
docker volume rm "$VOLUME_NAME" 2>/dev/null || true

# Create new volume
docker volume create "$VOLUME_NAME"

# Restore backup
docker run --rm \
    -v "${VOLUME_NAME}:/target" \
    -v "$(dirname $BACKUP_FILE):/backup:ro" \
    alpine tar xzf "/backup/$(basename $BACKUP_FILE)" -C /target

# Start containers
docker start yunt

echo "Volume restore completed from: $BACKUP_FILE"
```

## Disaster Recovery

### Recovery Plan

1. **Assess the Situation**
   - Identify what failed (server, database, storage)
   - Determine the recovery point objective (RPO)
   - Estimate recovery time

2. **Prepare Recovery Environment**
   - Provision new server if needed
   - Install Docker and dependencies
   - Restore configuration files

3. **Restore Data**
   - Restore database from latest backup
   - Restore volumes if using filesystem storage
   - Verify data integrity

4. **Validate Recovery**
   - Test authentication
   - Verify email access
   - Check all services are operational

### Full System Recovery Script

```bash
#!/bin/bash
# disaster-recovery.sh

set -euo pipefail

BACKUP_DIR="/backup/yunt"
RESTORE_DATE="${1:-latest}"

echo "Starting disaster recovery..."

# Find latest backups or use specified date
if [ "$RESTORE_DATE" == "latest" ]; then
    DB_BACKUP=$(ls -t "$BACKUP_DIR/db/"* | head -1)
    VOL_BACKUP=$(ls -t "$BACKUP_DIR/volumes/yunt-data_"* | head -1)
    CONFIG_BACKUP=$(ls -t "$BACKUP_DIR/config/"* | head -1)
else
    DB_BACKUP="$BACKUP_DIR/db/yunt_${RESTORE_DATE}.dump"
    VOL_BACKUP="$BACKUP_DIR/volumes/yunt-data_${RESTORE_DATE}.tar.gz"
    CONFIG_BACKUP="$BACKUP_DIR/config/config_${RESTORE_DATE}.tar.gz"
fi

echo "Using backups:"
echo "  Database: $DB_BACKUP"
echo "  Volume: $VOL_BACKUP"
echo "  Config: $CONFIG_BACKUP"

# Stop existing services
docker-compose down 2>/dev/null || true

# Restore configuration
echo "Restoring configuration..."
tar xzf "$CONFIG_BACKUP" -C /

# Restore volumes
echo "Restoring volumes..."
docker volume rm yunt-data 2>/dev/null || true
docker volume create yunt-data
docker run --rm \
    -v yunt-data:/target \
    -v "$BACKUP_DIR/volumes:/backup:ro" \
    alpine tar xzf "/backup/$(basename $VOL_BACKUP)" -C /target

# Start services
echo "Starting services..."
cd /opt/yunt
docker-compose up -d

# Wait for services
echo "Waiting for services to start..."
sleep 30

# Restore database
echo "Restoring database..."
# (Database-specific restore commands here)

# Verify recovery
echo "Verifying recovery..."
curl -sf http://localhost:8025/health || { echo "Health check failed!"; exit 1; }

echo "Disaster recovery completed successfully!"
```

## Backup Verification

### Automated Verification

```bash
#!/bin/bash
# verify-backup.sh

BACKUP_DIR="/backup/yunt"
LOG_FILE="/var/log/yunt/backup-verify.log"

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

# Verify SQLite backup
verify_sqlite() {
    local backup="$1"
    local temp_db="/tmp/verify_yunt.db"
    
    gunzip -c "$backup" > "$temp_db"
    sqlite3 "$temp_db" "PRAGMA integrity_check;" | grep -q "ok"
    local result=$?
    rm -f "$temp_db"
    
    return $result
}

# Verify PostgreSQL backup
verify_postgres() {
    local backup="$1"
    pg_restore --list "$backup" > /dev/null 2>&1
    return $?
}

# Main verification
log "Starting backup verification..."

latest_backup=$(ls -t "$BACKUP_DIR/db/"* | head -1)

if [[ "$latest_backup" == *.db.gz ]]; then
    if verify_sqlite "$latest_backup"; then
        log "SQLite backup verified: $latest_backup"
    else
        log "ERROR: SQLite backup verification failed: $latest_backup"
        exit 1
    fi
elif [[ "$latest_backup" == *.dump ]]; then
    if verify_postgres "$latest_backup"; then
        log "PostgreSQL backup verified: $latest_backup"
    else
        log "ERROR: PostgreSQL backup verification failed: $latest_backup"
        exit 1
    fi
fi

log "Backup verification completed successfully"
```

### Restore Testing

Schedule periodic restore tests:

```bash
#!/bin/bash
# test-restore.sh

# Create isolated test environment
docker network create yunt-test
docker volume create yunt-test-data

# Restore to test environment
# (Restore commands here)

# Verify functionality
curl -sf http://yunt-test:8025/health

# Cleanup
docker stop yunt-test
docker rm yunt-test
docker volume rm yunt-test-data
docker network rm yunt-test
```

## Next Steps

- [Deployment Guide](deployment.md) - Initial deployment setup
- [Production Guide](production.md) - Security and optimization
- [Reverse Proxy Setup](reverse-proxy.md) - TLS termination and routing
