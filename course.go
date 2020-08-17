package course

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"image/color"
	"net/url"
	"strconv"
	"strings"

	"golang.org/x/image/colornames"
)

type (
	Block int
	Stamp struct {
		Type           int
		ScaleX, ScaleY float64
	}
	Text struct {
		Content        string
		ScaleX, ScaleY float64
		Color          color.RGBA
	}
	Line struct {
		Erase     bool
		Thickness int
		Color     color.RGBA
		Segments  []XY
	}
)

type XY struct{ X, Y int }
type Layer map[XY][]interface{}

func (l Layer) Get(x, y int) ([]interface{}, bool) {
	s, ok := l[XY{x, y}]
	return s, ok
}

func (l Layer) Peek(x, y int) (interface{}, bool) {
	s, ok := l[XY{x, y}]
	if !ok {
		return nil, false
	}
	return s[len(s)-1], true
}

func (l Layer) Push(x, y int, obj interface{}) {
	pos := XY{x, y}
	s, ok := l[pos]
	if !ok {
		l[pos] = []interface{}{obj}
	} else {
		l[pos] = append(s, obj)
	}
}

func (l Layer) Pop(x, y int) interface{} {
	pos := XY{x, y}
	s, ok := l[pos]
	if !ok {
		return -1
	}

	var popped interface{}
	popped, l[pos] = s[len(s)-1], s[:len(s)-1]

	if len(s) == 1 {
		delete(l, pos)
	}

	return popped
}

const currentDataFormat = "m3"

var (
	textContentEncoder = strings.NewReplacer("`", "#96", ",", "#44", ";", "#59", "#", "#35")
	textContentDecoder = strings.NewReplacer("#96", "`", "#44", ",", "#59", ";", "#35", "#")
)

type Course struct {
	Live            bool
	HasPass         bool
	Title           string
	Note            string
	GameMode        string
	Credits         []string
	Gravity         float64
	MaxTime         int
	MinRank         int
	Song            int
	CowboyChance    int
	Items           []int
	BackgroundColor color.Color
	BackgroundImage int

	Blocks                                  Layer
	Line00, Line0, Line1, Line2, Line3      Layer
	Stamp00, Stamp0, Stamp1, Stamp2, Stamp3 Layer

	Pass string

	queries map[string]string
}

// Default level as in PR2 level editor.
func Default() *Course {
	c := new(Course)
	c.BackgroundColor = hex2col(0xbbbbdd)
	c.BackgroundImage = -1
	c.MaxTime = 120
	c.Gravity = 1.0
	c.CowboyChance = 5
	c.GameMode = "race"
	c.Items = []int{1, 2, 3, 4, 5, 6, 7, 8, 9}
	c.Blocks = make(Layer)
	c.Stamp1 = make(Layer)
	c.Stamp2 = make(Layer)
	c.Stamp3 = make(Layer)
	c.Line1 = make(Layer)
	c.Line2 = make(Layer)
	c.Line3 = make(Layer)
	c.Stamp00 = make(Layer)
	c.Stamp0 = make(Layer)
	c.Line00 = make(Layer)
	c.Line0 = make(Layer)
	c.Blocks.Push(12390, 10050, 111)
	c.Blocks.Push(12420, 10050, 112)
	c.Blocks.Push(12450, 10050, 113)
	c.Blocks.Push(12480, 10050, 114)
	return c
}

func parseBlocks(format, data string) (blocks Layer, err error) {
	blocks = make(Layer)
	switch format {
	case "o":
	case "m1":
	case "m2":
	case "m3":
		var curX, curY, curT int
		for _, o := range strings.Split(data, ",") {
			var dx, dy int

			e := strings.Split(o, ";")
			dx, err = strconv.Atoi(e[0])
			if err != nil {
				err = fmt.Errorf("block dx: %v", err)
				return
			}
			curX += dx
			if len(e) > 1 {
				dy, err = strconv.Atoi(e[1])
				if err != nil {
					err = fmt.Errorf("block dy: %v", err)
					return
				}
				curY += dy
			}
			if len(e) > 2 {
				curT, err = strconv.Atoi(e[2])
				if err != nil {
					err = fmt.Errorf("block t: %v", err)
					return
				}
				// if curT > 100 {
				// 	curT -= 100
				// }
			}
			// normalize blocks to pixel coords, like everything else
			blocks.Push(curX*30, curY*30, curT)
		}
	}
	return
}

