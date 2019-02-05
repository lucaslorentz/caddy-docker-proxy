FROM golang:alpine3.9 as build
RUN apk add -U --no-cache ca-certificates git

WORKDIR /src
COPY main.go go.* ./
RUN CGO_ENABLED=0 GOARCH=arm GOARM=7 GOOS=linux go build -o /build/caddy

# Image starts here
FROM alpine:3.9

EXPOSE 80 443 2015
ENV HOME /root

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /build/caddy /bin/

ENTRYPOINT ["/bin/caddy"]
CMD [ "-agree", "-log", "stdout" ]
