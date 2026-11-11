package tcp_test

import (
	"net"
	"testing"
)

func TestTcp(t *testing.T) {
	in, err := net.Interfaces()
	if err != nil {
		t.Fatal(err)
	}
	for _, interfac := range in {
		addrs, err := interfac.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			t.Log(addr.String())
		}
	}

}
