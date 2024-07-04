# leechy-torrent

a bittorrent cli client that implements Bittorrent v1.0 that only leeches( _downloads_ ) rather than seeding as well. Written in Go.

## Pre-requisites

- Go `1.16` or later in case you want to build from source in the `src` directory. Otherwise, you can download the pre-built binaries from the releases page.
- Supported OS: _Linux, MacOS, Windows_

## Installation

- You can download the pre-built binaries from the releases page.
- Or you can build from source( _if you have trust issues_ ) by running the following commands:

  ```bash
  cd src
  make build
  ```

  This will generate a binary in the `bin` directory called `leechy`.

## Usage

- To download a file over Bittorrent, you can run the following command:

  ```bash
  ./leechy <torrent-file> <output-file>
  ```

  This will start downloading the torrent file in the current directory. You will see a stream of logs indicating the progress of the download.

## Action!

[![asciicast](https://asciinema.org/a/666794.svg)](https://asciinema.org/a/666794)

## Testing

- To run the tests, you can run the following command:

  ```bash
  make test
  ```

- To get test coverage reports, you can run the following command:

  ```bash
  make coverage
  ```

  This will generate a coverage report in the `coverage-reports` directory.

- In case you wanna clean up the coverage reports, you can run the following command:

  ```bash
  make clean-coverage
  ```

## Features

- [x] Downloading single file torrents.
- [x] Bittorrent v1.0 support.
- [x] Supports leeching.
- [x] Fast and efficient with downloading. It pipelines 8 requests at a time while using `go`routines for parallelism.
- [x] Supports downloading from multiple peers.
- [x] Command line interface.
- [x] HTTP tracker support.
- [ ] UDP tracker support.
- [ ] Multi-file torrent support.
- [ ] Seeding support.
- [ ] Magnet link support.
- [ ] DHT support.
- [ ] Bittorrent v2.0 support.

But we are working on adding the missing features in the future. _They're sort of todo items._

## References

- [Bittorrent v1.0 Spec](https://www.bittorrent.org/beps/bep_0003.html)
- [Unofficial Wiki on Bittorrent](https://wiki.theory.org/BitTorrentSpecification)

## License

[MIT](LICENSE)
