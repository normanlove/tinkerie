package flashie

import (
	"bytes"
	"encoding/xml"
	"strconv"
	"strings"
)

// parsePlist decodes an XML property list into maps, slices and scalars.
func parsePlist(b []byte) (any, error) {
	dec := xml.NewDecoder(bytes.NewReader(b))
	var parse func(start xml.StartElement) (any, error)
	parse = func(start xml.StartElement) (any, error) {
		switch start.Name.Local {
		case "dict":
			m := map[string]any{}
			key := ""
			for {
				t, err := dec.Token()
				if err != nil {
					return nil, err
				}
				switch t := t.(type) {
				case xml.StartElement:
					if t.Name.Local == "key" {
						var k string
						if err := dec.DecodeElement(&k, &t); err != nil {
							return nil, err
						}
						key = k
						continue
					}
					v, err := parse(t)
					if err != nil {
						return nil, err
					}
					m[key] = v
				case xml.EndElement:
					return m, nil
				}
			}
		case "array":
			var a []any
			for {
				t, err := dec.Token()
				if err != nil {
					return nil, err
				}
				switch t := t.(type) {
				case xml.StartElement:
					v, err := parse(t)
					if err != nil {
						return nil, err
					}
					a = append(a, v)
				case xml.EndElement:
					return a, nil
				}
			}
		case "true", "false":
			dec.Skip()
			return start.Name.Local == "true", nil
		default:
			var s string
			if err := dec.DecodeElement(&s, &start); err != nil {
				return nil, err
			}
			if start.Name.Local == "integer" {
				n, _ := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
				return n, nil
			}
			return s, nil
		}
	}
	for {
		t, err := dec.Token()
		if err != nil {
			return nil, err
		}
		if se, ok := t.(xml.StartElement); ok && se.Name.Local != "plist" {
			return parse(se)
		}
	}
}
