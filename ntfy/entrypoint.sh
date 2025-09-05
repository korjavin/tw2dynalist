#!/bin/sh

# Exit on error
set -e

# Path to the user database
USER_DB="/var/lib/ntfy/user.db"
CONFIG_FILE="/etc/ntfy/server.yml"

# Check if username and password are provided and if the user.db doesn't exist
if [ -n "$NTFY_USERNAME" ] && [ -n "$NTFY_PASSWORD" ] && [ ! -f "$USER_DB" ]; then
  echo "User database not found. Creating it and adding user '$NTFY_USERNAME'..."
  # The user add command reads the password from stdin
  echo "$NTFY_PASSWORD" | ntfy user add "$NTFY_USERNAME" --role "admin"
  echo "User '$NTFY_USERNAME' added successfully."
  echo "Authentication enabled with deny-all default access."
elif [ -n "$NTFY_USERNAME" ] && [ -n "$NTFY_PASSWORD" ] && [ -f "$USER_DB" ]; then
  echo "User database already exists, skipping user creation."
  echo "Authentication enabled with deny-all default access."
else
  echo "NTFY_USERNAME and/or NTFY_PASSWORD not set."
  echo "Updating config to allow public access..."
  # Update the config to allow public access when no auth is configured
  sed -i 's/auth-default-access: "deny-all"/auth-default-access: "read-write"/' "$CONFIG_FILE"
  echo "The server will start without authentication."
fi

# Execute the command passed into the container, e.g., "serve"
echo "Starting ntfy server..."
exec "$@"
