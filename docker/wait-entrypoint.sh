#!/bin/sh

# Blow up if EMISSARY_CONFIG is not set
if [ -z "$EMISSARY_CONFIG" ]; then
  echo "EMISSARY_CONFIG must be set. Exiting."
  exit 1
fi

# Wait for the config file to exist.
CONFIG_PATH="${EMISSARY_CONFIG#file://}"
echo "Waiting for config file to exist at ${CONFIG_PATH}..."
while [ ! -f ${CONFIG_PATH} ]; do
  sleep 0.25
done

# After that, emissary will use fnotify to watch for changes to the config file
# CONSIDER: have emissary itself gracefully handle config-doesn't-exist-yet case?
exec /app/emissary "$@"

