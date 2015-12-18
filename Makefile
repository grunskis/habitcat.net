.PHONY: build deploy install-server

build: activities

activities: main.go
	env GOOS=linux GOARCH=arm go build

deploy: activities
	ssh rpi mkdir -p activities
	scp -r activities static rpi:activities/

install-server:
	sudo apt-get install -y postgresql-9.4 postgresql-contrib-9.4
	createdb -U postgres activities
	psql -U postgres activities < create_all.sql
	psql -U postgres activities < load_data.sql
