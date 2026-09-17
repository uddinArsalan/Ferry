-include .env
export

run:
	go run ./cmd/main.go

create_migration:
	migrate create -ext sql -dir ./migrations $(NAME)

migrate_up:
	migrate -path ./migrations -database "$(DB_URL)" up

migrate_down:
	migrate -path ./migrations -database "$(DB_URL)" down

migrate_force:
	migrate -path ./migrations -database "$(DB_URL)" force $(VERSION)

migrate_version:
	migrate -path ./migrations -database "$(DB_URL)" version

protoc:
	protoc --go_out=. --go_opt=paths=source_relative \
    	--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/$(FILE_PATH)