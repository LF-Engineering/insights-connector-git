FROM alpine:3.14

WORKDIR /app

COPY --from=golang:1.15-alpine /usr/local/go/ /usr/local/go/

ENV PATH="/usr/local/go/bin:${PATH}"
COPY . /
RUN make git
ENV REPO_URL='<GIT-REPO-URL>'
ENV ES_URL='<GIT-ES-URL>'
ENV STAGE='<STAGE>'
RUN apk update && apk add git
RUN apk add cloc
RUN apk add --no-cache bash
RUN ls -ltra
COPY git ./
COPY gitops /usr/bin/
COPY detect-removed-commits.sh /usr/bin/

CMD ./git --git-url=${REPO_URL} --git-es-url=${ES_URL}