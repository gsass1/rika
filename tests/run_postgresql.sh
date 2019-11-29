#!/bin/sh

set -e

echo "Starting PostgreSQL container..."

if [ ! "$(docker ps -a  | grep test-postgres)" ]; then
    docker run --name test-postgres --health-cmd='pg_isready -U test' -e POSTGRES_USER=test -e POSTGRES_PASSWORD=test -e POSTGRES_DATABASE=test -e POSTGRES_ROOT_PASSWORD=test -d postgres:12.1-alpine 1>/dev/null
else 
    docker start test-postgres 1>/dev/null
fi

printf "Waiting for database to start up"

while [ ! $(docker inspect --format {{.State.Health.Status}} test-postgres | grep healthy) ]; do
    printf "%c" .
    sleep 1
done

echo ""

echo "Creating example database entry"
docker exec -t test-postgres psql -U test test -c "DROP TABLE IF EXISTS test; CREATE TABLE test (test int); INSERT INTO test (test) VALUES(1337);"

rm -rf ./storage

echo "Running backup"
../rika --verbose run test_postgresql.yaml

cleanup() {
    echo "Cleaning up"
    docker rm -f test-postgres 1>/dev/null
}

DUMPFILE=$(find storage -iname "*.sql.xz" -type f)

if [ ! -f $DUMPFILE ]; then
    echo "Did not produce a .sql.xz file!"
    cleanup
    exit 1
fi

if [ ! "$(xzcat $DUMPFILE | grep 1337 )" ]; then
    echo "Dump file did not contain 1337"
    cleanup
    exit 1
fi

echo "Success!"

cleanup
