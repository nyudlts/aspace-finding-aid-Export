hello:
	echo "Hello"

build:
	go build -o aspace-export
	zip aspace-export-linux-v$(version).zip aspace-export go-aspace.yml
	mv aspace*.zip bin/linux
