DB_URL=postgresql://root:secret@localhost:5432/todos?sslmode=disable

network:
	docker network create todo-network

postgresstart:
	docker run --rm --name postgres -p 5432:5432 -e POSTGRES_USER=root -e POSTGRES_PASSWORD=secret -d postgres:14-alpine

postgresstop:
	docker stop postgres

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

createlocalteststorage:
	mkdir uploads

clearlocalteststorage:
	rm -rf uploads

sqlc:
	sqlc generate

test:
	go test -cover -count=1 ./...

testverbose:
	go test -v -cover -count=1 ./...

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

proto:
	rm -f pb/*.go
	protoc --proto_path=proto --go_out=pb --go_opt=paths=source_relative \
	--go-grpc_out=pb --go-grpc_opt=paths=source_relative \
	--grpc-gateway_out=pb --grpc-gateway_opt=paths=source_relative \
	proto/*.proto

evans:
	evans --host localhost --port 9090 -r repl

.PHONY: network postgresstart postgresstop createdb dropdb migrateup migrateup1 \
		migratedown migratedown1 sqlc server mocksql mockstorage clearlocalteststorage \
		dockerbuild openapispec createlocalteststorage testverbose proto evans
