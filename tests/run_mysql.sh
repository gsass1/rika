#!/bin/sh

set -e

echo "Starting MySQL container..."

if [ ! "$(docker ps -a  | grep test-mysql)" ]; then
    docker run --name test-mysql --health-cmd='mysqladmin ping --silent' -e MYSQL_USER=test -e MYSQL_PASSWORD=test -e MYSQL_DATABASE=test -e MYSQL_ROOT_PASSWORD=test -d mysql:latest 1>/dev/null
else 
    docker start test-mysql 1>/dev/null
fi

printf "Waiting for database to start up"

while [ ! $(docker inspect --format {{.State.Health.Status}} test-mysql | grep healthy) ]; do
    printf "%c" .
    sleep 1
done

echo ""

echo "Creating example database entry"
docker exec -t test-mysql mysql -u test -ptest test -e "DROP TABLE IF EXISTS test; CREATE TABLE test (test int); INSERT INTO test (test) VALUES(1337);"

rm -rf ./storage

echo "Running backup"
../rika --verbose run test_mysql.yaml

cleanup() {
    echo "Cleaning up"
    docker rm -f test-mysql 1>/dev/null
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
