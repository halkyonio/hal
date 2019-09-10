FROM alpine:3.10

RUN apk add \
    curl \
    bash

# Add glibc which is needed for oc client
RUN apk --no-cache add ca-certificates wget && \
    wget -q -O /etc/apk/keys/sgerrand.rsa.pub https://alpine-pkgs.sgerrand.com/sgerrand.rsa.pub && \
    wget https://github.com/sgerrand/alpine-pkg-glibc/releases/download/2.30-r0/glibc-2.30-r0.apk && \
    apk add glibc-2.30-r0.apk

ADD https://mirror.openshift.com/pub/openshift-v4/clients/oc/4.1/linux/oc.tar.gz ./
RUN tar -xzf oc.tar.gz && \
    chmod +x oc && \
    mv oc /usr/local/bin/ && \
    rm -rf oc.tar.gz

ENV HOME=/home/hal

# Create non-root user (with a randomly chosen UID/GUI).
RUN adduser hal -Du 2342 -h /home/hal

# Create kube home dir
RUN mkdir $HOME/.kube && touch $HOME/.kube/config
# Change permissions
RUN chmod 775 -R /home/hal

USER hal

ADD ./dist/hal_linux_amd64/hal /usr/local/bin/hal