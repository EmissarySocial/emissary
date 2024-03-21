#!/bin/bash
mongorestore /data/dump/
docker-entrypoint.sh mongod
