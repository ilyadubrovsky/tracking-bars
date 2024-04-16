build:
	docker-compose build tracking-bars

run:
	docker-compose up -d

stop:
	docker-compose down

# make N=your_name create-migration
create-migration:
	goose --dir migrations create ${N} sql

migrate-up:
	goose -dir=./migrations postgres "host=localhost port=5432 user=postgres password=postgres dbname=tracking-bars" up

migrate-down:
	goose -dir=./migrations postgres "host=localhost port=5432 user=postgres password=postgres dbname=tracking-bars" down