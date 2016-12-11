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
	YAMLEncoding = NewEncoding(WithDelimiter(YAMLDelimiter), WithMarshalFunc(yaml.Marshal), WithUnmarshalFunc(yaml.Unmarshal))
	TOMLEncoding = NewEncoding(WithDelimiter(TOMLDelimiter), WithMarshalFunc(tomlMarshal), WithUnmarshalFunc(toml.Unmarshal))
	JSONEncoding = NewEncoding(WithDelimiter(JSONDelimiterPair), WithMarshalFunc(jsonMarshal), WithUnmarshalFunc(json.Unmarshal), WithSplitFunc(SpaceSeparatedTokenDelimiters), WithIncludeDelimiter())
)

type SplitFunc func(string) (string, string, bufio.SplitFunc)
type MarshalFunc func(interface{}) ([]byte, error)
type UnmarshalFunc func([]byte, interface{}) error
type EncodingOptionFunc func(*Encoding) error

type Writer struct{ w io.Writer }

func (l *Writer) Write(p []byte) (n int, err error) {
	n, err = l.w.Write(p)
	return
}

func WithDelimiter(s string) EncodingOptionFunc {
	return func(e *Encoding) error {
		e.delimiter = s
		return nil
	}
}

func WithMarshalFunc(fn MarshalFunc) EncodingOptionFunc {
	return func(e *Encoding) error {
		e.marshalFunc = fn
		return nil
	}
}

func WithUnmarshalFunc(fn UnmarshalFunc) EncodingOptionFunc {
	return func(e *Encoding) error {
		e.unmarshalFunc = fn
		return nil
	}
}

func WithIncludeDelimiter() EncodingOptionFunc {
	return func(e *Encoding) error {
		e.outputDelimiter = true
		return nil
	}
}

func WithSplitFunc(fn SplitFunc) EncodingOptionFunc {
	return func(e *Encoding) error {
		e.inSplitFunc = fn
		return nil
	}
}

func NewDecoder(e *Encoding, r io.Reader, v interface{}) (io.Reader, error) {
	m, o := e.readFrom(r)
	if err := e.readUnmarshal(m, v); err != nil {
		return nil, err
	}

	return o, nil
}

func NewEncoder(e *Encoding, w io.Writer, v interface{}) (io.Writer, error) {
	o := &Writer{w: w}

	f, err := e.encodeFrontmatter(v)
	if err != nil {
		return nil, err
	}
	o.Write(f) // write frontmatter first

	return o, nil
}

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

func (e *Encoding) Decode(dst, src []byte, v interface{}) (int, error) {
	m, r := e.readFrom(bytes.NewBuffer(src))
	if err := e.readUnmarshal(m, v); err != nil {
		return 0, err
	}

	return io.ReadFull(r, dst)
}

func (e *Encoding) DecodeString(src string, v interface{}) ([]byte, error) {
	return e.DecodeReader(bytes.NewBufferString(src), v)
}

func (e *Encoding) DecodeReader(r io.Reader, v interface{}) ([]byte, error) {
	m, r := e.readFrom(r)
	if err := e.readUnmarshal(m, v); err != nil {
		return nil, err
	}
	return ioutil.ReadAll(r)
}

func (e *Encoding) EncodeToString(src []byte, v interface{}) string {
	b := make([]byte, e.EncodeLen(src, v))
	e.Encode(b, src, v)
	return string(b)
}

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

func (e *Encoding) EncodeLen(src []byte, v interface{}) int {
	f, err := e.encodeFrontmatter(v)
	if err != nil {
		panic(err)
	}
	return len(f) + len(src)
}

func (e *Encoding) hashFrontmatter(v interface{}) string {
	h := md5.Sum([]byte(fmt.Sprintf("%#v", v)))
	return string(h[:])
}

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

func (e *Encoding) readFrom(r io.Reader) (frontmatter, content io.Reader) {
	mr, mw := io.Pipe()
	cr, cw := io.Pipe()

	go func() {
		defer mw.Close() // if the matter writer is never written to...
		defer cw.Close() // if data witer is never written to...

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

func SingleTokenDelimiter(delim string) (start string, end string, fn bufio.SplitFunc) {
	return delim, delim, baseSplitter([]byte(delim+"\n"), []byte("\n"+delim+"\n"), []byte(delim))
}

func SpaceSeparatedTokenDelimiters(delim string) (start string, end string, fn bufio.SplitFunc) {
	delims := strings.Split(delim, " ")
	if len(delims) != 2 {
		panic("The delimiter token does not split into exactly two")
	}
	start, end = delims[0], delims[1]
	return start, end, baseSplitter([]byte(start+"\n"), []byte("\n"+end+"\n"), []byte(delim))
}

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

func jsonMarshal(data interface{}) ([]byte, error) {
	buf := new(bytes.Buffer)
	b, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	json.Indent(buf, b, "", "\t")
	return buf.Bytes(), nil
}

func tomlMarshal(data interface{}) ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := toml.NewEncoder(buf).Encode(data); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