func parseLines(format, data string) (art Layer, err error) {
	art = make(Layer)

	if data == "" {
		return
	}

	switch format {
	case "m3":
		var (
			mode      = "draw"
			thickness = 4
			col       = colornames.Black
		)

		commands := strings.Split(data, ",")
		for _, command := range commands {
			typ := command[0:1]
			content := command[1:]
			switch typ {
			case "c":
				var hex uint64
				hex, err = strconv.ParseUint(content, 16, 32)
				if err != nil {
					err = fmt.Errorf("line color: %v", err)
					return
				}
				col = hex2col(uint(hex))
			case "t":
				thickness, err = strconv.Atoi(content)
				if err != nil {
					err = fmt.Errorf("line thickness: %v", err)
					return
				}
			case "d":
				polyline := []XY{}
				values := strings.Split(content, ";")
				var lx, ly, dx, dy int
				for i := 0; i < len(values); i += 2 {
					dx, err = strconv.Atoi(values[i])
					dy, err = strconv.Atoi(values[i+1])
					lx += dx
					ly += dy
					polyline = append(polyline, XY{lx, ly})
				}
				art.Push(polyline[0].X, polyline[0].Y, &Line{
					Erase:     mode != "draw",
					Thickness: thickness,
					Color:     col,
					Segments:  polyline,
				})
			case "m":
				mode = content
			}
		}
	}
	return
}

func parseStamps(format, data string) (Layer, error) {
	layer := make(Layer)
	if data == "" {
		return layer, nil
	}

	contentDecoder := strings.NewReplacer("#96", "`", "#44", ",", "#59", ";", "#35", "#")

	switch format {
	case "m3":
		var (
			curX, curY int
			id         string
		)
		for _, command := range strings.Split(data, ",") {
			split := strings.Split(command, ";")

			dx, err := strconv.Atoi(split[0])
			if err != nil {
				return layer, fmt.Errorf("stamp x: %v", err)
			}
			dy, err := strconv.Atoi(split[1])
			if err != nil {
				return layer, fmt.Errorf("stamp y: %v", err)
			}
			curX += dx
			curY += dy

			if len(split) < 3 {
				id = "0"
			} else {
				id = split[2]
			}

			if id == "t" {
				// text object
				text := &Text{}
				text.Content = contentDecoder.Replace(split[3])

				var hex int
				hex, err = strconv.Atoi(split[4])
				if err != nil {
					return layer, fmt.Errorf("text color: %v", err)
				}
				text.Color = hex2col(uint(hex))

				scaleX, err := strconv.Atoi(split[5])
				if err != nil {
					return layer, fmt.Errorf("scaleX: %v", err)
				}
				text.ScaleX = float64(scaleX) / 100

				scaleY, err := strconv.Atoi(split[6])
				if err != nil {
					return layer, fmt.Errorf("scaleY: %v", err)
				}
				text.ScaleY = float64(scaleY) / 100

				layer.Push(curX, curY, text)
			} else {
				// stamp
				stamp := &Stamp{}
				stamp.ScaleX = 1.0
				stamp.ScaleY = 1.0

				if len(split) > 3 {
					scaleX, err := strconv.Atoi(split[3])
					if err != nil {
						return layer, fmt.Errorf("scaleX: %v", err)
					}
					stamp.ScaleX = float64(scaleX) / 100

					scaleY, err := strconv.Atoi(split[4])
					if err != nil {
						return layer, fmt.Errorf("scaleY: %v", err)
					}
					stamp.ScaleY = float64(scaleY) / 100
				}

				layer.Push(curX, curY, stamp)
			}

		}
	}

	return layer, nil
}

