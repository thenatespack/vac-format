# Makefile for vac-format project

# Name of the output executable
BINARY := bin/vac-format

# Main package location
MAIN := ./cmd/vac-format

# Default target: build the executable
all: $(BINARY)

# Build the executable
$(BINARY):
	@mkdir -p bin
	go build -o $(BINARY) $(MAIN)

# Run the program without building manually
run: $(BINARY)
	$(BINARY) $(ARGS)

# Clean the build
clean:
	rm -f $(BINARY)

.PHONY: all run clean

