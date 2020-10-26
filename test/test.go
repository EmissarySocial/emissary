package main

import (
	"bytes"
	"text/template"

	"github.com/davecgh/go-spew/spew"
)

type Object string

func (o *Object) List() *Object {
	return o
}

func (o *Object) With(value string) *Object {
	*o = Object(value)
	return o
}

func (o *Object) Reverse() string {
	return "reversed: " + string(*o)
}

func main() {

	/*
	t1 := Object("")
	t1.With("howdy")
	spew.Dump(t1.Reverse())

	return
	*/

	var result bytes.Buffer

	t, err := template.New("test").Parse(`{{(.List.With "hey-oh").Reverse}}`)

	if err != nil {
		spew.Dump(err)
		return
	}

	o := Object("")

	if err := t.Execute(&result, &o); err != nil {
		spew.Dump(err)
		return
	}

	spew.Dump(result.String())
}