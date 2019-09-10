FROM alpine:3.8

RUN apk add curl
ADD https://storage.googleapis.com/kubernetes-release/release/v1.14.6/bin/linux/amd64/kubectl /usr/local/bin/kubectl

USER nobody

ADD ./dist/hal_linux_amd64/hal /usr/local/bin/hal