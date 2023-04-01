build:
	docker-compose build telegram-service
	docker-compose build user-service
	docker-compose build grades-service

run:
	docker-compose up -d

stop:
	docker-compose down

migrate-up:
	migrate -path ./migrations -database "postgres://postgres:12345678@localhost:5433/trackingbars?sslmode=disable" up

migrate-down:
	migrate -path ./migrations -database "postgres://postgres:12345678@localhost:5433/trackingbars?sslmode=disable" down
