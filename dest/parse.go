package dest

import (
	"fmt"
	"strconv"
	"strings"
)

type parser struct {
	d       Dest
	keyVals map[string]string
}

// type=fifo:id=/var/log/vserial/fifo:nonblocking:bufsize=102400:create

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

func (p *parser) parse(s string) error {
	fail := func(err error) error {
		return fmt.Errorf("invalid dest: dest=[%s]: %w", s, err)
	}

	tokens := strings.Split(s, ",")

	for _, t := range tokens {
		err := p.parseToken(t)
		if err != nil {
			return fail(fmt.Errorf("parse failure: token=[%s]: %w", t, err))
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

func (p *parser) parseToken(token string) error {
	var err error
	k, v := splitEquals(token)
	if v != "" {
		err = p.parseKeyVal(k, v)
	} else {
		err = p.parseStandalone(token)
	}
	return err
}

func (p *parser) parseKeyVal(k string, v string) error {
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

func (p *parser) parseStandalone(token string) error {
	switch token {
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
		return fmt.Errorf("unrecognized token")
	}
}

func splitEquals(token string) (string, string) {
	parts := strings.SplitN(token, "=", 2)
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], parts[1]
}
