#!/bin/bash
if ! command -v osqueryi &> /dev/null
then
    echo "osquery is not installed. Please install it from https://osquery.io/"
    exit 1
fi