func (c *Course) parseData(data string) error {
	split := strings.Split(data, "`")
	format := split[0]

	var err error
	// hex, err := strconv.ParseUint(split[1], 16, 32)
	c.BackgroundColor, err = parseColor(split[1])
	if err != nil {
		return fmt.Errorf("background color: %v", err)
	}
	// c.BackgroundColor = hex2col(uint(hex))

	c.Blocks, err = parseBlocks(format, split[2])
	if err != nil {
		return fmt.Errorf("blocks: %v", err)
	}

	c.Stamp1, err = parseStamps(format, split[3])
	if err != nil {
		return fmt.Errorf("stamp1: %v", err)
	}
	c.Stamp2, err = parseStamps(format, split[4])
	if err != nil {
		return fmt.Errorf("stamp1: %v", err)
	}
	c.Stamp3, err = parseStamps(format, split[5])
	if err != nil {
		return fmt.Errorf("stamp1: %v", err)
	}

	c.Line1, err = parseLines(format, split[6])
	if err != nil {
		return fmt.Errorf("line1: %v", err)
	}
	c.Line2, err = parseLines(format, split[7])
	if err != nil {
		return fmt.Errorf("line2: %v", err)
	}
	c.Line3, err = parseLines(format, split[8])
	if err != nil {
		return fmt.Errorf("line3: %v", err)
	}

	if split[9] == "" {
		c.BackgroundImage = -1
	} else {
		c.BackgroundImage, err = strconv.Atoi(split[9])
		if err != nil {
			return fmt.Errorf("background image: %v", err)
		}
	}

	if len(split) > 10 {
		c.Stamp0, err = parseStamps(format, split[10])
		if err != nil {
			return fmt.Errorf("stamp0: %v", err)
		}
		c.Stamp00, err = parseStamps(format, split[11])
		if err != nil {
			return fmt.Errorf("stamp00: %v", err)
		}

		c.Line0, err = parseLines(format, split[12])
		if err != nil {
			return fmt.Errorf("line0: %v", err)
		}
		c.Line00, err = parseLines(format, split[13])
		if err != nil {
			return fmt.Errorf("line00: %v", err)
		}
	}
	return nil
}

func parseItems(data string) []int {
	var items = []int{}
	for _, s := range strings.Split(data, "`") {
		switch s {
		case "1", "Laser Gun", "Laser":
			items = append(items, 1)
		case "2", "Mine":
			items = append(items, 2)
		case "3", "Lightning":
			items = append(items, 3)
		case "4", "Teleport":
			items = append(items, 4)
		case "5", "Super Jump":
			items = append(items, 5)
		case "6", "Jet Pack":
			items = append(items, 6)
		case "7", "Speed Burst":
			items = append(items, 7)
		case "8", "Sword":
			items = append(items, 8)
		case "9", "Ice Wave":
			items = append(items, 9)
		}
	}
	return items
}

func parseQuery(query string) (map[string]string, error) {
	m := make(map[string]string)
	for _, qvp := range strings.Split(query, "&") {
		split := strings.Split(qvp, "=")
		if len(split) < 2 {
			split = append(split, "")
		}
		name, value := split[0], split[1]
		m[name] = value
	}
	return m, nil
}

