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

type protocolDetails struct {
	methods       map[string]methodDetails
	notifications map[string]reflect.Type
}

func (pd *protocolDetails) checkNotifType(n interface{}) bool {
	nt := reflect.TypeOf(n)
	if nt.Kind() == reflect.Ptr {
		nt = nt.Elem()
	}

	vt, ok := pd.notifications[nt.Name()]
	if !ok {
		return false
	}
	return vt == nt
}

func parseSessionProtocol(p SessionProtocol) (*protocolDetails, error) {
	errorInterface := reflect.TypeOf((*error)(nil)).Elem()
	pType := reflect.TypeOf(p)
	if pType.Kind() != reflect.Ptr {
		return nil, fmt.Errorf("pointer to protocol instance expected")
	}

	ret := &protocolDetails{
		methods: make(map[string]methodDetails),
	}
	ndescr, err := parseNotifications(pType.Elem())
	if err != nil {
		return nil, err
	}
	ret.notifications = ndescr

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

		ret.methods[m.Name] = methodDetails{m.Func, inT, outT}
	}
	return ret, nil
}

func parseNotifications(pt reflect.Type) (map[string]reflect.Type, error) {
	field, ok := pt.FieldByName("Notifications")
	if !ok {
		return nil, fmt.Errorf("no Notifications declaration found in session protocol")
	}

	ns := field.Type
	if ns.Kind() != reflect.Struct {
		return nil, fmt.Errorf("Notifications must be declared as a struct")
	}

	ret := make(map[string]reflect.Type)
	for i := 0; i < ns.NumField(); i++ {
		f := ns.Field(i)
		if f.Type.Kind() != reflect.Ptr {
			return nil, fmt.Errorf("notification %s must be a pointer", f.Name)
		}
		n := f.Type.Elem()
		ret[n.Name()] = n
	}

	return ret, nil

}
