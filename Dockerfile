###################################################
# THIS IS EXPERIMENTAL
###################################################
# Use the Go 1.23 alpine official image
# https://hub.docker.com/_/golang
FROM golang:1.25-alpine AS builder 

# Define build argument for the application folder in cmd directory
# Force this to be specified
ARG CMD_FOLDER=invalid 

# Define build argument for insecure flag (defaults to true for dev/local)
ARG INSECURE=true

# Create and change to the app directory.
WORKDIR /build

# Copy go mod and sum files
COPY go.mod go.sum ./
# COPY go.mod ./

# Copy local code to the container image.
COPY . ./

# Install project dependencies
RUN go mod download

# Build the app from the specified cmd folder with linker flags
# RUN go build -ldflags="-X 'github.com/scalesql/portal/internal/build.insecure=${INSECURE}'" -o appexe ./cmd/${CMD_FOLDER}
RUN go build -o sqlxewriterapp ./cmd/sqlxewriter

# Run the service on container startup.
# ENTRYPOINT ["./app"]
######################################################
# Second Stage
######################################################
FROM alpine:latest

# Install CA certificates for HTTPS calls
# RUN apk add --no-cache ca-certificates

WORKDIR /app

# Copy only the compiled binary
COPY --from=builder /build/sqlxewriterapp /usr/local/bin/

EXPOSE 8080

ENTRYPOINT ["sqlxewriterapp"]