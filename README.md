# PostgreSQL Cluster with Go API

This project demonstrates how to set up a PostgreSQL primary-replica cluster using Docker and interact with it using a Golang API server.

## Project Structure

```
postgres-cluster/
├── docker-compose.yml    # Docker Compose configuration
├── primary/              # Primary PostgreSQL configuration
│   ├── Dockerfile
│   ├── primary.conf      # PostgreSQL configuration
│   └── pg_hba.conf       # PostgreSQL authentication configuration
├── replica/              # Replica PostgreSQL configuration
│   ├── Dockerfile
│   ├── replica.conf
│   └── docker-entrypoint-initdb.d/
│       └── init-replica.sh
└── go-server/            # Go application
    ├── go.mod
    ├── go.sum
    └── main.go
```

## Step 1: Set Up PostgreSQL Cluster

### Create Configuration Files

First, create the necessary configuration files:

**primary/primary.conf**:
```
listen_addresses = '*'
wal_level = replica
max_wal_senders = 10
wal_keep_size = 64
hot_standby = on
```

**primary/pg_hba.conf**:
```
# "local" is for Unix domain socket connections only
local   all             all                                     trust
# IPv4 local connections:
host    all             all             127.0.0.1/32            trust
# IPv6 local connections:
host    all             all             ::1/128                 trust
# Allow replication connections
host    replication     postgres        all                     md5
# Allow all connections from Docker network
host    all             postgres        all                     md5
```

**primary/Dockerfile**:
```dockerfile
FROM postgres:15

COPY primary.conf /etc/postgresql/postgresql.conf
COPY pg_hba.conf /etc/postgresql/pg_hba.conf

CMD ["postgres", "-c", "config_file=/etc/postgresql/postgresql.conf", "-c", "hba_file=/etc/postgresql/pg_hba.conf"]
```

**replica/replica.conf**:
```
hot_standby = on
```

**replica/docker-entrypoint-initdb.d/init-replica.sh**:
```bash
#!/bin/bash
set -e

# Wait until primary is available
until pg_isready -h $PRIMARY_HOST -p 5432; do
  echo "Waiting for primary to be ready..."
  sleep 2
done

# Perform base backup
rm -rf /var/lib/postgresql/data/*
PGPASSWORD=$POSTGRES_PASSWORD pg_basebackup -h $PRIMARY_HOST -D /var/lib/postgresql/data -U $POSTGRES_USER -Fp -Xs -P -R
```

Make the init script executable:
```bash
chmod +x replica/docker-entrypoint-initdb.d/init-replica.sh
```

**replica/Dockerfile**:
```dockerfile
FROM postgres:15

COPY replica.conf /etc/postgresql/postgresql.conf
COPY docker-entrypoint-initdb.d /docker-entrypoint-initdb.d/
```

**docker-compose.yml**:
```yaml
version: '3.8'

services:
  primary:
    build: ./primary
    container_name: primary
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: password
      POSTGRES_DB: appdb
    ports:
      - "5432:5432"
    volumes:
      - primary_data:/var/lib/postgresql/data
    networks:
      - pg-network

  replica:
    build: ./replica
    container_name: replica
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: password
      POSTGRES_DB: appdb
      PRIMARY_HOST: primary
    ports:
      - "5433:5432"
    depends_on:
      - primary
    volumes:
      - replica_data:/var/lib/postgresql/data
    networks:
      - pg-network

volumes:
  primary_data:
  replica_data:

networks:
  pg-network:
```

### Launch PostgreSQL Cluster

```bash
# Build and start the cluster
docker compose build
docker compose up -d

# To see the logs
docker compose logs -f

# To check container status
docker ps

# To verify the replica is connected
docker exec -it primary psql -U postgres -d appdb -c "SELECT * FROM pg_stat_replication;"
```

## Step 2: Set Up Go Server

### Create Go Project

```bash
mkdir -p go-server
cd go-server

# Initialize Go module
go mod init postgres-cluster-api

# Install dependencies
go get github.com/lib/pq
go get github.com/gorilla/mux
```

### Create main.go

Create a file called `main.go` with the Go code to interact with the PostgreSQL cluster (see separate file).

### Run the Go Server

```bash
cd go-server
go run main.go
```

The server will start on port 8080 by default.

## Step 3: Test the API

Use curl to test the API endpoints:

```bash
# Health check
curl http://localhost:8080/health

# Create a user
curl -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"name":"John Doe","email":"john@example.com"}'

# Get all users
curl http://localhost:8080/users

# Get a specific user (replace 1 with actual ID)
curl http://localhost:8080/users/1
```

## Verify Replication

To verify that data is being properly replicated:

### Using psql

```bash
# Connect to primary
docker exec -it primary psql -U postgres -d appdb -c "SELECT * FROM users;"

# Connect to replica
docker exec -it replica psql -U postgres -d appdb -c "SELECT * FROM users;"
```

### Using DBeaver or another GUI tool

Connect to both databases:
- Primary: localhost:5432
- Replica: localhost:5433
- Username: postgres
- Password: password
- Database: appdb

## Troubleshooting

If you don't see tables in your database:

1. Make sure you're looking in the correct database (`appdb`)
2. Check if the Go application has run correctly and created the tables
3. Run this query to see all tables:
```sql
SELECT table_name FROM information_schema.tables WHERE table_schema = 'public';
```

If the replica is not connecting:

1. Check logs: `docker compose logs replica`
2. Make sure network connectivity is working: `docker exec -it replica ping primary`
3. Verify pg_hba.conf allows connections: `docker exec -it primary cat /etc/postgresql/pg_hba.conf`

## Cleaning Up

To stop and remove the containers:

```bash
docker compose down

# To also remove volumes (deletes all data)
docker compose down -v
```

## References

- [PostgreSQL Replication Documentation](https://www.postgresql.org/docs/current/runtime-config-replication.html)
- [Docker Compose Documentation](https://docs.docker.com/compose/)
- [Golang PostgreSQL Driver](https://github.com/lib/pq)