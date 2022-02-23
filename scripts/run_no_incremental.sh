#!/bin/bash
GIT_NO_INCREMENTAL=1 ./scripts/git.sh --git-stream='' --git-url='https://github.com/magma/magma-website' 2>&1 | tee -a run.log
