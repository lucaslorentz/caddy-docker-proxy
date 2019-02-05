TAG=vlkoti/caddy-docker-proxy:latest

all: image

image: Dockerfile
	docker build -t $(TAG) .

push:
	docker push $(TAG)
