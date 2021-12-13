FROM golang

RUN apt-get update && apt-get install -y ca-certificates git-core ssh

WORKDIR /app

COPY cmd/git/git.go ./
COPY go.mod ./
COPY go.sum ./

# ssh keys to access the private repositories
ADD keys/my_key_rsa /root/.ssh/id_rsa
RUN chmod 700 /root/.ssh/id_rsa
RUN echo "Host github.com\n\tStrictHostKeyChecking no\n" >> /root/.ssh/config
RUN git config --global url.ssh://git@github.com/.insteadOf https://github.com/

RUN go install .

CMD ["git"]
