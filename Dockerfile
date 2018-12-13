FROM alpine:3.7 as alpine
RUN apk add -U --no-cache ca-certificates

# Image starts here
FROM scratch
LABEL maintainer "Lucas Lorentz <lucaslorentzlara@hotmail.com>"

ARG CADDY_EXEC=caddy 

EXPOSE 80 443 2015
ENV HOME /root

WORKDIR /
COPY --from=alpine /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

COPY artifacts/binaries/linux/amd64/$CADDY_EXEC /bin/caddy

ENTRYPOINT ["/bin/caddy"]