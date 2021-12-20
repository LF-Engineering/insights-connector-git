FROM alpine:3.14

WORKDIR /app

ENV REPO='--git-url=<GIT-REPO-URL>'
ENV ES='--git-es-url=<GIT-ES-URL>'
RUN apk update && apk add git
RUN apk add cloc
RUN apk add --no-cache bash
COPY git ./
COPY gitops /usr/bin/
COPY detect-removed-commits.sh /usr/bin/

CMD ./git ${REPO} ${ES}
