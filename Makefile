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
	chmod +x aspace-export

clean:
	rm aspace-export
	rm -r aspace-exports*

install:
	sudo cp aspace-export /usr/local/bin/