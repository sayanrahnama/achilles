.PHONY: proto clean

PROTO_DIR = schema/proto
SVC_DIR = internal/svc
SERVICES = auth user wallet transaction

proto: $(SERVICES)

$(SERVICES): 
	@echo "Generating proto files for $@ service..."
	@mkdir -p $(SVC_DIR)/$@/pb
	protoc --proto_path=$(PROTO_DIR) \
		--go_out=$(SVC_DIR)/$@/pb --go_opt=paths=source_relative \
		--go-grpc_out=$(SVC_DIR)/$@/pb --go-grpc_opt=paths=source_relative \
		$(PROTO_DIR)/$@/$@.proto
	@echo "Proto files for $@ service generated successfully!"

clean:
	@echo "Cleaning generated proto files..."
	@for svc in $(SERVICES); do \
		rm -rf $(SVC_DIR)/$$svc/pb/*.go; \
	done
	@echo "Proto files cleaned successfully!"

help:
	@echo "Available commands:"
	@echo "  make proto     - Generate proto files for all services"
	@echo "  make auth      - Generate proto files for auth service only"
	@echo "  make user      - Generate proto files for user service only"
	@echo "  make wallet    - Generate proto files for wallet service only"
	@echo "  make transaction - Generate proto files for transaction service only"
	@echo "  make clean     - Remove all generated proto files"
	@echo "  make help      - Show this help message"