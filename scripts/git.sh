#!/bin/bash
# ESENV=prod|test
if [ -z "${ESENV}" ]
then
  ESENV=test
fi
./git --git-url='https://github.com/lukaszgryglicki/trailers-test' --git-es-url="`cat ./secrets/ES_URL.${ESENV}.secret`" $*
