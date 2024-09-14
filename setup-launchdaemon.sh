#!/bin/bash

# Path to your executable
# This will be set by the package installer
EXECUTABLE_PATH="$2/Contents/MacOS/filemodtracker"

# Name for your launch daemon
DAEMON_NAME="com.savannahtech.filemodtracker"

# Create plist file
cat << EOF > /Library/LaunchDaemons/${DAEMON_NAME}.plist
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>${DAEMON_NAME}</string>
    <key>ProgramArguments</key>
    <array>
        <string>${EXECUTABLE_PATH}</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/tmp/${DAEMON_NAME}.stdout</string>
    <key>StandardErrorPath</key>
    <string>/tmp/${DAEMON_NAME}.stderr</string>
</dict>
</plist>
EOF

# Set correct permissions
chown root:wheel /Library/LaunchDaemons/${DAEMON_NAME}.plist
chmod 644 /Library/LaunchDaemons/${DAEMON_NAME}.plist

# Load the launch daemon
launchctl load /Library/LaunchDaemons/${DAEMON_NAME}.plist

echo "Launch daemon has been created and loaded."
echo "Your application will now run with elevated privileges at system startup."