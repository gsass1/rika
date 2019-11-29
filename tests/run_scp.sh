#!/bin/sh

set -e 

echo "Generating key..."

rm -rf keys
mkdir -p keys
ssh-keygen -t rsa -b 4096 -f keys/test -N "" 1>/dev/null

chmod 600 keys/*

echo "Starting SSH container..."

UID=$(id -u $(whoami))

rm -rf scp-data
mkdir scp-data

docker run --name test-scp -d -p 2222:22 \
  -v ${PWD}/keys/test.pub:/etc/authorized_keys/test:ro \
  -v $(pwd)/scp-data/:/scp-data/ \
  -e SSH_USERS="test:$UID:$UID" \
  docker.io/panubo/sshd:1.1.0 1>/dev/null

printf "Waiting for SSH server to come online"

set +e

while true; do
  ssh -q -o "UserKnownHostsFile /dev/null" -o "StrictHostKeyChecking no" -p 2222 -i keys/test test@localhost exit

  if [ $? -eq 0 ]; then
    break
  fi

  printf "."
  sleep 1
done

set -e

echo ""

rm -rf volume
mkdir volume

echo "Test File" > volume/file
echo "Test File 2" > volume/file2

sleep 2

echo "Running backup"
../rika --verbose run test_scp.yaml

cleanup() {
    echo "Cleaning up"
    docker rm -f test-scp 1>/dev/null
    rm -rf volume
    rm -rf scp-data
    rm -rf keys
}

cd scp-data
tar -xf *

cd volume

if [ ! -f "file" ]; then
    echo "File is missing"
    cd ../../
    cleanup
    exit 1
fi

if [ ! -f "file2" ]; then
    echo "File 2 is missing"
    cd ../../
    cleanup
    exit 1
fi

CONTENTS=$(cat "file")

if [ "$CONTENTS" != "Test File" ]; then
    echo "Wrong file contents"
    cd ../../
    cleanup
    exit 1
fi

cd ../../

echo "Success!"

cleanup

exit 0
