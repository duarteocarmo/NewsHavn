build: 
	go build -o bin/app cmd/server/main.go

run: build
	./bin/app

test: 
	go test -v ./... -count=1 

c: 
	gofmt -s -w .

clean-db: 
	rm mydatabase.db
	sqlite3 mydatabase.db < sql/init_db.sql
