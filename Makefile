
build:
	docker build -f Dockerfile.dev --target build --tag build .
	docker build -f Dockerfile.dev --target cdp --tag cdp .

run:
	docker-compose up

list:
	docker run --rm -it cdp caddy list-modules