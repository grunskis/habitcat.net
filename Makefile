.PHONY: build deploy install-server

build: activities

activities: main.go
	env GOOS=linux GOARCH=arm go build

deploy: activities
	ssh rpi mkdir -p activities
	ssh rpi pkill activities || true
	scp -r activities static templates sql_migrations rpi:activities/
	ssh rpi "cd activities && dtach -n /tmp/activities.socket ./activities >> /tmp/activities.log 2>&1"

install-server:
	sudo apt-get install -y postgresql-9.4 postgresql-contrib-9.4
	createdb -U postgres activities
	psql -U postgres activities < create_all.sql
	psql -U postgres activities < load_data.sql
