#!/bin/bash

set -e

docker build -t filestation_debian10 .
# Create a temporary container
docker create --name temp_container filestation_debian10

# Copy the compiled binary from the container to the host machine
docker cp temp_container:/app/bin/filestation ./filestation

# Remove the temporary container
docker rm temp_container
