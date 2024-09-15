#!/bin/bash

# Explicitly copy the executable
cp /private/tmp/filemodtracker /usr/local/bin/filemodtracker

# Set correct permissions for the executable
chmod 755 /usr/local/bin/filemodtracker

# Load and start the daemon
launchctl load /Library/LaunchDaemons/com.example.filemodtracker.daemon.plist
launchctl start com.example.filemodtracker.daemon

# Copy the UI launchd plist to all existing user LaunchAgents folders
for userDir in /Users/*; do
    if [ -d "$userDir" ]; then
        username=$(basename "$userDir")
        mkdir -p "$userDir/Library/LaunchAgents"
        cp /Library/LaunchAgents/com.example.filemodtracker.ui.plist "$userDir/Library/LaunchAgents/"
        chown "$username:staff" "$userDir/Library/LaunchAgents/com.example.filemodtracker.ui.plist"
    fi
done

# Load the UI launchd job for the current console user
currentUser=$(/usr/bin/stat -f "%Su" /dev/console)
su - "$currentUser" -c "launchctl load ~/Library/LaunchAgents/com.example.filemodtracker.ui.plist"

exit 0