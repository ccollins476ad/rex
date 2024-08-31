package dest

import (
	"fmt"
	"strconv"
	"strings"
)

// Example dest specifier string:
// type=fifo,id=/tmp/myfifo,nonblocking,bufsize=102400,create

type parser struct {
	d       Dest
	keyVals map[string]string
}

// Parse parses the given dest specifier string, returning the resulting Dest
// on success, error on failure.
func Parse(s string) (*Dest, error) {
	p := parser{
		d:       makeDest(),
		keyVals: map[string]string{},
	}

	err := p.parse(s)
	if err != nil {
		return nil, err
	}

	return &p.d, nil
}

// Parse parses the given dest specifier string, writing the result to the
// parser's internal dest field. It returns an error on parse failure.
func (p *parser) parse(s string) error {
	fail := func(err error) error {
		return fmt.Errorf("invalid dest: dest=[%s]: %w", s, err)
	}

	// Dest fields are separated by commas.
	fields := strings.Split(s, ",")

	for _, t := range fields {
		err := p.parseField(t)
		if err != nil {
			return fail(fmt.Errorf("parse failure: field=[%s]: %w", t, err))
		}
	}

	if p.d.Type == unsetType {
		return fmt.Errorf("missing 'type' field")
	}

	if p.d.ID == "" {
		return fmt.Errorf("missing 'id' field")
	}

	return nil
}

// parseField parses a single dest specifier field. On success, it populates
// the parser's internal dest struct accordingly.
func (p *parser) parseField(field string) error {
	var err error

	// Some fields have `k=v` notation, others have `x`. Determine which type
	// of field this is by checking for the presence of an `=` character.
	parts := strings.SplitN(field, "=", 2)
	if len(parts) == 2 {
		err = p.parseKeyVal(parts[0], parts[1])
	} else {
		err = p.parseStandalone(field)
	}
	return err
}

func (p *parser) parseKeyVal(k string, v string) error {
	// Don't allow the same key to be specified twice in a dest specifier
	// string.
	if p.keyVals[k] == "" {
		p.keyVals[k] = v
	} else if p.keyVals[k] != v {
		return fmt.Errorf("duplicate keyval: key=%s val1=%s val2=%s", k, p.keyVals[k], v)
	}

	invalidVal := func(err error) error {
		return fmt.Errorf("invalid %s: %w", k, err)
	}

	switch k {
	case "type":
		dt, ok := nameTypeMap[v]
		if !ok {
			return fmt.Errorf("unrecognized type: %s", k)
		}
		p.d.Type = dt
		return nil

	case "id":
		p.d.ID = v
		return nil

	case "perm":
		perm, err := strconv.ParseUint(v, 0, 32)
		if err != nil {
			return invalidVal(err)
		}
		p.d.Perm = uint32(perm)
		return nil

	case "args":
		allArgs := strings.TrimSpace(v)
		p.d.Args = strings.Fields(allArgs)
		return nil

	case "bufsize":
		bs, err := strconv.Atoi(v)
		if err != nil {
			return invalidVal(err)
		}
		p.d.BufSize = bs
		return nil

	default:
		return fmt.Errorf("unrecognized key: %s", k)
	}
}

func (p *parser) parseStandalone(field string) error {
	switch field {
	case "nonblocking":
		p.d.NonBlocking = true
		return nil

	case "create":
		p.d.Create = true
		return nil

	case "append":
		p.d.Append = true
		return nil

	default:
		return fmt.Errorf("unrecognized field")
	}
}
