#!/bin/sh

cd ../ && make && cd tests

for f in *.sh
do
  if [ "$f" != "run_tests.sh" ]; then
    echo "Running $f"
    sh $f

    if [ ! $? -eq 0 ]; then
      echo "$f failed"
      exit 1
    fi
  fi
done

echo "Everything working fine!"
exit 0
