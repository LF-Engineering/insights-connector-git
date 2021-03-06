#!/bin/bash
if [ ! -z "$CONSOLE" ]
then
  GIT_TAGS="c,d,e" ./scripts/git.sh --git-url='https://github.com/LF-engineering/insights-datasource-git' --git-date-from "2015-01" --git-date-to "2023-01" --git-pack-size=100 --git-tags="a,b,c" --git-project=Kubernetes --git-skip-cache-cleanup=1 --git-stream='' $*
else
  GIT_TAGS="c,d,e" ./scripts/git.sh --git-url='https://github.com/LF-engineering/insights-datasource-git' --git-date-from "2015-01" --git-date-to "2023-01" --git-pack-size=100 --git-tags="a,b,c" --git-project=Kubernetes --git-skip-cache-cleanup=1 $*
fi
