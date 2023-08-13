.SILENT:

service-a:
	go run . -service a

service-b:
	go run . -service b

req:
	curl -k http://localhost:8081/serviceA
