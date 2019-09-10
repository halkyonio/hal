FROM frolvlad/alpine-glibc:alpine-3.10

RUN apk add \
    curl \
    bash \
    'musl>1.1.22-r3'

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