package wsrpc

import (
	"fmt"
	"reflect"
)

type methodDetails struct {
	funcVal reflect.Value
	inType  reflect.Type
	outType reflect.Type
}

func parseSessionProtocol(p SessionProtocol) (map[string]methodDetails, error) {
	methods := make(map[string]methodDetails)

	errorInterface := reflect.TypeOf((*error)(nil)).Elem()
	pType := reflect.TypeOf(p)
	if pType.Kind() != reflect.Ptr {
		return nil, fmt.Errorf("pointer to protocol instance expected")
	}
	for i := 0; i < pType.NumMethod(); i++ {
		m := pType.Method(i)
		switch m.Name {
		case "OnConnect", "OnDisconnect":
			continue
		default:
		}
		//fmt.Printf("method #%d: name=%s, type=%s, func=%s\n", i, m.Name, m.Type, m.Func)

		// check inputs
		if m.Type.NumIn() != 2 {
			return nil, fmt.Errorf("one input structure expected in method %s", m.Name)
		}

		in := m.Type.In(1)
		if in.Kind() != reflect.Ptr {
			return nil, fmt.Errorf("input must be a pointer in method %s", m.Name)
		}
		inT := in.Elem()
		if inT.Kind() != reflect.Struct {
			return nil, fmt.Errorf("input must be a pointer to struct in method %s", m.Name)
		}

		// check outputs
		if m.Type.NumOut() != 2 {
			return nil, fmt.Errorf("expected response and error as output in method %s", m.Name)
		}
		out := m.Type.Out(0)
		if out.Kind() != reflect.Ptr {
			return nil, fmt.Errorf("output must be a pointer in method %s", m.Name)
		}
		outT := out.Elem()
		if outT.Kind() != reflect.Struct {
			return nil, fmt.Errorf("output must be a pointer to struct in method %s", m.Name)
		}
		out = m.Type.Out(1)
		if !out.Implements(errorInterface) {
			return nil, fmt.Errorf("method must return error type in method %s", m.Name)
		}

		methods[m.Name] = methodDetails{m.Func, inT, outT}
	}
	return methods, nil
}
