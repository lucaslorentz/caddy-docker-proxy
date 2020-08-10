ARG CADDY_VERSION=2.1.1

FROM caddy:${CADDY_VERSION}-builder AS builder

ARG PLUGINS=

RUN caddy-builder \
    github.com/lucaslorentz/caddy-docker-proxy/plugin/v2 \
    ${PLUGINS}

FROM caddy:${CADDY_VERSION}

COPY --from=builder /usr/bin/caddy /usr/bin/caddy