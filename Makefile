.PHONY: build deploy install-server test test-coverage-html clean

build: activities

activities: main.go habits.go goals.go
	go build

deploy: clean activities
	git push heroku $$(git rev-parse --abbrev-ref HEAD):master


TEST_DBNAME=gandhi_test

test:
	dropdb --if-exists $(TEST_DBNAME) && createdb $(TEST_DBNAME)
	psql -c 'CREATE EXTENSION IF NOT EXISTS "uuid-ossp"' $(TEST_DBNAME)
	find sql_migrations -name '*.sql' | sort | xargs cat | psql $(TEST_DBNAME) -f -
	go test -v -coverprofile=coverage.out

test-coverage-html: coverage.out
	go tool cover -html=coverage.out

coverage.out: test

clean:
	rm -f activities
