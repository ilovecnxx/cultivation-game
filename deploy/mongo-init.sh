#!/bin/bash
# Wait for MongoDB to be ready
until mongosh --quiet --eval "db.adminCommand('ping').ok" 2>/dev/null | grep -q 1; do
  sleep 2
done
# Initialize replica set
mongosh --quiet --eval "
rs.initiate({_id: 'rs0', members: [{_id: 0, host: 'localhost:27017'}]})
" 2>/dev/null
echo "MongoDB replica set initialized"
