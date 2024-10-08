# Configuration Guide for File Modification Tracker

This document outlines the configuration options for the File Modification Tracker application.

## Configuration File

The application uses a YAML configuration file. By default, it looks for `config.yaml` in the following locations:

1. Current directory
2. `$HOME/.filemodtracker/`
3. `/etc/filemodtracker/`

## Configuration Options

| Option            | Description                                             | Default Value                   |
|-------------------|---------------------------------------------------------|---------------------------------|
| `port`            | The port on which the application's HTTP server listens | ":8080"                         |
| `monitor_dir`     | The directory to monitor for file modifications         | "/Users/%%" (current directory) |
| `check_frequency` | How often to check for file modifications               | "1m" (1 minute)                 |
| `timeout`         | Timeout for operations                                  | "1m" (1 minute)                 |
| `port`            | Port for remote reporting                               | ":80"                           |
| `osquery_config`  | Path to the osquery socket                              | "/osquery_fim.conf"             |

## Changing Configuration

You can change the configuration in two ways:

1. Edit the `config.yaml` file directly.
2. Use the CLI command: `filemodtracker config set [key] [value]`

Example:
```
filemodtracker config set monitor_dir /path/to/monitor
```

## Viewing Current Configuration

To view the current configuration, use:

```
filemodtracker config view
```

## Validation

The configuration is validated on application start. If any values are invalid, the application will exit with an error message.