#!/bin/bash
clear; GIT_TAGS="c,d,e" ./scripts/git.sh --git-date-from "2021-01" --git-date-to "2021-08" --git-pack-size=100 --git-tags="a,b,c" --git-project=Kubernetes --git-debug=2 $*
