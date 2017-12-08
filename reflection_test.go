package wsrpc

import (
	"io"
	"testing"
)

type NTest struct{ F int }
type ReqTest struct{ D string }
type RespTest struct{ B float64 }

type SProt struct{}

func (p SProt) OnConnect(io.Closer, *RPCNotifier) {}
func (p SProt) OnDisconnect(error)                {}

type SProtWithErrNotify struct {
	SProt
	Notifications chan string
}

type SProtWithErrNotify2 struct {
	SProt
	Notifications struct {
		NTest
	}
}

type SProtWithErrMethod struct {
	SProt
	Notifications struct {
		*NTest
	}
}

func (p *SProtWithErrMethod) InvalidMethod() {}

type SProtWithErrMethod2 struct {
	SProt
	Notifications struct {
		*NTest
	}
}

func (p *SProtWithErrMethod2) InvalidMethod(r ReqTest) {}

type SProtWithErrMethod3 struct {
	SProt
	Notifications struct {
		*NTest
	}
}

func (p *SProtWithErrMethod3) InvalidMethod(r *int) {}

type SProtWithErrMethod4 struct {
	SProt
	Notifications struct {
		*NTest
	}
}

func (p *SProtWithErrMethod4) InvalidMethod(r *ReqTest) {}

type SProtWithErrMethod5 struct {
	SProt
	Notifications struct {
		*NTest
	}
}

func (p *SProtWithErrMethod5) InvalidMethod(r *ReqTest) (RespTest, int) {
	return RespTest{}, 55
}

type SProtWithErrMethod6 struct {
	SProt
	Notifications struct {
		*NTest
	}
}

func (p *SProtWithErrMethod6) InvalidMethod(r *ReqTest) (*int, int) {
	t := 44
	return &t, 55
}

type SProtWithErrMethod7 struct {
	SProt
	Notifications struct {
		*NTest
	}
}

func (p *SProtWithErrMethod7) InvalidMethod(r *ReqTest) (*RespTest, int) {
	return &RespTest{66}, 55
}

type SProtWithValidMethods struct {
	SProt
	Notifications struct {
		*NTest
	}
}

func (p *SProtWithValidMethods) somePrivate()                               {}
func (p *SProtWithValidMethods) FirstMethod(r *ReqTest) (*RespTest, error)  { return &RespTest{77}, nil }
func (p *SProtWithValidMethods) SecondMethod(r *ReqTest) (*RespTest, error) { return &RespTest{66}, nil }

func TestSessionProtocolReflection(t *testing.T) {
	// pass nil
	pd, err := parseSessionProtocol(nil)
	if pd != nil {
		t.Fatalf("%v should be nil", pd)
	}
	if err.Error() != "pointer to protocol instance expected" {
		t.Fatalf("unexpected error: %s", err)
	}

	// pass non-pointer
	pd, err = parseSessionProtocol(SProt{})
	if err.Error() != "pointer to protocol instance expected" {
		t.Fatalf("unexpected error: %s", err)
	}

	// no notifications
	pd, err = parseSessionProtocol(&SProt{})
	if err.Error() != "no Notifications declaration found in session protocol" {
		t.Fatalf("unexpected error: %s", err)
	}

	pd, err = parseSessionProtocol(&SProtWithErrNotify{})
	if err.Error() != "Notifications must be declared as a struct" {
		t.Fatalf("unexpected error: %s", err)
	}

	pd, err = parseSessionProtocol(&SProtWithErrNotify2{})
	if err.Error() != "notification NTest must be a pointer" {
		t.Fatalf("unexpected error: %s", err)
	}

	pd, err = parseSessionProtocol(&SProtWithErrMethod{})
	if err.Error() != "one input structure expected in method InvalidMethod" {
		t.Fatalf("unexpected error: %s", err)
	}

	pd, err = parseSessionProtocol(&SProtWithErrMethod2{})
	if err.Error() != "input must be a pointer in method InvalidMethod" {
		t.Fatalf("unexpected error: %s", err)
	}

	pd, err = parseSessionProtocol(&SProtWithErrMethod3{})
	if err.Error() != "input must be a pointer to struct in method InvalidMethod" {
		t.Fatalf("unexpected error: %s", err)
	}

	pd, err = parseSessionProtocol(&SProtWithErrMethod4{})
	if err.Error() != "expected response and error as output in method InvalidMethod" {
		t.Fatalf("unexpected error: %s", err)
	}

	pd, err = parseSessionProtocol(&SProtWithErrMethod5{})
	if err.Error() != "output must be a pointer in method InvalidMethod" {
		t.Fatalf("unexpected error: %s", err)
	}

	pd, err = parseSessionProtocol(&SProtWithErrMethod6{})
	if err.Error() != "output must be a pointer to struct in method InvalidMethod" {
		t.Fatalf("unexpected error: %s", err)
	}

	pd, err = parseSessionProtocol(&SProtWithErrMethod7{})
	if err.Error() != "method must return error type in method InvalidMethod" {
		t.Fatalf("unexpected error: %s", err)
	}

	pd, err = parseSessionProtocol(&SProtWithValidMethods{})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if len(pd.methods) != 2 {
		t.Fatal("expected 2 parsed methods")
	}
	if _, ok := pd.methods["FirstMethod"]; !ok {
		t.Fatal("FirstMethod not found")
	}
	if _, ok := pd.methods["SecondMethod"]; !ok {
		t.Fatal("SecondMethod not found")
	}

	if len(pd.notifications) != 1 {
		t.Fatal("expected 1 parsed notification")
	}
	if _, ok := pd.notifications["NTest"]; !ok {
		t.Fatal("NTest notification not found")
	}
}
