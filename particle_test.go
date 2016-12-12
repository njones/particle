package particle

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

var testCaseData = map[string]map[string]string{
	"YAML": map[string]string{
		"file": `---
name: John Doe
date: 10-10-2016
title: example YAML
---

This is an example file.
`,
	},
	"TOML": map[string]string{
		"file": `+++
Name = "John Doe"
Date = "10-10-2016"
Title = "example TOML"
+++

This is an example file.
`,
	},
	"JSON": map[string]string{
		"file": `{
	"Name": "John Doe",
	"Date": "10-10-2016",
	"Title": "example JSON"
}

This is an example file.
`,
	},
}

type testMetaData struct {
	Name, Date, Title string
}

var wantInt = 25
var wantContent = "This is an example file.\n"
var wantMetaData = testMetaData{Name: "John Doe", Date: "10-10-2016"}

// TODO: Test Large File
// TODO: Test Bad fontmatter block
// TODO: Test Unicode in the Fontmatter Block

func TestCustomEncoding(t *testing.T) {
	wantDelimiter := "xoxo"
	wantOutputDelimiter := true

	var wantMarshalFunc MarshalFunc = func(i interface{}) ([]byte, error) {
		return []byte("custom: data"), nil
	}

	var wantUnmarshalFunc UnmarshalFunc = func(b []byte, i interface{}) error {
		switch oo := i.(type) {
		case map[string]string:
			oo["custom"] = "data"
		}
		return nil
	}

	haveEnc := NewEncoding(
		WithDelimiter(wantDelimiter),
		WithMarshalFunc(wantMarshalFunc),
		WithUnmarshalFunc(wantUnmarshalFunc),
		WithIncludeDelimiter(),
	)

	if wantDelimiter != haveEnc.delimiter {
		t.Errorf("want: %+v have: %+v", wantDelimiter, haveEnc.delimiter)
	}

	if wantOutputDelimiter != haveEnc.outputDelimiter {
		t.Errorf("want: %+v have: %+v", wantOutputDelimiter, haveEnc.outputDelimiter)
	}

	// They should have the same pointer (at least)
	if &wantMarshalFunc == &haveEnc.marshalFunc {
		t.Errorf("want: %v have: %v", wantMarshalFunc, haveEnc.marshalFunc)
	}

	if &wantUnmarshalFunc == &haveEnc.unmarshalFunc {
		t.Errorf("want: %+v have: %+v", wantUnmarshalFunc, haveEnc.unmarshalFunc)
	}
}

func TestEncoding(t *testing.T) {

	var runner = []struct {
		Name     string
		Encoding *Encoding
	}{
		{"YAML", YAMLEncoding},
		{"TOML", TOMLEncoding},
		{"JSON", JSONEncoding},
	}

	for _, r := range runner {
		t.Log("Testing: " + r.Name)

		wantMetaData.Title = "example " + r.Name
		wantContentFile := testCaseData[r.Name]["file"]

		haveContent1 := new(bytes.Buffer)
		out, err := NewEncoder(r.Encoding, haveContent1, wantMetaData)
		if err != nil {
			t.Errorf(r.Name+"(NewEncoder): err: %s", err)
		}
		out.Write([]byte(wantContent))

		if wantContentFile != haveContent1.String() {
			t.Errorf(r.Name+"(NewEncoder): \nwant: %+v \nhave: %+v", wantContentFile, haveContent1.String())
		}

		haveContent2 := r.Encoding.EncodeToString([]byte(wantContent), wantMetaData)

		if wantContentFile != haveContent2 {
			t.Errorf(r.Name+"(EncodeToString): \nwant: %+v \nhave: %+v", wantContentFile, haveContent2)
		}

		haveContent3 := make([]byte, len(wantContentFile))
		r.Encoding.Encode(haveContent3, []byte(wantContent), wantMetaData)

		if wantContentFile != string(haveContent3) {
			t.Errorf(r.Name+"(Encode): \nwant: %+v \nhave: %+v", wantContentFile, string(haveContent3))
		}
	}
}

func TestDecoding(t *testing.T) {

	var runner = []struct {
		Name     string
		Encoding *Encoding
	}{
		{"YAML", YAMLEncoding},
		{"TOML", TOMLEncoding},
		{"JSON", JSONEncoding},
	}

	for _, r := range runner {
		t.Log("Testing: " + r.Name)

		wantMetaData.Title = "example " + r.Name
		wantContentFile := testCaseData[r.Name]["file"]

		haveMetaData1 := testMetaData{}
		out, err := NewDecoder(r.Encoding, strings.NewReader(wantContentFile), &haveMetaData1)
		if err != nil {
			t.Errorf(r.Name+"(NewDecoder): err: %s", err)
		}

		haveContent1 := new(bytes.Buffer)
		haveContent1.ReadFrom(out)

		if wantContent != haveContent1.String() {
			t.Errorf(r.Name+"(NewDecoder): \nwant: %+v \nhave: %+v", wantContent, haveContent1.String())
		}

		if !reflect.DeepEqual(wantMetaData, haveMetaData1) {
			t.Errorf(r.Name+"(DecodeString): \nwant: %+v \nhave: %+v", wantMetaData, haveMetaData1)
		}

		haveMetaData2 := testMetaData{}
		haveContent2, err := r.Encoding.DecodeString(wantContentFile, &haveMetaData2)
		if err != nil {
			t.Errorf(r.Name+"(DecodeString): err %s", err)
		}

		if wantContent != string(haveContent2) {
			t.Errorf(r.Name+"(DecodeString): \nwant: %+v \nhave: %+v", wantContent, string(haveContent2))
		}

		if !reflect.DeepEqual(wantMetaData, haveMetaData2) {
			t.Errorf(r.Name+"(DecodeString): \nwant: %+v \nhave: %+v", wantMetaData, haveMetaData2)
		}

		var (
			haveMetaData3 = testMetaData{}
			haveContent3  = make([]byte, wantInt)
		)

		haveInt3, err := r.Encoding.Decode(haveContent3, []byte(wantContentFile), &haveMetaData3)
		if err != nil {
			t.Errorf(r.Name+"(Decode): err %s", err)
		}

		if wantInt != haveInt3 {
			t.Errorf(r.Name+"(Decode): \nwant: %+v \nhave: %+v", wantInt, string(haveInt3))
		}

		if wantContent != string(haveContent3) {
			t.Errorf(r.Name+"(Decode): \nwant: %+v \nhave: %+v", wantContent, string(haveContent3))
		}

		if !reflect.DeepEqual(wantMetaData, haveMetaData3) {
			t.Errorf(r.Name+"(Decode): \nwant: %+v \nhave: %+v", wantMetaData, haveMetaData3)
		}
	}
}
