#!/bin/bash
GIT_NO_INCREMENTAL=1 ./scripts/git.sh --git-stream='' --git-url='git://dpdk.org/dpdk-stable' 2>&1 | tee -a run.log
