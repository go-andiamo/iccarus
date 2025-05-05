# Iccarus
[![GoDoc](https://godoc.org/github.com/go-andiamo/iccarus?status.svg)](https://pkg.go.dev/github.com/go-andiamo/iccarus)
[![Latest Version](https://img.shields.io/github/v/tag/go-andiamo/iccarus.svg?sort=semver&style=flat&label=version&color=blue)](https://github.com/go-andiamo/iccarus/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/go-andiamo/iccarus)](https://goreportcard.com/report/github.com/go-andiamo/iccarus)

Iccarus is a native Go library for parsing ICC Color Profiles

---

## Features

* Direct parsing of ICC profile files (v2 & v4) with no external dependencies
* Flexible parsing
  * Full, Header only or Header & Tag Table
  * Lazy decoding of tags
  * Extensible tag decoders
* Extract (parse) ICC profiles from images (`.jpeg`,`.png`, `.tif` & `.webp`)
* Color space conversions (experimental)

---

## Installation

```bash
go get github.com/go-andiamo/iccarus
```

---

## Example

```go
package main

import (
    "fmt"
    "github.com/go-andiamo/iccarus"
    "log"
    "os"
)

func main() {
    f, err := os.Open("path/to/profile.icc")
    if err != nil {
        log.Fatal(err)
    }
    defer f.Close()

    profile, err := iccarus.ParseProfile(f, nil)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Color space: %s\n", profile.Header.ColorSpace)
    fmt.Printf("    Version: %s\n", profile.Header.Version)
    if cprt, err := profile.TagValue(iccarus.TagHeaderCopyright); err == nil {
        fmt.Printf("  Copyright: %s\n", cprt)
    }
}
```
