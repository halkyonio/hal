FROM frolvlad/alpine-glibc:alpine-3.10

RUN apk add \
    curl \
    bash \
    busybox-suid

ADD https://mirror.openshift.com/pub/openshift-v4/clients/oc/4.1/linux/oc.tar.gz ./
RUN tar -xzf oc.tar.gz && \
    chmod +x oc && \
    mv oc /usr/local/bin/ && \
    rm -rf oc.tar.gz

# Create non-root user (with a randomly chosen UID/GUI).
RUN adduser hal -Du 2342 -h /home

# Create kube home dir
RUN mkdir $HOME/.kube && touch $HOME/.kube/config

USER hal

ADD ./dist/hal_linux_amd64/hal /usr/local/bin/hal