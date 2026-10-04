.PHONY: all up down gen migrate

all: gen migrate up

up:
	docker compose -f docker-compose.dev.yaml up --build

down:
	docker compose -f docker-compose.dev.yaml down

gen:
	buf generate

migrate:
	docker compose -f docker-compose.dev.yaml run --rm migrations-configuration up
	docker compose -f docker-compose.dev.yaml run --rm migrations-catalog up