package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"unicode"

	"github.com/hashicorp/terraform/helper/hashcode"

	"github.com/davecgh/go-spew/spew"

	"github.com/hashicorp/hcl/hcl/printer"
	"github.com/hashicorp/terraform/terraform"
)

type ResouceFormart struct {
	Name       string
	Target     string
	Attributes map[string]interface{}
}
type ResouceBuilder struct {
	Buffer bytes.Buffer
}

func main() {
	f, err := os.Open("./terraform.tfstate")
	if err != nil {
		log.Fatalln(err)
	}
	var state terraform.State
	if err = json.NewDecoder(f).Decode(&state); err != nil {
		log.Fatalln(err)
	}

	var resouceFormats []ResouceFormart

	for _, module := range state.Modules {
		var resouceFormat ResouceFormart
		resouceFormat.Attributes = make(map[string]interface{})
		for name, resouce := range module.Resources {
			resouceName := strings.Split(name, ".")
			resouceFormat.Name = resouceName[0]
			resouceFormat.Target = resouceName[1]

			for key, val := range resouce.Primary.Attributes {
				fmt.Printf("key:%#v\tvalue:%#v\n", key, val)
				if strings.Contains(key, ".") {
					maps := strings.Split(key, ".")
					//keyが存在しないときはmapを作成する
					if _, ok := resouceFormat.Attributes[maps[0]]; !ok {
						resouceFormat.Attributes[maps[0]] = make(map[string]interface{})
					}
					tmp, ok := resouceFormat.Attributes[maps[0]].(map[string]interface{})
					if !ok {
						fmt.Printf("%T", tmp)
						log.Fatalln("cast error")
					}
					tmp[maps[1]] = val
					resouceFormat.Attributes[maps[0]] = tmp

				} else {
					resouceFormat.Attributes[key] = val
				}
			}
		}
		resouceFormats = append(resouceFormats, resouceFormat)
	}

	spew.Dump(resouceFormats)

	for _, resouceFormat := range resouceFormats {
		builder := &ResouceBuilder{}
		builder.Printer(resouceFormat)
		fmt.Println(string(builder.Buffer.Bytes()))
		res, err := printer.Format(builder.Buffer.Bytes())
		if err != nil {
			log.Fatalln(err)
		}
		fmt.Println(string(res))
	}

}

func (builder *ResouceBuilder) Printer(resouceFormat ResouceFormart) {
	fmt.Fprintf(&builder.Buffer, "resource %q %q {\n", resouceFormat.Name, resouceFormat.Target)
	builder.PrintAttributes(resouceFormat.Attributes)
	fmt.Fprintf(&builder.Buffer, "}\n")
}
func (builder *ResouceBuilder) PrintAttributes(attributes map[string]interface{}) {
	for key, value := range attributes {

		switch val := value.(type) {
		case string:
			builder.PrintString(key, val)
		case map[string]interface{}:
			builder.PrintMap(key, val)
		}

	}
}

func (builder *ResouceBuilder) PrintString(key string, value interface{}) {
	head := []rune(key)[0]
	switch {
	case isLetter(head):
		//nop
	default:
		key = fmt.Sprintf("\"%v\"", key)
	}
	switch val := value.(type) {
	case string:
		fmt.Fprintf(&builder.Buffer, "%s = %q\n", key, val)
	case int, int8, int16, int32, int64:
		fmt.Fprintf(&builder.Buffer, "%s = %d\n", key, val)
	}
}

func (builder *ResouceBuilder) PrintMap(key string, attributes map[string]interface{}) {
	for k, value := range attributes {
		if val, ok := value.(string); ok {
			i, err := strconv.Atoi(k)
			if err == nil && i == hashcode.String(val) {
				builder.PrintTypeSet(key, attributes)
				return
			}
		}
	}
	fmt.Fprintf(&builder.Buffer, "%s {\n", key)

	builder.PrintAttributes(attributes)
	fmt.Fprintf(&builder.Buffer, "}\n")
}
func (builder *ResouceBuilder) PrintTypeSet(key string, attributes map[string]interface{}) {
	fmt.Fprintf(&builder.Buffer, "%s = [", key)
	for k, v := range attributes {
		if strings.Contains(k, "#") {
			continue
		}
		fmt.Fprintf(&builder.Buffer, " %q,", v.(string))
	}
	fmt.Fprintf(&builder.Buffer, "]\n")
}

// isHexadecimal returns true if the given rune is a letter
func isLetter(ch rune) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_' || ch >= 0x80 && unicode.IsLetter(ch)
}
