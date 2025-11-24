FROM alpine:3.20.1

ARG REF=local
LABEL org.opencontainers.image.authors="Mirantis Inc. https://mirantis.com" \
      org.opencontainers.image.source="https://github.com/Mirantis/pelagia" \
      org.opencontainers.image.ref.commit="${REF}"

# rebuild=1
ENV USER_UID=1001 \
    USER_GID=1001 \
    USER_NAME=pelagia-ceph \
    CONTROLLER_NAME=pelagia-ceph

RUN apk update \
     && apk upgrade -U --no-cache \
     && apk add shadow --no-cache

COPY build/docker build/bin /usr/local/bin
RUN  /usr/local/bin/user_setup

# save tini for disk-daemon init process to reap zombies
RUN wget https://github.com/krallin/tini/releases/download/v0.19.0/tini -O /usr/local/bin/tini \
     && chmod +x /usr/local/bin/tini

USER ${USER_UID}:${USER_GID}
