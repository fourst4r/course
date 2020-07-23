package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/fourst4r/course"
)

func main() {
	c := course.Default()
	c.Stamp1.Push(13348, 10085, course.Text{
		Content: "generated pog",
		ScaleX:  1,
		ScaleY:  1,
	})
	err := ioutil.WriteFile("gen.txt", []byte(c.String("oxy")), os.ModeAppend)
	if err != nil {
		panic(err)
	}
	fmt.Printf("ok")
}
