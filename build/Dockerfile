FROM frolvlad/alpine-glibc:3.10

RUN apk add curl

ADD https://mirror.openshift.com/pub/openshift-v4/clients/oc/4.1/linux/oc.tar.gz ./
RUN tar -xzf oc.tar.gz && \
    chmod +x oc && \
    mv oc /usr/local/bin/ && \
    rm -rf oc.tar.gz

USER nobody

ADD ./dist/hal_linux_amd64/hal /usr/local/bin/hal