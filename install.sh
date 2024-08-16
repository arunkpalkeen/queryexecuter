#!/bin/bash
#Installer for Query excuter program 
#Auther AKP
#Property K&A
#Date 2024Aug15


# Function to check if a command exists
command_exists() {
    command -v "$1" &> /dev/null
}

# Function to check if a file or directory exists
check_exists() {
    if [ ! -e "$1" ]; then
        echo "ERROR: $1 not found. Please ensure the required file or directory is present and try again."
        exit 1
    fi
}

# Function to check if a Podman container exists
container_exists() {
    podman ps -a --filter "name=$1" --format "{{.Names}}" | grep -w "$1" &> /dev/null
}

# Check if all required files and directories exist
echo "Checking if all required files and directories exist..."
check_exists "query-executer"
check_exists "db_config.json"
check_exists "static"
check_exists "static/favicon.ico"
check_exists "static/logo.png"
check_exists "static/main.js"
check_exists "static/style.css"
check_exists "templates"
check_exists "templates/index.html"
check_exists "templates/login.html"
check_exists "templates/queries.html"
echo "All required files and directories are present."

# Check if Podman is installed
echo "Checking if Podman is installed..."
if ! command_exists podman; then
    echo "ERROR: Podman is not installed. Please install Podman and try again."
    exit 1
else
    echo "Podman is installed."
fi

# Check if the PostgreSQL image is available
IMAGE_NAME="postgres:14"
echo "Checking if PostgreSQL image $IMAGE_NAME is available..."
if ! podman image exists $IMAGE_NAME; then
    echo "PostgreSQL image $IMAGE_NAME not found. Pulling the image..."
    podman pull $IMAGE_NAME
    if [ $? -ne 0 ]; then
        echo "ERROR: Failed to pull the PostgreSQL image. Please check your internet connection or the image name and try again."
        exit 1
    else
        echo "PostgreSQL image $IMAGE_NAME successfully pulled."
    fi
else
    echo "PostgreSQL image $IMAGE_NAME is already available."
fi

# Read the DB port from db_config.json
CONFIG_FILE="db_config.json"
echo "Reading the database port from $CONFIG_FILE..."
if [ -f "$CONFIG_FILE" ]; then
    DB_PORT=$(jq -r '.databases[] | select(.name=="Local Database") | .port' $CONFIG_FILE)
    if [ -z "$DB_PORT" ]; then
        echo "ERROR: No port defined for Local Database in $CONFIG_FILE. Exiting."
        exit 1
    else
        echo "Port $DB_PORT found for Local Database."
    fi
else
    echo "ERROR: $CONFIG_FILE not found. Please provide the configuration file and try again."
    exit 1
fi

# Check if the specified port is available
echo "Checking if port $DB_PORT is available..."
if lsof -Pi :$DB_PORT -sTCP:LISTEN -t >/dev/null ; then
    echo "ERROR: Port $DB_PORT is already in use. Please choose a different port in db_config.json and re-run the script."
    exit 1
else
    echo "Port $DB_PORT is available."
fi

# Check if the container with the name "local-postgres" already exists
CONTAINER_NAME="local-postgres"
echo "Checking if container $CONTAINER_NAME already exists..."
if container_exists "$CONTAINER_NAME"; then
    echo "ERROR: A container with the name $CONTAINER_NAME already exists. Please remove the existing container or use a different name."
    exit 1
else
    echo "No existing container with the name $CONTAINER_NAME found."
fi

# Prompt for the volume path to mount
read -p "Please enter the host path to mount as a volume for PostgreSQL data (default: /home/arun/work/query-executer/postgres_data): " VOLUME_PATH
VOLUME_PATH=${VOLUME_PATH:-/home/arun/work/query-executer/postgres_data}

# Check if the volume directory exists
if [ ! -d "$VOLUME_PATH" ]; then
    echo "ERROR: The directory $VOLUME_PATH does not exist. Please create it and re-run the script."
    exit 1
fi

# Run the PostgreSQL container using Podman
PODMAN_COMMAND="podman run --name $CONTAINER_NAME \
  -e POSTGRES_USER=localdbuser \
  -e POSTGRES_PASSWORD=localdb123 \
  -e POSTGRES_DB=localdb \
  -v $VOLUME_PATH:/var/lib/postgresql/data \
  -p $DB_PORT:5432 \
  -d $IMAGE_NAME"

echo "Running PostgreSQL container with command: $PODMAN_COMMAND"
eval $PODMAN_COMMAND

# Wait for PostgreSQL to start
echo "Waiting for PostgreSQL to start..."
sleep 10

# Check if PostgreSQL is running
echo "Checking if PostgreSQL container is running..."
if ! podman ps --filter "name=$CONTAINER_NAME" --filter "status=running" | grep $CONTAINER_NAME &> /dev/null; then
    echo "ERROR: PostgreSQL container failed to start. Please check the logs and try again."
    exit 1
else
    echo "PostgreSQL container is running."
fi

# Set up the local database
SETUP_SQL="
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(255) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL
);

CREATE TABLE IF NOT EXISTS submitted_queries (
    id SERIAL PRIMARY KEY,
    query_text TEXT NOT NULL,
    submitted_by VARCHAR(255) NOT NULL,
    approved_by VARCHAR(255),
    target_db VARCHAR(255),
    execution_time TIMESTAMP NOT NULL,
    execution_duration INTERVAL,
    output TEXT,
    status VARCHAR(50) NOT NULL
);

INSERT INTO users (username, password) VALUES ('user1', 'Redhat@123');
"

echo "Setting up the local database and inserting default user..."
# Run the setup SQL script inside the container
podman exec -i local-postgres psql -U localdbuser -d localdb -c "$SETUP_SQL"

# Check if the setup was successful
if [ $? -eq 0 ]; then
    echo "Local database setup completed successfully, and default user inserted."
else
    echo "ERROR: Failed to set up the local database."
    exit 1
fi

echo "Installation complete!"

