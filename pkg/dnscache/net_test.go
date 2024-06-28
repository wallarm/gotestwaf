package dnscache

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"testing"
	"time"
)

func TestDialFunc(t *testing.T) {
	resolver := &Resolver{
		cache: map[string][]net.IP{
			"deeeet.com": {
				net.IP("127.0.0.1"),
				net.IP("127.0.0.2"),
				net.IP("127.0.0.3"),
			},
		},
	}

	cases := []struct {
		permF func(n int) []int
		dialF dialFunc
	}{
		{
			permF: func(n int) []int {
				return []int{0}
			},
			dialF: func(ctx context.Context, network, addr string) (net.Conn, error) {
				if got, want := addr, net.JoinHostPort(net.IP("127.0.0.1").String(), "443"); got != want {
					t.Fatalf("got addr %q, want %q", got, want)
				}
				return nil, nil
			},
		},
		{
			permF: func(n int) []int {
				return []int{1}
			},
			dialF: func(ctx context.Context, network, addr string) (net.Conn, error) {
				if got, want := addr, net.JoinHostPort(net.IP("127.0.0.2").String(), "443"); got != want {
					t.Fatalf("got addr %q, want %q", got, want)
				}
				return nil, nil
			},
		},
		{
			permF: func(n int) []int {
				return []int{2}
			},
			dialF: func(ctx context.Context, network, addr string) (net.Conn, error) {
				if got, want := addr, net.JoinHostPort(net.IP("127.0.0.3").String(), "443"); got != want {
					t.Fatalf("got addr %q, want %q", got, want)
				}
				return nil, nil
			},
		},
	}

	origFunc := randPerm
	defer func() {
		randPerm = origFunc
	}()

	for n, tc := range cases {
		t.Run(fmt.Sprintf("%d", n), func(t *testing.T) {
			randPerm = tc.permF
			if _, err := DialFunc(resolver, tc.dialF)(context.Background(), "tcp", "deeeet.com:443"); err != nil {
				t.Fatalf("err: %s", err)
			}
		})
	}

}

func TestDialFuncRand(t *testing.T) {
	rand.Seed(time.Now().UTC().UnixNano())
	defer func() {
		rand.Seed(1)
	}()

	resolver := &Resolver{
		cache: map[string][]net.IP{
			"deeeet.com": {
				net.IP("127.0.0.1"),
				net.IP("127.0.0.2"),
				net.IP("127.0.0.3"),
			},
		},
	}

	count := make(map[string]int)
	dialF := func(ctx context.Context, network, addr string) (net.Conn, error) {
		count[addr]++
		return nil, nil
	}

	for i := 0; i < 100; i++ {
		if _, err := DialFunc(resolver, dialF)(context.Background(), "tcp", "deeeet.com:443"); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	for _, c := range count {
		got := float32(c) / float32(100)
		if got < float32(0.2) {
			t.Fatalf("expect rate more than 0.2, got %f", got)
		}
	}
}

func TestDialFuncError1(t *testing.T) {
	resolver := testResolver(t)
	if _, err := DialFunc(resolver, nil)(context.Background(), "tcp", "deeeet.jp"); err == nil {
		t.Fatalf("expect to be failed") // need to specify port
	}
}

func TestDialFuncError2(t *testing.T) {
	originalFunc := lookupIP
	defer func() {
		lookupIP = originalFunc
	}()

	lookupIP = func(ctx context.Context, host string) ([]net.IP, error) {
		return nil, fmt.Errorf("err")
	}

	if _, err := DialFunc(testResolver(t), nil)(context.Background(), "tcp", "tcnksm.io:443"); err == nil {
		t.Fatalf("expect to be failed")
	}
}

func TestDialFuncError3(t *testing.T) {
	resolver := &Resolver{
		cache: map[string][]net.IP{
			"tcnksm.io": {
				net.IP("1.1.1.1"),
				net.IP("2.2.2.2"),
				net.IP("3.3.3.3"),
			},
		},
	}

	origFunc := randPerm
	randPerm = func(n int) []int {
		return []int{0, 1, 2}
	}
	defer func() {
		randPerm = origFunc
	}()

	want := errors.New("error1")
	dialF := func(ctx context.Context, network, addr string) (net.Conn, error) {
		if addr == net.JoinHostPort(net.IP("1.1.1.1").String(), "443") {
			return nil, want // first error should be returned
		}
		if addr == net.JoinHostPort(net.IP("2.2.2.2").String(), "443") {
			return nil, fmt.Errorf("error2")
		}
		if addr == net.JoinHostPort(net.IP("3.3.3.3").String(), "443") {
			return nil, fmt.Errorf("error3")
		}
		return nil, nil
	}

	_, got := DialFunc(resolver, dialF)(context.Background(), "tcp", "tcnksm.io:443")
	if got != want {
		t.Fatalf("got error %v, want %v", got, want)
	}
}
