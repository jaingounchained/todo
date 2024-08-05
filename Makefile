DB_URL=postgresql://root:secret@localhost:5432/simple_bank?sslmode=disable

network:
	docker network create todo-network

postgres:
	docker run --name postgres --network todo-network -p 5432:5432 -e POSTGRES_USER=root -e POSTGRES_PASSWORD=secret -d postgres:14-alpine

createdb:
	docker exec -it postgres createdb --username=root --owner=root todos

dropdb:
	docker exec -it postgres dropdb todos

migrateup:
	migrate -path db/migration -database "$(DB_URL)" -verbose up

migrateup1:
	migrate -path db/migration -database "$(DB_URL)" -verbose up 1

migratedown:
	migrate -path db/migration -database "$(DB_URL)" -verbose down

migratedown1:
	migrate -path db/migration -database "$(DB_URL)" -verbose down 1

clearlocalteststorage:
	rm -rf ./uploads/

sqlc:
	sqlc generate

test:
	go test -v -cover -short ./...

server:
	go run main.go

mocksql:
	mockgen -package mockdb -destination db/mock/store.go github.com/jaingounchained/todo/db/sqlc Store

mockstorage:
	mockgen -package mockStorage -destination storage/mock/storage.go github.com/jaingounchained/todo/storage Storage

dockerbuild:
	docker build -t todos:latest .

openapispec:
	swag init

.PHONY: network postgres createdb dropdb migrateup migrateup1 migratedown migratedown1 sqlc server mocksql mockstorage clearlocalteststorage dockerbuild openapispec
