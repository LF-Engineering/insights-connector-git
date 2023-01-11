FROM alpine:3.14

WORKDIR /app

ENV GIT_REPO_URL='<GIT-REPO-URL>'
ENV ES_URL='<GIT-ES-URL>'
ENV STAGE='<STAGE>'
ENV ELASTIC_LOG_URL='<ELASTIC-LOG-URL>'
ENV ELASTIC_LOG_USER='<ELASTIC-LOG-USER>'
ENV ELASTIC_LOG_PASSWORD='<ELASTIC-LOG-PASSWORD>'
ENV GIT_SOURCE_ID='<GIT_SOURCE_ID>'
ENV GIT_REPOSITORY_SOURCE='<GIT_REPOSITORY_SOURCE>'
ENV LAST_SYNC='<LAST_SYNC>'
ENV SPAN='<SPAN>'
RUN apk update && apk add git
RUN apk add cloc
RUN apk add --no-cache bash
RUN ls -ltra
COPY git ./
COPY gitops /usr/bin/
COPY detect-removed-commits.sh /usr/bin/

CMD ./git --git-url=${GIT_REPO_URL} --git-es-url=${ES_URL} --git-source-id=${GIT_SOURCE_ID} --git-repository-source=${GIT_REPOSITORY_SOURCE}
