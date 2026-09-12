-include .env
export

create_migration:
	migrate create -ext sql -dir ./migrations $(NAME)

migrate_up:
	migrate -path ./migrations -database "$(DB_URL)" up

migrate_down:
	migrate -path ./migrations -database "$(DB_URL)" down

migrate_force:
	migrate -path ./migrations -database "$(DB_URL)" force $(VERSION)