hello:
	echo "Hello"

build:
	go build -o aspace-export
	zip aspace-export-$(OS)-v$(VERSION).zip aspace-export go-aspace.yml README.md
	mv aspace*.zip bin/$(OS)