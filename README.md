# My Favorite Place on Mars

An experiment using SSH to sign in to websites.
[Explanatory blog post](https://vtllf.org/blog/ssh-web-sign-in).
[Live demo](https://mars.vtllf.org/).

[![](screenshot_cropped.png)](https://mars.vtllf.org/)


## How it works

A custom SSH server is started alongside a web server. Unlike normal SSH
servers this one accepts any key presented to it and creates a new
account on the web site. A secret, temporary link to the site is printed
into the user's terminal and the server closes the connection. No shell
access is granted.

When the user opens that link in a browser they are signed in and
associated with their public key. A session cookie is set and they can
use the site as normal.

No passwords are stored by the site, only their public key. Users can
choose to share the same key across many websites or instead make
site-specific keys. Temporary keys can be made for throwaway accounts. 

If the same key is shared across several sites, and those sites publish
their users' public keys (like GitHub and this demo both do), those
accounts can be linked back to the same person.


## Developing

    go get github.com/duncankl/ssh-mars
    cd $GOPATH/src/github.com/duncankl/ssh-mars
    make keygen
    make run

The server should be available at:
[https://localhost:3000/](https://localhost:3000/). It uses a
self-signed TLS certificate by default, so you'll have to add an
exception to your browser.


## Security

This demo is new, unreviewed and untested. Don't use it for anything
that handles sensitive data. If you are interested, please do pull apart
the code and report back vulnerabilities, I'd be very grateful.


## Acknowledgements

Thanks to [Andrey Petrov](http://shazow.net/) for showing how the Go ssh
package can be (ab)used to make these kinds of experiments.


## License

GPLv3

