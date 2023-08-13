.SILENT:
.PHONY: service-a service-b req docker-compose clean

service-a:
	go run . -service a

service-b:
	go run . -service b

req:
	curl -k http://localhost:8081/serviceA

docker-compose:
	docker-compose up --abort-on-container-exit

clean:
	docker-compose down -v
