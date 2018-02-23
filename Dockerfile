FROM alpine:3.7
LABEL maintainer "Lucas Lorentz <lucaslorentzlara@hotmail.com>"

EXPOSE 80 443 2015

ADD caddy /usr/bin

RUN apk add --no-cache ca-certificates curl

ENTRYPOINT ["/usr/bin/caddy"]