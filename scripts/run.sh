#!/bin/bash
set -e

DB_PATH=mydatabase.db
LITESTREAM_CONFIG=/config/litestream.yml

if [ -f ${DB_PATH} ]; then
	echo "Database already exists, skipping restore"
else
	echo "No database found, restoring from replica if exists"
	litestream restore -config=${LITESTREAM_CONFIG} --if-replica-exists ${DB_PATH}
	echo "Restored."
fi

exec litestream replicate -exec "./server" --config ${LITESTREAM_CONFIG}
