postgres:
	docker run --name postgres -p 5432:5432 -e POSTGRES_USER=root -e POSTGRES_PASSWORD=secret -d postgres:14-alpine

createdb:
	docker exec -it postgres createdb --username=root --owner=root todos

dropdb:
	docker exec -it postgres dropdb todos

migrateup:
	migrate -path db/migration -database "postgresql://root:secret@localhost:5432/todos?sslmode=disable" -verbose up

migrateup1:
	migrate -path db/migration -database "postgresql://root:secret@localhost:5432/todos?sslmode=disable" -verbose up 1

migratedown:
	migrate -path db/migration -database "postgresql://root:secret@localhost:5432/todos?sslmode=disable" -verbose down

migratedown1:
	migrate -path db/migration -database "postgresql://root:secret@localhost:5432/todos?sslmode=disable" -verbose down 1

clearlocalteststorage:
	rm -rf ./uploads/

sqlc:
	sqlc generate

test:
	go test -v -cover ./...

server:
	go run main.go

mocksql:
	mockgen -package mockdb -destination db/mock/store.go github.com/jaingounchained/todo/db/sqlc Store

mockstorage:
	mockgen -package mockStorage -destination storage/mock/storage.go github.com/jaingounchained/todo/storage Storage

.PHONY: postgres createdb dropdb migrateup migrateup1 migratedown migratedown1 sqlc server mocksql mockstorage clearlocalteststorage