// Parse a course from a data string.
func Parse(data string) (*Course, error) {
	if len(data) < 32 {
		return nil, errors.New("course.Parse: data too short")
	}

	checksum, data := data[len(data)-32:], data[:len(data)-32]
	// checksum is calculated with: <title><lower(user)><data>84ge5tnr
	_ = checksum

	c := Default()

	var err error
	c.queries, err = parseQuery(data)
	if err != nil {
		return nil, fmt.Errorf("course.Parse: query: %v", err)
	}

	for name, value := range c.queries {
		switch name {
		case "live":
			c.Live = value != "0"
		case "has_pass", "hasPass":
			c.HasPass = value != "0"
		case "title":
			c.Title, err = url.QueryUnescape(value)
		case "note":
			c.Note, err = url.QueryUnescape(value)
		case "gameMode":
			c.GameMode = value
		case "credits":
			c.Credits = strings.Split(value, "`")
		case "gravity":
			c.Gravity, err = strconv.ParseFloat(value, 64)
		case "max_time":
			c.MaxTime, err = strconv.Atoi(value)
		case "min_level":
			c.MinRank, err = strconv.Atoi(value)
		case "song":
			if value == "" || value == "random" {
				c.Song = 0
			} else {
				c.Song, err = strconv.Atoi(value)
			}
		case "cowboyChance":
			c.CowboyChance, err = strconv.Atoi(value)
		case "items":
			c.Items = parseItems(value)
		case "data":
			err = c.parseData(value)
		}

		if err != nil {
			return c, fmt.Errorf("course.Parse: %s: %v", name, err)
		}
	}

	return c, nil
}

func formatColor(c color.Color) string {
	r, g, b, _ := c.RGBA()
	return strconv.FormatUint(uint64((r>>8)<<16|(g>>8)<<8|(b>>8)), 16)
}

func parseColor(s string) (col color.Color, err error) {
	hex, err := strconv.ParseUint(s, 16, 32)
	col = hex2col(uint(hex))
	return
}

func formatBlocks(blocks Layer) string {
	var sb strings.Builder
	var curX, curY, curT int
	for xy, stack := range blocks {
		for _, block := range stack {
			sb.WriteByte(',')
			sb.WriteString(strconv.Itoa((xy.X - curX) / 30))
			curX = xy.X
			// if curY != xy.Y {
			sb.WriteByte(';')
			sb.WriteString(strconv.Itoa((xy.Y - curY) / 30))
			curY = xy.Y
			// }
			if curT != block.(int) {
				curT = block.(int)
				sb.WriteByte(';')
				sb.WriteString(strconv.Itoa(curT))
			}
		}
	}
	return sb.String()[1:]
}

func formatStamps(layer Layer) string {
	var sb strings.Builder
	var curX, curY int
	for xy, stack := range layer {
		for _, stamp := range stack {
			sb.WriteByte(',')
			sb.WriteString(strconv.Itoa(xy.X - curX))
			curX = xy.X

			sb.WriteByte(';')
			sb.WriteString(strconv.Itoa(xy.Y - curY))
			curY = xy.Y

			switch v := stamp.(type) {
			case Text:
				// color is NOT hex encoded here
				r, g, b, _ := v.Color.RGBA()
				col := r<<16 | g<<8 | b
				sb.WriteString(fmt.Sprintf(";t;%s;%d;%d;%d",
					v.Content, col, int(v.ScaleX*100), int(v.ScaleY*100)))
			case Stamp:
				// TODO
			default:
				// TODO
			}

		}
	}
	if sb.Len() == 0 {
		return ""
	}
	return sb.String()[1:]
}

func formatLines(layer Layer) string {
	return ""
}

func (c *Course) formatData() string {
	fields := make([]string, 14)
	fields[0] = currentDataFormat
	fields[1] = formatColor(c.BackgroundColor) //hex.EncodeToString([]byte{c.BackgroundColor.R, c.BackgroundColor.G, c.BackgroundColor.B})
	fields[2] = formatBlocks(c.Blocks)
	fields[3] = formatStamps(c.Stamp1)
	fields[4] = formatStamps(c.Stamp2)
	fields[5] = formatStamps(c.Stamp3)
	fields[6] = formatLines(c.Line1)
	fields[7] = formatLines(c.Line2)
	fields[8] = formatLines(c.Line3)
	fields[9] = strconv.Itoa(c.BackgroundImage)
	fields[10] = formatStamps(c.Stamp0)
	fields[11] = formatStamps(c.Stamp00)
	fields[12] = formatStamps(c.Line0)
	fields[13] = formatStamps(c.Line00)
	return strings.Join(fields, "`")
}

