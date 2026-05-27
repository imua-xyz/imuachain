package types

import "testing"

func TestValidSOLAddressWithPrefix(t *testing.T) {
	tcs := []struct {
		name string
		addr string
		ok   bool
	}{
		{
			name: "valid 32-byte hex with 0x prefix",
			addr: "0x" + "11fbcd8f6beb886acf61dbffa61c8e24ec721464000000000000000000000000",
			ok:   true,
		},
		{
			name: "invalid prefix",
			addr: "11fbcd8f6beb886acf61dbffa61c8e24ec721464000000000000000000000000",
			ok:   false,
		},
		{
			name: "wrong length (20-byte address)",
			addr: "0x" + "11fbcd8f6beb886acf61dbffa61c8e24ec721464",
			ok:   false,
		},
		{
			name: "non-hex characters",
			addr: "0x" + "zzfbcd8f6beb886acf61dbffa61c8e24ec721464000000000000000000000000",
			ok:   false,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			if got := ValidSOLAddressWithPrefix(tc.addr); got != tc.ok {
				t.Fatalf("ValidSOLAddressWithPrefix(%q)=%v, want %v", tc.addr, got, tc.ok)
			}
		})
	}
}

func TestNormalizeBSCAddress(t *testing.T) {
	tcs := []struct {
		name string
		addr string
		want string
		ok   bool
	}{
		{
			name: "valid 20-byte hex address with 0x prefix",
			addr: "0x" + "11fbcd8f6beb886acf61dbffa61c8e24ec721464",
			want: "0x11fbcd8f6beb886acf61dbffa61c8e24ec721464",
			ok:   true,
		},
		{
			name: "zero address is invalid",
			addr: "0x0000000000000000000000000000000000000000",
			want: "",
			ok:   false,
		},
		{
			name: "no 0x prefix is allowed (normalized)",
			addr: "11fbcd8f6beb886acf61dbffa61c8e24ec721464",
			want: "0x11fbcd8f6beb886acf61dbffa61c8e24ec721464",
			ok:   true,
		},
		{
			name: "left-padded/32-byte hex is allowed (take last 20 bytes)",
			addr: "0x" + "11fbcd8f6beb886acf61dbffa61c8e24ec721464000000000000000000000000",
			want: "0xa61c8e24ec721464000000000000000000000000",
			ok:   true,
		},
		{
			name: "non-hex characters",
			addr: "0x" + "zzfbcd8f6beb886acf61dbffa61c8e24ec721464",
			want: "",
			ok:   false,
		},
		{
			name: "left-padded/overlong string keeps last 20 bytes",
			addr: "0x00000000000000000000000000000000000000000000000011fbcd8f6beb886acf61dbffa61c8e24ec721464",
			want: "0x11fbcd8f6beb886acf61dbffa61c8e24ec721464",
			ok:   true,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			got, err := NormalizeBSCAddress(tc.addr)
			if tc.ok {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if got != tc.want {
					t.Fatalf("NormalizeBSCAddress(%q)=%q, want %q", tc.addr, got, tc.want)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error, got addr=%q", got)
			}
		})
	}
}
