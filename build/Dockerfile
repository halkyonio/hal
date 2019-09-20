FROM alpine:3.10

RUN apk add \
    curl \
    bash

# Add glibc which is needed for oc client
RUN echo http://nl.alpinelinux.org/alpine/edge/testing >> /etc/apk/repositories && \
    apk --no-cache add ca-certificates wget && \
    wget -q -O /etc/apk/keys/sgerrand.rsa.pub https://alpine-pkgs.sgerrand.com/sgerrand.rsa.pub && \
    wget https://github.com/sgerrand/alpine-pkg-glibc/releases/download/2.30-r0/glibc-2.30-r0.apk && \
    apk --no-cache --update add glibc-2.30-r0.apk gosu

ADD https://mirror.openshift.com/pub/openshift-v4/clients/oc/4.1/linux/oc.tar.gz ./
RUN tar -xzf oc.tar.gz && \
    chmod +x oc && \
    mv oc /usr/local/bin/ && \
    rm -rf oc.tar.gz

ARG username=hal
ARG userid=1000

ENV LOCAL_USER_NAME=$username
ENV LOCAL_USER_ID=$userid

ADD ./dist/hal_linux_amd64/hal /usr/local/bin/hal
COPY ./build/entrypoint.sh /usr/local/bin/adduser-and-run.sh

WORKDIR /home/$username

ENTRYPOINT ["adduser-and-run.sh"]

