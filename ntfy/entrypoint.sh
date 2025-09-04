#!/bin/sh

# Exit on error
set -e

# Path to the user database
USER_DB="/var/lib/ntfy/user.db"

# Check if username and password are provided and if the user.db doesn't exist
if [ -n "$NTFY_USERNAME" ] && [ -n "$NTFY_PASSWORD" ] && [ ! -f "$USER_DB" ]; then
  echo "User database not found. Creating it and adding user '$NTFY_USERNAME'..."
  # The user add command reads the password from stdin
  echo "$NTFY_PASSWORD" | ntfy user add "$NTFY_USERNAME" --role "admin"
  echo "User '$NTFY_USERNAME' added successfully."
elif [ -n "$NTFY_USERNAME" ] && [ -n "$NTFY_PASSWORD" ] && [ -f "$USER_DB" ]; then
  echo "User database already exists, skipping user creation."
else
  echo "NTFY_USERNAME and/or NTFY_PASSWORD not set. Skipping user creation."
  echo "The server will start without authentication."
fi

# Execute the command passed into the container, e.g., "serve"
echo "Starting ntfy server..."
exec "$@"
