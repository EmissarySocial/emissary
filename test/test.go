package main

import (
	"fmt"
)

type T1 interface {
	update(string)
}

type Data struct {
	value string
}

func (d *Data) update(v string) {
	d.value = v
}

func updateViaInterface(t1 T1, value string) {
	t1.update(value)
}

func main() {

	data := Data{value: "b0rken"}

	updateViaInterface(&data, "worked")

	fmt.Println(data)

}
