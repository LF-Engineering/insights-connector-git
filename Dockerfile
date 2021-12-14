FROM alpine:3.14

WORKDIR /app

RUN apk update && apk add git

COPY git ./
COPY gitops ./
COPY detect-removed-commits.sh ./

ENTRYPOINT ["./git"]
