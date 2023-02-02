build:
	docker-compose build app

run:
	docker-compose up -d

stop:
	docker-compose down

migrate-up:
	migrate -path ./migrations -database "postgres://postgres:12345678@localhost:5433/trackingbars?sslmode=disable" up

migrate-down:
	migrate -path ./migrations -database "postgres://postgres:12345678@localhost:5433/trackingbars?sslmode=disable" down