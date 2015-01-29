proj = ssh-mars
binary = $(proj)
rundir = run
sshkey = $(rundir)/ssh-identity
tlskey = $(rundir)/tls-identity
db = $(rundir)/mars.sqlite
pkg = $(proj)-1-1-any.pkg.tar.xz
user = root
host = mars.vtllf.org
serviceport = 9291

build: *.go
	go build .

deps:
	go get .

clean:
	rm $(binary)

keygen:
	mkdir -p $(rundir)
	ssh-keygen -f $(sshkey) -P ''
	openssl req -x509 -newkey rsa:2048 -nodes -subj /CN=localhost -days 365 -keyout $(tlskey).key -out $(tlskey).crt

run: build
	mkdir -p $(rundir)
	./$(binary) --ssh-key $(sshkey) --tls-cert $(tlskey).crt --tls-key $(tlskey).key --db $(db)

archive: build
	tar -zcf dist/$(proj).tar.gz $(binary) assets

package: archive dist/PKGBUILD
	cd dist && makepkg --skipinteg -fc

deploy: package
	rsync -e "ssh -p $(serviceport)" dist/$(pkg) $(user)@$(host):/var/cache/pacman/pkg/
	ssh -p $(serviceport) $(user)@$(host) \
		"pacman --noconfirm --force -U /var/cache/pacman/pkg/$(pkg); \
		systemctl restart $(proj).service;"

pull:
	rsync -e "ssh -p $(serviceport)" $(user)@$(host):/srv/$(proj)/mars.sqlite dist/

.PHONY: build deps clean keygen run archive deploy pull
