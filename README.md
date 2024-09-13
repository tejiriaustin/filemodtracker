# File Modification Tracker

This project is a File Modification Tracker implemented in Go, designed to run as a background service. It tracks and records modifications to files in a specified directory, integrates system monitoring via osquery, and provides configuration management.

## Prerequisites

- Go (version 1.20 or later)
- osquery
- make (for running Makefile commands)

## Installation

1. Clone the repository:
   ```
   git clone https://github.com/tejiriaustin/filemodtracker.git
   cd filemodtracker
   ```

2. Build the project:
   ```
   make build
   ```

## Usage

### Running the Service and UI

To run both the service and UI components:

```
make run
```

### Running Only the Service

To run only the background service:

```
make run-service
```

### Running Only the UI

To run only the UI component:

```
make run-ui
```

## Development

### Building

To build the project:

```
make build
```

### Cleaning

To clean the build artifacts and Go mod cache:

```
make clean
```

### Testing

To run tests with coverage:

```
make test
```

### Mocking

To generate mocks for testing:

```
make mocks
```

To remove existing mocks:

```
make rm-mocks
```

## Project Structure

- `cmd/`: Entry points for running commands from the command-line.
- `config/`: Handles configuration management using viper.
- `daemon/`: Contains the implementation of the background service.
- `monitoring/`: Implements file monitoring logic, potentially using osquery.
- `server/`: The entry point of the application.
- `ui/`: Contains the implementation of the UI component.
- `main.go`: The entry point of the application.

## Configuration

The project uses a configuration file to manage service settings. Ensure you have a properly configured file before running the service.

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## Contact

Your Name - tejiiaustin123@example.com

Project Link: [https://github.com/tejiriaustin/filemodtracker](https://github.com/tejiriaustin/filemodtracker)