// Copyright 2016 Nika Jones. All rights reserved.
// Use of this source code is governed by the MIT license.
// license that can be found in the LICENSE file.

// Package particle implements frontmatter encoding as specified by
// the Jekyll specification.
package particle

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"sync"

	"encoding/json"
	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v2"
)

const (
	YAMLDelimiter     = "---"
	TOMLDelimiter     = "+++"
	JSONDelimiterPair = "{ }"
)

var (
	// YAMLEncoding is the encoding for standard frontmatter files
	// that use YAML as the metadata format.
	YAMLEncoding = NewEncoding(WithDelimiter(YAMLDelimiter), WithMarshalFunc(yaml.Marshal), WithUnmarshalFunc(yaml.Unmarshal))

	// TOMLEncoding is the encoding for frontmatter files that use
	// TOML as the metadata format.
	TOMLEncoding = NewEncoding(WithDelimiter(TOMLDelimiter), WithMarshalFunc(tomlMarshal), WithUnmarshalFunc(toml.Unmarshal))

	// JSONEncoding is the encoding for frontmatter files that use
	// JSON as the metadata format, note there is no delimiter, just
	// use a single open and close curly bracket on a line to
	// designate the JSON frontmatter metadata block.
	JSONEncoding = NewEncoding(
		WithDelimiter(JSONDelimiterPair),
		WithMarshalFunc(jsonMarshal),
		WithUnmarshalFunc(json.Unmarshal),
		WithSplitFunc(SpaceSeparatedTokenDelimiters),
		WithIncludeDelimiter(),
	)
)

// The SplitFunc type returns the open and close delimiters, along
// with a bufio.SplitFunc that will be used to parse the frontmatter
// file.
type SplitFunc func(string) (string, string, bufio.SplitFunc)

// The MarshalFunc type is the standard unmarshal function that maps a
// struct or map to frontmatter encoded byte string.
type MarshalFunc func(interface{}) ([]byte, error)

// The UnmarshalFunc type is the standard marshal function that maps
// frontmatter encoded metadata to a struct or map.
type UnmarshalFunc func([]byte, interface{}) error

// The EncodingOptionFunc type the function signature for adding encoding
// options to the formatter.
type EncodingOptionFunc func(*Encoding) error

// The encoder type is a writer that will add the frontmatter encoded metadata
// before the source data stream is written to the underlying writer type
// encoder struct{ w io.Writer }
type encoder struct{ w io.Writer }

func (l *encoder) Write(p []byte) (n int, err error) {
	n, err = l.w.Write(p)
	return
}

// WithDelimiter adds the string delimiter to designate frontmatter encoded
// metadata section to *Encoding
func WithDelimiter(s string) EncodingOptionFunc {
	return func(e *Encoding) error {
		e.delimiter = s
		return nil
	}
}

// WithMarshalFunc adds the MarshalFunc function that will marshal a struct or
// map to frontmatter encoded metadata string *Encoding
func WithMarshalFunc(fn MarshalFunc) EncodingOptionFunc {
	return func(e *Encoding) error {
		e.marshalFunc = fn
		return nil
	}
}

// WithUnmarshalFunc adds the UnmarshalFunc function that will unmarshal the
// frontmatter encoded metadata to a struct or map to *Encoding
func WithUnmarshalFunc(fn UnmarshalFunc) EncodingOptionFunc {
	return func(e *Encoding) error {
		e.unmarshalFunc = fn
		return nil
	}
}

// WithSplitFunc adds the SplitFunc function to *Encoding
func WithSplitFunc(fn SplitFunc) EncodingOptionFunc {
	return func(e *Encoding) error {
		e.inSplitFunc = fn
		return nil
	}
}

// WithIncludeDelimiter is a bool that includes the delimiter in the
// frontmatter metadata for *Encoding
func WithIncludeDelimiter() EncodingOptionFunc {
	return func(e *Encoding) error {
		e.outputDelimiter = true
		return nil
	}
}

// NewDecoder constructs a new frontmatter stream decoder, adding the
// marshaled frontmatter metadata to interface v.
func NewDecoder(e *Encoding, r io.Reader, v interface{}) (io.Reader, error) {
	m, o := e.readFrom(r)
	if err := e.readUnmarshal(m, v); err != nil {
		return nil, err
	}

	return o, nil
}

