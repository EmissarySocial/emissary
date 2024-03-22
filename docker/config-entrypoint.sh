#!/bin/sh

# Wait for the config file to exist
# CONSIDER: use an env var for the config path?
echo "Waiting for config file.to exist..."
while [ ! -f /data/config/config.json ]; do
  sleep 0.25
done

# After that, emissary will use fnotify to watch for changes to the config file
# CONSIDER: have emissary itself gracefully handle config-doesn't-exist-yet case?
exec /app/emissary "$@"

