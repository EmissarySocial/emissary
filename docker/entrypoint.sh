#!/bin/sh

# Blow up if EMISSARY_CONFIG is not set
if [ -z "$EMISSARY_CONFIG" ]; then
  echo "EMISSARY_CONFIG must be set. Exiting."
  exit 1
fi

# If WAIT_FOR_CONFIG is set, wait for the config file to exist
# before starting emissary.
if [ -n "$WAIT_FOR_CONFIG" ]; then
  CONFIG_PATH="${EMISSARY_CONFIG#file://}"
  echo "Waiting for config file to exist at ${CONFIG_PATH}..."
  while [ ! -f ${CONFIG_PATH} ]; do
    sleep 0.25
  done
fi

# Start emissary with the provided arguments
exec /app/emissary "$@"

