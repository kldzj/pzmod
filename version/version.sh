#!/bin/bash
version=$(git describe --tags --abbrev=0)
echo -n $version > ./version.txt