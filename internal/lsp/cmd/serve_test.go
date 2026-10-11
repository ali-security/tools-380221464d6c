// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"golang.org/x/tools/internal/jsonrpc2"
	errors "golang.org/x/xerrors"
)

func TestListenParsing(t *testing.T) {
	tests := []struct {
		input, wantNetwork, wantAddr string
	}{
		{"127.0.0.1:0", "tcp", "127.0.0.1:0"},
		{"unix;/tmp/sock", "unix", "/tmp/sock"},
		{"auto", "auto", ""},
		{"auto;foo", "auto", "foo"},
	}

	for _, test := range tests {
		gotNetwork, gotAddr := parseAddr(test.input)
		if gotNetwork != test.wantNetwork {
			t.Errorf("network = %q, want %q", gotNetwork, test.wantNetwork)
		}
		if gotAddr != test.wantAddr {
			t.Errorf("addr = %q, want %q", gotAddr, test.wantAddr)
		}
	}
}

// TestServeRejectsImplicitAllInterfaces checks that -listen rejects addresses
// with an empty host (e.g. ":12345" or "tcp;"), which would implicitly bind all
// network interfaces.
func TestServeRejectsImplicitAllInterfaces(t *testing.T) {
	for _, address := range []string{":0", ":37374", "tcp;:0", "tcp;"} {
		t.Run(address, func(t *testing.T) {
			// The deadline only matters if the address is (wrongly) accepted:
			// the server would then listen until the context is done.
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			s := &Serve{Address: address, app: New("gopls", "", nil, nil)}
			err := s.Run(ctx)
			if err == nil || !strings.Contains(err.Error(), "implicitly binds all network interfaces") {
				t.Fatalf("Serve{Address: %q}.Run() = %v, want implicit all-interfaces error", address, err)
			}
		})
	}
}

// TestServeAcceptsExplicitHost checks that -listen with an explicit host
// still starts a daemon.
func TestServeAcceptsExplicitHost(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	s := &Serve{
		Address:     "127.0.0.1:0",
		IdleTimeout: 10 * time.Millisecond,
		app:         New("gopls", "", nil, nil),
	}
	if err := s.Run(ctx); !errors.Is(err, jsonrpc2.ErrIdleTimeout) {
		t.Fatalf("Serve{Address: %q}.Run() = %v, want %v", s.Address, err, jsonrpc2.ErrIdleTimeout)
	}
}

// TestServeHasNoPortFlag checks that the -port flag, which bound all network
// interfaces, has been removed.
func TestServeHasNoPortFlag(t *testing.T) {
	typ := reflect.TypeOf(Serve{})
	for i := 0; i < typ.NumField(); i++ {
		if name, ok := typ.Field(i).Tag.Lookup("flag"); ok && name == "port" {
			t.Errorf("Serve.%s defines the -port flag, which binds all network interfaces", typ.Field(i).Name)
		}
	}
}
