# fmajor

`fmajor` is a self-hosted file upload service. It is basic, but easy
to install.

![Screenshot of fmajor running in Firefox](doc/screenshot.png)

## Features

* Upload, download and delete files all from the web interface.  Only
  users with a password can upload and delete file but *everyone can
  download all uploaded files assuming they have the link*.

* `fmajor` is compiled to one static binary, which includes all
  resources. This makes deployment easy, no need for containers or
  virtual machines.

* Doesn't require a database. All data is stored on the file system.

## What it Doesn't Do

* `fmajor` does not include transport encryption (i.e. HTTPS). Please
  use a proxy like `nginx` with TLS enabled to ensure that nobody
  listens to your login password.

## Building

Assuming you have `go` version 1.17 or later, you should only have to run

	$ go install github.com/kissen/fmajor@latest

You should now have `fmajor` available on your system.

## Setup With `systemd`

The following instructions were tested on Debian 10 and assume that
you know how to proxy and secure HTTP services with something like
`nginx`.

1. Build `fmajor` like explained in the previous section. You can do
   this on your workstation.

2. Move the `fmajor` binary to a reasonable place on your server.
   This tutorial assumes that place to be `/usr/bin/fmajor`. All following
   commands need to be run on your server as `root`.

3. Create a dedicated user for running `fmajor`.

		# adduser --home /var/lib/fmajor --shell /sbin/nologin --disabled-password fmajor

   This creates a user named `fmajor` with home directory `/var/lib/fmajor`
   which is where we will let `fmajor` put all its uploads.

4. Copy the configuration file to `/etc`.

		# wget https://raw.githubusercontent.com/kissen/fmajor/master/doc/fmajor.conf
		# mv fmajor.conf /etc/fmajor.conf

   You should now have a configuration file `/etc/fmajor.conf`.

5. You need to set up at least one password. Without a password,
   you will not be able to log in and therefore upload files.

   The easiest way is to use the `htpasswd` tool to generate the
   hash. On Debian, you can get `htpasswd` with the `apache2-utils`
   package. With it installed, run

		$ htpasswd -n -B -C 12 "" | tr -d ':\n'

   and you will be prompted for the password. The hash is printed to
   `stdout`.

   Open `/etc/fmajor.conf` with a text editor and edit section
   `PassHashes` accordingly. It should look something like

		PassHashes = [
			"$2y$12$uTLL4JVVyJg9aunt.hyraej3m0yW6siY2cAQ1MakmUxtxgR4EoPbK"
		]

6. Install the `systemd` service file.

		# wget https://raw.githubusercontent.com/kissen/fmajor/master/doc/fmajor.service
		# mv fmajor.service /lib/systemd/system/fmajor.service
		# systemctl daemon-reload

7. You can now start the `fmajor` service with

		# systemctl start fmajor

   which will make `systemd` take care of keeping your logs. Access
   the logs using

		# journalctl -u fmajor.service

   If you want `fmajor` to start during boot, enable the service with
   `systemctl` like so

		# systemctl enable fmajor


8. `fmajor` listens to the `ListenAddress` defined in configuration
   file `/etc/fmajor.conf`. Per default, this is `localhost` which of
   course isn't very useful if you want to access the service
   remotely. To make `fmajor` accessible on the open internet,
   configure your reverse proxy (e.g. `nginx`) to forward requests to
   `ListenAddress`.

   You should configure your reverse proxy to use HTTPS, otherwise
   third parties will be able to spy on your interactions with
   `fmajor`.  [Let's Encrypt](https://letsencrypt.org/) with
   [Certbot](https://certbot.eff.org/) is the canonical choice.

## Credit

(c) 2020 - 2021 Andreas Schärtl

This program (`fmajor`) is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by the Free
Software Foundation, either version 3 of the License, or (at your option) any
later version. For a copy of this license, see `LICENSE`.

### Feather Icons

Above applies only to the source code. Excluded are included icons
from the [Feather](https://feathericons.com/) icon set licensed
under the following terms.

	The MIT License (MIT)

	Copyright (c) 2013-2017 Cole Bemis

	Permission is hereby granted, free of charge, to any person obtaining a copy
	of this software and associated documentation files (the "Software"), to deal
	in the Software without restriction, including without limitation the rights
	to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
	copies of the Software, and to permit persons to whom the Software is
	furnished to do so, subject to the following conditions:

	The above copyright notice and this permission notice shall be included in all
	copies or substantial portions of the Software.

	THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
	IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
	FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
	AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
	LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
	OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
	SOFTWARE.

The affected files are

	/static/trash-2.svg /static/paperclip.svg /static/upload-cloud.svg
	/static/log-out.svg

### Fonts

This repository also contains a copy of the [Quicksand](https://github.com/andrew-paglinawan/QuicksandFamily)
font.

	Copyright 2011 The Quicksand Project Authors (https://github.com/andrew-paglinawan/QuicksandFamily),
	with Reserved Font Name “Quicksand”.

	This Font Software is licensed under the SIL Open Font License, Version 1.1.
	This license is copied below, and is also available with a FAQ at:
	http://scripts.sil.org/OFL

For a full text of the license, please take a look at the
[Quicksand repository](https://github.com/andrew-paglinawan/QuicksandFamily/blob/master/OFL.txt).
The affected file is

	/static/quicksand.ttf
