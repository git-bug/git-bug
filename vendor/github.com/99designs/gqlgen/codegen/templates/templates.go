//go:generate go run ./inliner/inliner.go

package templates

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"unicode"

	"log"

	"github.com/pkg/errors"
	"golang.org/x/tools/imports"
)

func Run(name string, tpldata interface{}) (*bytes.Buffer, error) {
	t := template.New("").Funcs(template.FuncMap{
		"ucFirst":     ucFirst,
		"lcFirst":     lcFirst,
		"quote":       strconv.Quote,
		"rawQuote":    rawQuote,
		"toCamel":     ToCamel,
		"dump":        dump,
		"prefixLines": prefixLines,
	})

	for filename, data := range data {
		_, err := t.New(filename).Parse(data)
		if err != nil {
			panic(err)
		}
	}

	buf := &bytes.Buffer{}
	err := t.Lookup(name).Execute(buf, tpldata)
	if err != nil {
		return nil, err
	}

	return buf, nil
}

func ucFirst(s string) string {
	if s == "" {
		return ""
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

func lcFirst(s string) string {
	if s == "" {
		return ""
	}

	r := []rune(s)
	r[0] = unicode.ToLower(r[0])
	return string(r)
}

func isDelimiter(c rune) bool {
	return c == '-' || c == '_' || unicode.IsSpace(c)
}

func ToCamel(s string) string {
	buffer := make([]rune, 0, len(s))
	upper := true
	lastWasUpper := false

	for _, c := range s {
		if isDelimiter(c) {
			upper = true
			continue
		}
		if !lastWasUpper && unicode.IsUpper(c) {
			upper = true
		}

		if upper {
			buffer = append(buffer, unicode.ToUpper(c))
		} else {
			buffer = append(buffer, unicode.ToLower(c))
		}
		upper = false
		lastWasUpper = unicode.IsUpper(c)
	}

	return string(buffer)
}

func rawQuote(s string) string {
	return "`" + strings.Replace(s, "`", "`+\"`\"+`", -1) + "`"
}

func dump(val interface{}) string {
	switch val := val.(type) {
	case int:
		return strconv.Itoa(val)
	case int64:
		return fmt.Sprintf("%d", val)
	case float64:
		return fmt.Sprintf("%f", val)
	case string:
		return strconv.Quote(val)
	case bool:
		return strconv.FormatBool(val)
	case nil:
		return "nil"
	case []interface{}:
		var parts []string
		for _, part := range val {
			parts = append(parts, dump(part))
		}
		return "[]interface{}{" + strings.Join(parts, ",") + "}"
	case map[string]interface{}:
		buf := bytes.Buffer{}
		buf.WriteString("map[string]interface{}{")
		var keys []string
		for key := range val {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		for _, key := range keys {
			data := val[key]

			buf.WriteString(strconv.Quote(key))
			buf.WriteString(":")
			buf.WriteString(dump(data))
			buf.WriteString(",")
		}
		buf.WriteString("}")
		return buf.String()
	default:
		panic(fmt.Errorf("unsupported type %T", val))
	}
}

func prefixLines(prefix, s string) string {
	return prefix + strings.Replace(s, "\n", "\n"+prefix, -1)
}

func RenderToFile(tpl string, filename string, data interface{}) error {
	var buf *bytes.Buffer
	buf, err := Run(tpl, data)
	if err != nil {
		return errors.Wrap(err, filename+" generation failed")
	}

	if err := write(filename, buf.Bytes()); err != nil {
		return err
	}

	log.Println(filename)

	return nil
}

func gofmt(filename string, b []byte) ([]byte, error) {
	out, err := imports.Process(filename, b, nil)
	if err != nil {
		return b, errors.Wrap(err, "unable to gofmt")
	}
	return out, nil
}

func write(filename string, b []byte) error {
	err := os.MkdirAll(filepath.Dir(filename), 0755)
	if err != nil {
		return errors.Wrap(err, "failed to create directory")
	}

	formatted, err := gofmt(filename, b)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gofmt failed: %s\n", err.Error())
		formatted = b
	}

	err = ioutil.WriteFile(filename, formatted, 0644)
	if err != nil {
		return errors.Wrapf(err, "failed to write %s", filename)
	}

	return nil
}
