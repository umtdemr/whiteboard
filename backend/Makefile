DB_NAME=wb
DB_USER=wb
DB_PASSWORD=wb
DB_URL=postgresql://${DB_USER}:${DB_PASSWORD}@localhost:5432/${DB_NAME}?sslmode=disable

createdb: createuser
	docker exec postgressi psql -U root -c "CREATE DATABASE ${DB_NAME};"
	docker exec postgressi psql -U root -c "GRANT ALL PRIVILEGES ON DATABASE ${DB_NAME} TO ${DB_USER};"
	docker exec postgressi psql -U root -c "GRANT ALL ON SCHEMA public TO ${DB_USER};"
createuser:
	docker exec postgressi psql -U root -c "CREATE USER ${DB_USER} WITH SUPERUSER PASSWORD '${DB_PASSWORD}';"
	docker exec postgressi psql -U root -c "ALTER ROLE ${DB_USER} SET client_encoding TO 'utf8';"
	docker exec postgressi psql -U root -c "ALTER ROLE ${DB_USER} SET timezone TO 'UTC';"
create_migration:
	goose -dir=./internal/db/migrations postgres "${DB_URL}" create $(name) sql
migrate_up:
	goose -dir=./internal/db/migrations postgres "${DB_URL}" up
migrate_down:
	goose -dir=./internal/db/migrations postgres "${DB_URL}" down
mock:
	mockgen -package mockdb -destination internal/db/mock/store.go github.com/umtdemr/wb-backend/internal/db/sqlc Store
	mockgen -package mockdata -destination internal/data/mock/user.go github.com/umtdemr/wb-backend/internal/data UserModel
	mockgen -package mockdata -destination internal/data/mock/board.go github.com/umtdemr/wb-backend/internal/data BoardModel
	mockgen -package mockdata -destination internal/data/mock/permissions.go github.com/umtdemr/wb-backend/internal/data PermissionModel
	mockgen -package mockdata -destination internal/data/mock/tokens.go github.com/umtdemr/wb-backend/internal/data TokenModel
	mockgen -package mockworker -destination internal/worker/mock/publisher.go github.com/umtdemr/wb-backend/internal/worker Publisher 

.PHONY: createdb createuser create_migration migrate_up migrate_down mock