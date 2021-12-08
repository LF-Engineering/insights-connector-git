#!/bin/bash
# example run: GIT_STREAM=xyz ./scripts/git.sh
# ESENV=prod|test
if [ -z "${ESENV}" ]
then
  ESENV=test
fi
# AWSENV=prod|test|dev
if [ -z "${AWSENV}" ]
then
  AWSENV=dev
fi
export AWS_ACCESS_KEY_ID="`cat ./secrets/AWS_ACCESS_KEY_ID.${AWSENV}.secret`"
export AWS_REGION="`cat ./secrets/AWS_REGION.${AWSENV}.secret`"
export AWS_SECRET_ACCESS_KEY="`cat ./secrets/AWS_SECRET_ACCESS_KEY.${AWSENV}.secret`"
./git --git-es-url="`cat ./secrets/ES_URL.${ESENV}.secret`" $*
