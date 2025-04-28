PostgreSQL Cluster with Go API
This project demonstrates how to set up a PostgreSQL primary-replica cluster using Docker and interact with it using a Golang API server.

Project Structure
pgsql
Copy
Edit
postgres-cluster/
├── docker-compose.yml    # Docker Compose configuration
├── primary/              # Primary PostgreSQL configuration
│   ├── Dockerfile
│   ├── primary.conf
│   └── pg_hba.conf
├── replica/              # Replica PostgreSQL configuration
│   ├── Dockerfile
│   ├── replica.conf
│   └── docker-entrypoint-initdb.d/
│       └── init-replica.sh
go-server/            # Go application
    ├── go.mod
    ├── go.sum
    └── main.go
    
Step 1: Set Up PostgreSQL Cluster
The necessary configuration files (primary.conf, pg_hba.conf, replica.conf, etc.) are already provided in the repository.

Launch PostgreSQL Cluster
bash
Copy
Edit
# Build and start the cluster
docker compose build
docker compose up -d

# View logs
docker compose logs -f

# Check container status
docker ps

# Verify replication status
docker exec -it primary psql -U postgres -d appdb -c "SELECT * FROM pg_stat_replication;"
Step 2: Set Up Go Server
Create Go Project
Navigate to the go-server directory and set up the Go environment:

bash
Copy
Edit
cd go-server

# Initialize Go module
go mod init postgres-cluster-api

# Install dependencies
go get github.com/lib/pq
go get github.com/gorilla/mux
Run the Go Server
bash
Copy
Edit
go run main.go
The server will start on port 8080 by default.

Step 3: Test the API
Use curl to test the API endpoints:

bash
Copy
Edit
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
Verify Replication
You can verify that the data is being properly replicated by connecting to the databases:

bash
Copy
Edit
# On Primary
docker exec -it primary psql -U postgres -d appdb -c "SELECT * FROM users;"

# On Replica
docker exec -it replica psql -U postgres -d appdb -c "SELECT * FROM users;"
Alternatively, use a GUI tool like DBeaver:

Primary: localhost:5432

Replica: localhost:5433

Username: postgres

Password: password

Database: appdb

Troubleshooting
If you encounter issues:

Ensure you are connected to the correct database (appdb).

Check if the Go application is running and has created the necessary tables.

To list all tables:

sql
Copy
Edit
SELECT table_name FROM information_schema.tables WHERE table_schema = 'public';
If the replica is not connecting:

Check logs: docker compose logs replica

Verify network connectivity: docker exec -it replica ping primary

Confirm pg_hba.conf allows connections.

Cleaning Up
To stop and remove the containers:

bash
Copy
Edit
docker compose down

# To also remove volumes (this will delete all data)
docker compose down -v
References
PostgreSQL Replication Documentation

Docker Compose Documentation

Golang PostgreSQL Driver