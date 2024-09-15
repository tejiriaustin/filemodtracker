# Installation Guide for File Modification Tracker

This guide will walk you through the process of installing and uninstalling the File Modification Tracker on macOS.

## Installation

1. Download the `FileModTracker_Installer2.pkg` file.

2. Double-click the `.pkg` file to start the installation process.

3. Follow the on-screen instructions in the installer.

4. Once installed, the application will be available as `filemodtracker` in your terminal.

5. To start the daemon, run:
   ```
   sudo filemodtracker daemon
   ```

6. To start the daemon, run:
   ```
   filemodtracker ui
   ```

## Configuration

After installation, you can configure the application by editing the `config.yaml` file located at `~/.filemodtracker/config.yaml` or using the CLI commands. See CONFIG.md for more details.

## Uninstallation

To uninstall the File Modification Tracker:

1. Stop the service if it's running:
   ```
   sudo filemodtracker stop
   ```

2. Run the following command to remove the application:
   ```
   sudo /usr/local/bin/filemodtracker_uninstall
   ```

3. This script will remove:
   - The `filemodtracker` binary
   - Configuration files
   - Any created log files

4. You may need to manually remove any data in the monitored directory that was created by the application.

## Troubleshooting

If you encounter any issues during installation or uninstallation, please check the following:

1. Ensure you have administrator privileges.
2. Check system logs for any error messages.
3. Verify that osquery is installed and running on your system.

For further assistance, please contact support at tejiriaustin123@gmail.com.
