# snappy command line tool

### How to get

```
go get -v github.com/ma6174/snappy
```

### How to use

compress

```
snappy <input files>
cat <input file> | snappy > <output file.snappy>
```

decompress

```
snappy -d <input files.snappy>
cat <input file.snappy> | snappy -d > <output file>
```

* use `-s` to change default suffix `.snappy`
* use `-v` to show percentage reduction and speed
* use `-c` to let output to stdout
* use `-h` to show help
