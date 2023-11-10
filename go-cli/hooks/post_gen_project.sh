#!/bin/bash

mkdir -p ./scripts/libs
curl https://raw.githubusercontent.com/hpcsc/repo-templates/main/global/libs/git-hook-prepush.sh > ./scripts/libs/git-hook-prepush.sh
chmod +x ./scripts/libs/git-hook-prepush.sh
