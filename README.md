# Particle 

Particle is a go library that provides an Encoder/Decoder and Marshaler/Unmarshaler for frontmatter files.

A frontmatter file contains a block of metadata at the beginning of the file. This block is usually delimited by a `---` or `+++`, as defined by [Jekyll](http://jekyllrb.com/docs/frontmatter/), however JSON may be included without a specific delimiter.

The Particle library can decode and encode frontmatter metadata blocks that are YAML, TOML or JSON. It is also a generic library so it's easy to define new custom block types to decode and encode as well.

## Installation

    go get github.com/njones/particle

## Example of how to pull data from a frontmatter data string

```go
package main

import (
  "fmt"

  "github.com/njones/particle"
)

var jsonFrontmatterFile = `
{
  "title":"front-matter example"
}

This is some example text
`

func main() {
  
  metadata := struct {
    Title string
  }{}

  body, err := particle.JSONEncoding.DecodeString(jsonFrontmatterFile, &metadata)
  if err != nil {
    // handle err
  }

  fmt.Println("Frontmatter metadata: %#v", metadata)
  fmt.Println("File body: %s", string(body))

}

```

## Example of creating a frontmatter data string

```go
package main

import (
  "fmt"

  "github.com/njones/particle"
)

func main() {
  
  metadata := struct {
    Title string
  }{
    Title: "front-matter example"
  }

  body := particle.JSONEncoding.EncodeToString([]byte("This is some example text\n"), &metadata)

  fmt.Println("File body: %s", body)

}

```

This package depends on the following external encoding/decoding libraries:

- http://gopkg.in/yaml.v2
- https://github.com/BurntSushi/toml

Note that the standard marshaling/unmarshaling function signatures can be used i.e `func(interface{}) ([]byte error)` and `func([]byte, interface{}) error` for the encoding and decoding of the frontmatter metadata.

# License

Particle is available under the [MIT License](https://opensource.org/licenses/MIT).

Copyright (c) 2016 Nika Jones <copyright@nikajon.es> All Rights Reserved.
