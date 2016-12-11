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

func ExampleDecodeString() {

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

	fmt.Printf("content: \n%s", w.String())
}

func ExampleEncodeToString() {

	// Setup the struct and reader...
	v := struct{ Name string }{Name: "A EncodeToString Example"}
	src := []byte("Content...")

	// Do the decoding...
	content := JSONEncoding.EncodeToString(src, &v)

	fmt.Printf("content: \n%s", content)

}
