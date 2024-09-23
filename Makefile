# Define the output directory for binaries
OUTPUT_DIR = bin

# Define package name (this should be the name of your Go project)
PACKAGE_NAME = wav-track-extract

# Platforms to build for (add/remove as needed)
PLATFORMS = \
    windows/amd64 \
    linux/amd64 \
    darwin/amd64 \
    darwin/arm64

# Default target when running `make`
.PHONY: all
all: clean build

# Build for each platform
.PHONY: build
build: $(PLATFORMS)
$(PLATFORMS):
	@GOOS=$(word 1, $(subst /, ,$@)) \
	 GOARCH=$(word 2, $(subst /, ,$@)) \
	 go build -o $(OUTPUT_DIR)/$(word 1, $(subst /, ,$@))/$(word 2, $(subst /, ,$@))/$(PACKAGE_NAME)$(if $(filter windows,$(word 1, $(subst /, ,$@))),.exe) .

# Clean the output directory
.PHONY: clean
clean:
	@rm -rf $(OUTPUT_DIR)
	@mkdir -p $(OUTPUT_DIR)/windows/amd64 $(OUTPUT_DIR)/linux/amd64 $(OUTPUT_DIR)/darwin/amd64 $(OUTPUT_DIR)/darwin/arm64