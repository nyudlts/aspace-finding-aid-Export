hello:
	echo "Hello"

package:
	go mod tidy
	go build -o aspace-export
	zip aspace-export-$(OS)-v$(VERSION).zip aspace-export go-aspace.yml README.md
	mv aspace*.zip bin/$(OS)

build:
	go mod tidy
	go build -o aspace-export