# Circus thing

## About

More to come.

## Requirements

* go 1.10 or newer
* [jaytaylor.com/archive.is](https://jaytaylor.com/archive.is)

`go get github.com/jaytaylor/archive.is/...`

* [jaytaylor.com/hn-utils](jaytaylor.com/hn-utils)

`go get github.com/jaytaylor/hn-utils/...`

* pdf2htmlEX for PDF -> HTML5 conversion

```bash
git clone git://github.com/coolwanglu/pdf2htmlEX.git
cd pdf2htmlEX
cmake . && make && make install
```

Ubuntu/Debian:

* build-essential
* python-dev
* virtualenv

## TODOs

- [ ] Write system service to scrape news.ycombinator.com/newest and submit all links to archive.is

- [ ] Determine cause of `ERROR 2019/01/16 20:25:43 Error: no lexer for alias '. python' found` -> `grep '\. python' -r $(find . -maxdepth 1 -type d | grep -v '^\.$\|venv')`

