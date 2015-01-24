BINARY = ssh-into-mars
KEY = host_key

all: $(BINARY)

$(BINARY): *.go
	go build .

deps:
	go get .

build: $(BINARY)

clean:
	rm $(BINARY)

$(KEY):
	ssh-keygen -f $(KEY) -P ''

run: $(BINARY) $(KEY)
	./$(BINARY) -i $(KEY)
