FROM golang:1.14-alpine

RUN addgroup -S ory; \
    adduser -S ory -G ory -D -H -s /bin/nologin

RUN apk add -U --no-cache ca-certificates

ENV GO111MODULE on

WORKDIR /go/src/app

ADD go.mod go.mod
ADD go.sum go.sum
RUN go mod download

ADD . .

RUN go install

USER ory

ENTRYPOINT ["slackinviter"]
