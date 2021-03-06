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

	"github.com/hashicorp/hcl/hcl/printer"
	"github.com/hashicorp/terraform/terraform"
)

type ResourcesFormat struct {
	Name       string
	ID         string
	Attributes map[string]interface{}
}
type ResourcesBuilder struct {
	Buffer bytes.Buffer
}

func main() {
	var state terraform.State
	if err := json.NewDecoder(os.Stdin).Decode(&state); err != nil {
		log.Fatalln(err)
	}

	var resourcesFormats []ResourcesFormat
	for _, module := range state.Modules {
		for name, resources := range module.Resources {
			var resourcesFormat ResourcesFormat
			resourcesFormat.Attributes = make(map[string]interface{})
			resourcesName := strings.Split(name, ".")
			resourcesFormat.Name = resourcesName[0]
			resourcesFormat.ID = resourcesName[1]
			for key, val := range resources.Primary.Attributes {

				if strings.Contains(key, ".") {
					maps := strings.Split(key, ".")
					if strings.Contains(maps[1], "#") || strings.Contains(maps[1], "%") {
						continue
					}

					if _, ok := resourcesFormat.Attributes[maps[0]]; !ok {
						resourcesFormat.Attributes[maps[0]] = make(map[string]interface{})
					}
					if tmp, ok := resourcesFormat.Attributes[maps[0]].(map[string]interface{}); ok {
						tmp[maps[1]] = val
						resourcesFormat.Attributes[maps[0]] = tmp
					} else {
						fmt.Printf("%T", tmp)
						log.Fatalln("cast error")
					}

				} else {
					resourcesFormat.Attributes[key] = val
				}
			}
			resourcesFormats = append(resourcesFormats, resourcesFormat)
		}
	}

	for _, resourcesFormat := range resourcesFormats {
		builder := &ResourcesBuilder{}
		builder.Build(resourcesFormat)
		res, err := printer.Format(builder.Buffer.Bytes())
		if err != nil {
			log.Fatalln(err)
		}
		fmt.Println(string(res))
	}

}

func (builder *ResourcesBuilder) Build(resourcesFormat ResourcesFormat) {
	fmt.Fprintf(&builder.Buffer, "resource %q %q {\n", resourcesFormat.Name, resourcesFormat.ID)
	builder.printAttributes(resourcesFormat.Attributes)
	fmt.Fprintf(&builder.Buffer, "}\n")
}

func (builder *ResourcesBuilder) printAttributes(attributes map[string]interface{}) {
	for key, value := range attributes {

		switch val := value.(type) {
		case string:
			builder.printString(key, val)
		case map[string]interface{}:
			builder.printMap(key, val)
		}

	}
}

func (builder *ResourcesBuilder) printString(key string, value interface{}) {
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

func (builder *ResourcesBuilder) printMap(key string, attributes map[string]interface{}) {
	for k, value := range attributes {
		if val, ok := value.(string); ok {
			//map or list
			i, err := strconv.Atoi(k)
			if err == nil && i == hashcode.String(val) {
				builder.printTypeSet(key, attributes)
				return
			}
		}
	}
	fmt.Fprintf(&builder.Buffer, "%s {\n", key)

	builder.printAttributes(attributes)
	fmt.Fprintf(&builder.Buffer, "}\n")
}
func (builder *ResourcesBuilder) printTypeSet(key string, attributes map[string]interface{}) {
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
