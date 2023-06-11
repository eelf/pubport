pubport.pb.go pubport_grpc.pb.go: pubport.proto
	protoc --go-grpc_out=paths=source_relative:. --go_out=paths=source_relative:. -I. pubport.proto

