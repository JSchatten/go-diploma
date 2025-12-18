build_server_out_folder = ./build/server_out

bs_out = $(build_server_out_folder)/server

dsn_db = postgres://postgres:admin54321@localhost:5678/postgres

src_server = ./cmd/gophermart/main.go


build_server:
	rm -rf $(build_server_out_folder)
	mkdir -p $(build_server_out_folder)
	go build -o $(bs_out) $(src_server)

run_server:
	go run cmd/gophermart/main.go

test_by_bin: build_server
# 	rm -rf ./messages.log
	./gophermarttest -test.v -test.run=^TestGophermart -gophermart-binary-path=$(bs_out) -gophermart-database-uri=$(dsn_db)  -gophermart-host=localhost -gophermart-port=8080 -accrual-binary-path=cmd/accrual/accrual_linux_amd64 -accrual-database-uri=$(dsn_db) -accrual-host=localhost -accrual-port=8081

test_local:
	go test ./...

test_coverage:
	go test ./... -coverprofile=c.out
	go tool cover -func=c.out
	go tool cover -html=c.out -o=./coverage.html
	go test ./... -coverprofile=c.out -race