// NewEncoder returns a new frontmatter stream encoder. Data written to the
// returned writer will be prefixed with the encoded frontmatter metadata
// using e and then written to w.
func NewEncoder(e *Encoding, w io.Writer, v interface{}) (io.Writer, error) {
	o := &encoder{w: w}

	f, err := e.encodeFrontmatter(v)
	if err != nil {
		return nil, err
	}
	o.Write(f) // write frontmatter first

	return o, nil
}

// Encoding is the set of options that determine the marshaling and
// unmarshaling encoding specifications of frontmatter metadata.
type Encoding struct {
	output                struct{ start, end string }
	start, end, delimiter string
	outputDelimiter       bool

	inSplitFunc   SplitFunc
	ioSplitFunc   bufio.SplitFunc
	marshalFunc   MarshalFunc
	unmarshalFunc UnmarshalFunc

	fmBufMutex sync.Mutex
	fmBuf      map[string][]byte
}

// NewEncoding returns a new Encoding defined by the any passed in options.
// All options can be changed by passing in the appropriate EncodingOptionFunc
// option.
func NewEncoding(options ...EncodingOptionFunc) *Encoding {
	e := &Encoding{
		outputDelimiter: false,
		inSplitFunc:     SingleTokenDelimiter,
	}
	for _, o := range options {
		if err := o(e); err != nil {
			panic(err)
		}
	}

	e.fmBuf = make(map[string][]byte)
	e.start, e.end, e.ioSplitFunc = e.inSplitFunc(e.delimiter)
	if e.outputDelimiter {
		e.output.start, e.output.end = e.start, e.end
	}
	return e
}

// Decode decodes src using the encoding e. It writes bytes to dst and returns
// the number of bytes written. If src contains invalid unmarshaled data, it
// will return the number of bytes successfully written along with an error.
func (e *Encoding) Decode(dst, src []byte, v interface{}) (int, error) {
	m, r := e.readFrom(bytes.NewBuffer(src))
	if err := e.readUnmarshal(m, v); err != nil {
		return 0, err
	}

	return io.ReadFull(r, dst)
}

// DecodeString returns the bytes representing the string data of src without
// the frontmatter. The interface v will contain the decoded frontmatter
// metadata. It returns an error if the underlining marshaler returns an
// error.
func (e *Encoding) DecodeString(src string, v interface{}) ([]byte, error) {
	return e.DecodeReader(bytes.NewBufferString(src), v)
}

// DecodeReader returns the bytes representing the data collected from reader
// r without frontmatter metadata. The interface v will contain the decoded
// frontmatter metadata.
func (e *Encoding) DecodeReader(r io.Reader, v interface{}) ([]byte, error) {
	m, r := e.readFrom(r)
	if err := e.readUnmarshal(m, v); err != nil {
		return nil, err
	}
	return ioutil.ReadAll(r)
}

// EncodeToString returns the frontmatter encoding of type e Encoding before
// the data bytes of src populated with the data of interface v.
func (e *Encoding) EncodeToString(src []byte, v interface{}) string {
	b := make([]byte, e.EncodeLen(src, v))
	e.Encode(b, src, v)
	return string(b)
}

// Encode encodes src using the encoding e, writing EncodedLen(len(encoded
// frontmatter)+len(src)) bytes to dst.
func (e *Encoding) Encode(dst, src []byte, v interface{}) {
	f, err := e.encodeFrontmatter(v)
	if err != nil {
		panic(err)
	}

	b := new(bytes.Buffer)
	b.Write(f)
	b.Write(src)

	io.ReadFull(b, dst)
}

// EncodedLen returns the length in bytes of the frontmatter encoding of an
// input buffer and frontmatter metadata of interface i of length n.
func (e *Encoding) EncodeLen(src []byte, v interface{}) int {
	f, err := e.encodeFrontmatter(v)
	if err != nil {
		panic(err)
	}
	return len(f) + len(src)
}

// hashFrontmatter returns a very simple hash of the interface v with data.
func (e *Encoding) hashFrontmatter(v interface{}) string {
	h := md5.Sum([]byte(fmt.Sprintf("%#v", v)))
	return string(h[:])
}

