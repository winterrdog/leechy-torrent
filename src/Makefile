BIN_DIR := bin
SRC_DIR := .
REPORTS_DIR := coverage-reports

build:
	@go build -ldflags "-s -w" -o $(BIN_DIR)/leechy $(SRC_DIR)/main.go

run:
	@go run $(SRC_DIR)/main.go

test:
	@go test -v ./...

clean:
	@rm -rf $(BIN_DIR)/*

clean-coverage:
	@rm -rf $(REPORTS_DIR)/*

coverage:
	@# Create reports directory if it doesn't exist 
	@mkdir -p $(REPORTS_DIR) 

	@# Run tests and generate coverage report
	@go test -coverprofile=$(REPORTS_DIR)/coverage.out ./...
	@go tool cover -html=$(REPORTS_DIR)/coverage.out -o $(REPORTS_DIR)/coverage.html
