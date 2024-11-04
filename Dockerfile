FROM --platform=${BUILDPLATFORM} alpine:3.20.3 as alpine
RUN apk add -U --no-cache ca-certificates

# Image starts here
FROM scratch
ARG TARGETPLATFORM
LABEL maintainer "Lucas Lorentz <lucaslorentzlara@hotmail.com>"

EXPOSE 80 443 2019
ENV XDG_CONFIG_HOME /config
ENV XDG_DATA_HOME /data

WORKDIR /

COPY --from=alpine /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

COPY artifacts/binaries/$TARGETPLATFORM/caddy /bin/

ENTRYPOINT ["/bin/caddy"]

CMD ["docker-proxy"]