// encodeFrontmatter marshals the data from interface v to frontmatter
// metadata. The result is cached, therefore it can be called multiple times
// with little performance hit.
func (e *Encoding) encodeFrontmatter(v interface{}) ([]byte, error) {
	h := e.hashFrontmatter(v)
	if f, ok := e.fmBuf[h]; ok {
		return f, nil
	}

	f, err := e.marshalFunc(v)
	if err != nil {
		return nil, err
	}

	var start, end string
	if !e.outputDelimiter {
		start, end = e.start+"\n", e.end
	}

	e.fmBufMutex.Lock()
	e.fmBuf[h] = append(append([]byte(start), f...), []byte(end+"\n\n")...)
	e.fmBufMutex.Unlock()
	return e.fmBuf[h], nil
}

// readUnmarshal takes the encoded frontmatter metadata from reader r and
// unmarshals the data to interface v.
func (e *Encoding) readUnmarshal(r io.Reader, v interface{}) error {
	f, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}

	if err := e.unmarshalFunc(f, v); err != nil {
		return err
	}
	return nil
}

// readFrom takes the incoming reader stream r and splits it into a reader
// stream for encoded frontmatter metadata and a stream for content.
func (e *Encoding) readFrom(r io.Reader) (frontmatter, content io.Reader) {
	mr, mw := io.Pipe()
	cr, cw := io.Pipe()

	go func() {
		e.start, e.end, e.ioSplitFunc = e.inSplitFunc(e.delimiter) // reset each time it's run

		defer mw.Close() // if the matter writer is never written to...
		defer cw.Close() // if data writer is never written to...

		scnr := bufio.NewScanner(r)
		scnr.Split(e.ioSplitFunc)

		for scnr.Scan() {
			txt := scnr.Text()
			if txt == e.delimiter {
				io.WriteString(mw, e.output.start)
				for scnr.Scan() {
					txt := scnr.Text()
					if txt == e.delimiter {
						io.WriteString(mw, e.output.end)
						break
					}
					io.WriteString(mw, txt)
				}
				mw.Close()
			} else {
				mw.Close()
				io.WriteString(cw, txt)
			}
			for scnr.Scan() {
				txt := scnr.Text()
				io.WriteString(cw, txt)
			}
			cw.Close()
		}
	}()

	return mr, cr
}

// SingleTokenDelimiter returns the start and end delimiter along with the
// bufio SplitFunc that will split out the frontmatter encoded metadata from
// the io.Reader stream.
func SingleTokenDelimiter(delim string) (start string, end string, fn bufio.SplitFunc) {
	return delim, delim, baseSplitter([]byte(delim+"\n"), []byte("\n"+delim+"\n"), []byte(delim))
}

// SpaceSeparatedTokenDelimiters returns the start and end delimiter which is
// split on a space from string delim. The bufio.SplitFunc will split out the
// frontmatter encoded data from the stream.
func SpaceSeparatedTokenDelimiters(delim string) (start string, end string, fn bufio.SplitFunc) {
	delims := strings.Split(delim, " ")
	if len(delims) != 2 {
		panic("The delimiter token does not split into exactly two")
	}
	start, end = delims[0], delims[1]
	return start, end, baseSplitter([]byte(start+"\n"), []byte("\n"+end+"\n"), []byte(delim))
}

// baseSplitter reads the characters of a steam and split returns a token when
// a frontmatter delimiter has been determined.
func baseSplitter(topDelimiter, botDelimiter, retDelimiter []byte) bufio.SplitFunc {
	var (
		firstTime            bool = true
		checkForBotDelimiter bool

		topDelimiterLen = len(topDelimiter)
		botDelimiterLen = len(botDelimiter)
	)

	checkDelimiterBytes := func(delim, data []byte) bool {
		if len(data) >= len(delim) {
			return string(delim) == string(data[:len(delim)])
		}
		return false
	}

	return func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}

		if firstTime {
			firstTime = false
			if checkDelimiterBytes(topDelimiter, data) {
				checkForBotDelimiter = true
				return topDelimiterLen, retDelimiter, nil
			}
		}

		if checkForBotDelimiter {
			if checkDelimiterBytes(botDelimiter, data) {
				checkForBotDelimiter = false
				return botDelimiterLen, retDelimiter, nil
			}
		}

		return 1, data[:1], nil
	}
}

// jsonMarshal wraps the json.Marshal function so that the resulting JSON will
// be formatted correctly
func jsonMarshal(data interface{}) ([]byte, error) {
	buf := new(bytes.Buffer)
	b, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	json.Indent(buf, b, "", "\t")
	return buf.Bytes(), nil
}

// tomlMarshal wraps the TOML encoder to a valid marshal function
func tomlMarshal(data interface{}) ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := toml.NewEncoder(buf).Encode(data); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
