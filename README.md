# fmajor

`fmajor` is a self-hosted file upload service. It is very basic, but
easy to install.

![Screenshot of fmajor running in Firefox](doc/screenshot.png)

## Features

* Upload, download and delete files all from the web interface.

* `fmajor` is compiled to one static binary, which includes all
  resources. This makes deployment very easy, no need for containers
  or virtual machines.

* Doesn't require a database. All data is stored on the file system.

## What it Doesn't Do

* Authentication and Authorization; there isn't any. You'll have
  to use a proxy like `nginx` with basic authentication if you don't
  want random people on the internet uploading files on your server.

## Install

The following instructions were tested on Debian 10, but installation
on any `systemd` based system should be about the same. If you aren't
using `systemd`, you probably know what to do differently.

(TODO)

## Credit

(c) 2020 Andreas Schärtl

This program (fmajor) is free software: you can redistribute it and/or
modify it under the terms of the GNU General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version. For a copy of this
license, see `LICENSE`.

Above applies only to the source code. Excluded are included icons
icons from the [Feather](https://feathericons.com/) icon set licensed
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

	/static/trash-2.svg /static/paperclip.svg
