DB_URL=postgresql://root:secret@localhost:5432/todos?sslmode=disable

network:
	docker network create todo-network

postgres-start:
	docker run --rm --name postgres -p 5432:5432 -e POSTGRES_USER=root -e POSTGRES_PASSWORD=secret -d postgres:14-alpine

postgres-stop:
	docker stop postgres

create-db:
	docker exec -it postgres createdb --username=root --owner=root todos

drop-db:
	docker exec -it postgres dropdb todos

migrateup:
	migrate -path db/migration -database "$(DB_URL)" -verbose up

migrateup1:
	migrate -path db/migration -database "$(DB_URL)" -verbose up 1

migratedown:
	migrate -path db/migration -database "$(DB_URL)" -verbose down

migratedown1:
	migrate -path db/migration -database "$(DB_URL)" -verbose down 1

create-local-test-storage:
	mkdir uploads

clear-local-test-storage:
	rm -rf uploads

sqlc:
	sqlc generate

test:
	go test -cover -count=1 ./...

test-verbose:
	go test -v -cover -count=1 ./...

server:
	go run main.go

mock-db:
	mockgen -package mockdb -destination db/mock/store.go github.com/jaingounchained/todo/db/sqlc Store

mock-wk:
	mockgen -package mockwk -destination worker/mock/distributor.go github.com/jaingounchained/todo/worker TaskDistributor

mock-storage:
	mockgen -package mockstorage -destination storage/mock/storage.go github.com/jaingounchained/todo/storage Storage

docker-build:
	docker build -t todos:latest .

swagger-generate:
	swag init

proto:
	rm -f pb/*.go
	protoc --proto_path=proto --go_out=pb --go_opt=paths=source_relative \
	--go-grpc_out=pb --go-grpc_opt=paths=source_relative \
	--grpc-gateway_out=pb --grpc-gateway_opt=paths=source_relative \
	proto/*.proto

evans:
	evans --host localhost --port 9090 -r repl

redis-start:
	docker run --rm --name redis -p 6379:6379 -d redis:7-alpine

redis-stop:
	docker stop redis

new-migration:
	migrate create -ext sql -dir db/migration -seq $(name)

.PHONY: network postgres-start postgres-stop create-db drop-db migrateup migrateup1 \
		migratedown migratedown1 sqlc server mock-db mock-storage clear-local-test-storage \
		docker-build swagger-generate create-local-test-storage test-verbose proto evans \
		new-migration mock-wk redis-start redis-stop
