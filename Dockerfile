# Start from the Go Windows image
FROM golang:1.20-windowsservercore as builder

# Set the working directory
WORKDIR C:\app

# Copy the entire project
COPY . .

# Download dependencies
RUN go mod download

# Build the application
RUN go build -v -o filemodtracker.exe -ldflags="-w -s" .

# Start a new stage from Windows Server Core
FROM mcr.microsoft.com/windows/servercore:ltsc2022

# Set the working directory
WORKDIR C:\app

# Copy the compiled executable from the builder stage
COPY --from=builder C:\app\filemodtracker.exe .

# Copy the config file
COPY config.yaml .

# Expose the API port
EXPOSE 8080

# Set the entry point
ENTRYPOINT ["C:\\app\\filemodtracker.exe"]

# Default command (can be overridden)
CMD ["service"]