func formatQuery(query map[string]string) string {
	var sb strings.Builder
	for name, value := range query {
		sb.WriteByte('&')
		sb.WriteString(name)
		sb.WriteByte('=')
		sb.WriteString(url.QueryEscape(value))
	}
	return sb.String()[1:]
}

func formatBool(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

func (c *Course) formatItems() string {
	if len(c.Items) == 0 {
		return ""
	}
	// itemStrings := []string{"Laser Gun", "Mine", "Lightning", "Teleport", "Super Jump", "Jet Pack", "Speed Burst", "Sword", "Ice Wave"}
	var sb strings.Builder
	sb.WriteString(strconv.FormatInt(int64(c.Items[0]), 32)) // sb.WriteString(itemStrings[c.Items[0]])
	for i := 1; i < len(c.Items); i++ {
		sb.WriteByte('`')
		sb.WriteString(strconv.FormatInt(int64(c.Items[i]), 32)) // sb.WriteString(itemStrings[c.Items[i]-1])
	}
	return sb.String()
}

// String returns a formatted course that can be uploaded to PR2.
func (c *Course) String(user, token string) string {
	c.queries = make(map[string]string)
	c.queries["live"] = formatBool(c.Live)
	c.queries["hasPass"] = formatBool(c.HasPass)
	c.queries["title"] = c.Title
	c.queries["note"] = c.Note
	c.queries["gameMode"] = c.GameMode
	c.queries["credits"] = strings.Join(c.Credits, "`")
	c.queries["gravity"] = strconv.FormatFloat(c.Gravity, 'f', 2, 64)
	c.queries["max_time"] = strconv.Itoa(c.MaxTime)
	c.queries["min_rank"] = strconv.Itoa(c.MinRank)
	c.queries["song"] = strconv.Itoa(c.Song)
	c.queries["cowboyChance"] = strconv.Itoa(c.CowboyChance)
	c.queries["items"] = c.formatItems()
	c.queries["data"] = c.formatData()
	if p := strings.ReplaceAll(c.Pass, "*", ""); p != "" {
		c.queries["passHash"] = md5str(p + "WGZSL3JWcUE9L3Q4YipZIQ==")
	} else {
		c.queries["passHash"] = ""
	}
	c.queries["hash"] = md5str(c.Title + strings.ToLower(user) + c.queries["data"] + "84ge5tnr")
	c.queries["token"] = token
	return formatQuery(c.queries)
}

func (c *Course) Upload(user, token string) string {
	queries := make(map[string]string)
	queries["title"] = c.Title
	queries["note"] = c.Note
	queries["data"] = c.formatData()
	queries["live"] = formatBool(c.Live)
	queries["min_level"] = strconv.Itoa(c.MinRank)
	queries["song"] = strconv.Itoa(c.Song)
	queries["gravity"] = strconv.FormatFloat(c.Gravity, 'f', 2, 64)
	queries["max_time"] = strconv.Itoa(c.MaxTime)
	queries["items"] = c.formatItems()
	queries["hash"] = md5str(c.Title + strings.ToLower(user) + queries["data"] + "84ge5tnr")
	if p := strings.ReplaceAll(c.Pass, "*", ""); p != "" {
		queries["passHash"] = md5str(p + "WGZSL3JWcUE9L3Q4YipZIQ==")
	} else {
		queries["passHash"] = ""
	}
	queries["hasPass"] = formatBool(c.HasPass)
	queries["gameMode"] = c.GameMode
	queries["cowboyChance"] = strconv.Itoa(c.CowboyChance)
	queries["token"] = token
	return formatQuery(queries)
}

func md5str(s string) string {
	sum := md5.Sum([]byte(s))
	return hex.EncodeToString(sum[:])
}

func hex2col(h uint) color.RGBA {
	r := uint8(h & 0xFF0000 >> 16)
	g := uint8(h & 0x00FF00 >> 8)
	b := uint8(h & 0x0000FF)
	return color.RGBA{r, g, b, 0xff}
}
