#!/bin/bash
# ESENV=prod|test
if [ -z "${ESENV}" ]
then
  ESENV=test
fi
./git --git-url='https://github.com/cncf/devstats-helm' --git-es-url="`cat ./secrets/ES_URL.${ESENV}.secret`" $*
