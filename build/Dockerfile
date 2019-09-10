FROM alpine:3.8

RUN apk add curl

ADD https://mirror.openshift.com/pub/openshift-v4/clients/oc/4.1/linux/oc.tar.gz ./
RUN tar -xzf oc.tar.gz && \
    chmod +x $PWD/${oc_tool_version}/oc && \
    sudo mv $PWD/${oc_tool_version}/oc /usr/local/bin/ && \
    rm -rf ${oc_tool_version}.tar.gz

USER nobody

ADD ./dist/hal_linux_amd64/hal /usr/local/bin/hal