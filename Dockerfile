FROM alpine:3.14

WORKDIR /app

ENV REPO_URL='--git-url=<GIT-REPO-URL>'
ENV ES_URL='--git-es-url=<GIT-ES-URL>'
RUN apk update && apk add git
RUN apk add cloc
RUN apk add --no-cache bash
COPY git ./
COPY gitops /usr/bin/
COPY detect-removed-commits.sh /usr/bin/

CMD ./git ${REPO_URL} ${ES_URL}
