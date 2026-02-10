# Makefile for building and running a Go app

# Binary name
BINARY=vac

# Default target: build and run
all: build run

# Build the Go program
build:
	go build -o $(BINARY) app.go

# Run the compiled program
run: build
	./$(BINARY)

# Clean the compiled binary
clean:
	rm -f $(BINARY)

