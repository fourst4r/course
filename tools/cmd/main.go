package main

import (
	"fmt"
	"image/color"
	"io/ioutil"
	"math/rand"
	"os"

	"github.com/fourst4r/course"
)

func hex2col(h uint) color.RGBA {
	r := uint8(h & 0xFF0000 >> 16)
	g := uint8(h & 0x00FF00 >> 8)
	b := uint8(h & 0x0000FF)
	// fmt.Printf("r%d g%d b%d\n", r, g, b)
	return color.RGBA{r, g, b, 0}
}

func text(content string, color uint) course.Text {
	return course.Text{
		Content: content,
		Color:   hex2col(color),
		ScaleX:  1,
		ScaleY:  1,
	}
}

func main() {
	c := course.Default()

	c.Stamp1 = make(course.Layer)

	startX := 12390
	startY := 10050

	for y := 0; y < 33; y++ {
		for x := 0; x < 33; x++ {
			c.Stamp1.Push(startX+x, startY+y, text(".", uint(rand.Intn(0xffffff))))
		}
	}

	// c.Stamp1.Push(12390, 10050, text(".", 0xffff00))
	// c.Stamp1.Push(12390, 10051, text(".", 0x0000ff))
	// c.Stamp1.Push(12391, 10050, text(".", 0x00ff00))
	// c.Stamp1.Push(12391, 10051, text(".", 0xff0000))
	err := ioutil.WriteFile("gen.txt", []byte(c.String("oxy")), os.ModeAppend)
	if err != nil {
		panic(err)
	}
	fmt.Printf("ok")
}
