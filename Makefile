BINARY = ssh-into-mars
SSHKEY = dist/ssh-host-key
TLSKEY = dist/tls-host-key

all: $(BINARY)

$(BINARY): *.go
	go build .

deps:
	go get .

build: $(BINARY)

clean:
	rm $(BINARY)

$(SSHKEY):
	ssh-keygen -f $(SSHKEY) -P ''

$(TLSKEY):
	openssl req -x509 -newkey rsa:2048 -nodes -subj /CN=localhost -days 365 -keyout $(TLSKEY) -out $(TLSKEY).pub

run: $(BINARY) $(SSHKEY) $(TLSKEY)
	./$(BINARY) --ssh-key $(SSHKEY) --tls-cert $(TLSKEY).pub --tls-key $(TLSKEY)

dist/ssh-into-mars.tar.gz: $(BINARY) assets
	tar -zcf dist/ssh-into-mars.tar.gz $(BINARY) assets

dist/ssh-into-mars-1-1-any.pkg.tar.xz: dist/PKGBUILD dist/ssh-into-mars.tar.gz
	cd dist && makepkg -fc

package: dist/ssh-into-mars-1-1-any.pkg.tar.xz

