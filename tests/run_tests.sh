#!/bin/sh

cd ../ && make && cd tests

find . -iname "run_*" -type f -not -path "./run_tests.sh" -exec sh {} \;
