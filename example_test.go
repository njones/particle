package particle

import (
	"bytes"
	"fmt"
	"strings"
)

func ExampleNewDecoder() {

	// Setup the struct and reader...
	v := struct{ Name string }{}
	r := strings.NewReader(`---
name: A NewDecoder Example
---

Content...`)

	// Do the decoding...
	output, err := NewDecoder(YAMLEncoding, r, &v)
	if err != nil {
		// handle errors here
		fmt.Println(err)
	}

	// Read the content to a buffer so we can see it...
	content := new(bytes.Buffer)
	content.ReadFrom(output)

	fmt.Printf("yaml: %+v\ncontent: %s", v, content)

	// Output:
	// yaml: {Name:A NewDecoder Example}
	// content: Content...
}

func ExampleEncoding_DecodeString() {

	// Setup the struct and reader...
	v := struct{ Name string }{}
	src := `+++
name = "A DecodeString Example"
+++

Content...`

	// Do the decoding...
	b, err := TOMLEncoding.DecodeString(src, &v)
	if err != nil {
		// handle errors here
		fmt.Println(err)
	}

	// Read the content to a buffer so we can see it...
	content := string(b)

	fmt.Printf("toml: %+v\ncontent: %s", v, content)

	// Output:
	// toml: {Name:A DecodeString Example}
	// content: Content...
}

func ExampleNewEncoder() {

	// Setup the struct and reader...
	v := struct{ Name string }{Name: "A NewEncoder Example"}
	w := new(bytes.Buffer)

	// Do the encoding...
	out, err := NewEncoder(YAMLEncoding, w, v)
	if err != nil {
		// handle errors here
		fmt.Println(err)
	}

	// Writing to the writer that we got back from the NewEncoder function
	out.Write([]byte("Content..."))

	// view the raw bytes
	fmt.Printf("content: % x", w.String())
	// output: content: 2d 2d 2d 0a 6e 61 6d 65 3a 20 41 20 4e 65 77 45 6e 63 6f 64 65 72 20 45 78 61 6d 70 6c 65 0a 2d 2d 2d 0a 0a 43 6f 6e 74 65 6e 74 2e 2e 2e
}

func ExampleEncoding_EncodeToString() {

	// Setup the struct and reader...
	v := struct{ Name string }{Name: "A EncodeToString Example"}
	src := []byte("Content...")

	// Do the decoding...
	content := JSONEncoding.EncodeToString(src, &v)

	// view the raw bytes
	fmt.Printf("content: % x", content)
	// output: content: 7b 0a 09 22 4e 61 6d 65 22 3a 20 22 41 20 45 6e 63 6f 64 65 54 6f 53 74 72 69 6e 67 20 45 78 61 6d 70 6c 65 22 0a 7d 0a 0a 43 6f 6e 74 65 6e 74 2e 2e 2e

}
