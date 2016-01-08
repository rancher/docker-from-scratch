#!/bin/bash
set -e

cd $(dirname $0)

dapper ./scripts/build

echo Done
