package base62

import "testing"

func TestEncode(t *testing.T) {
	cases := []struct {
		in   uint64
		want string
	}{
		{0, "0"},
		{1, "1"},
		{61, "z"},
		{62, "10"},
		{3843, "zz"},
		{1000000, "4C92"},
	}

	for _, c := range cases {
		if got := Encode(c.in); got != c.want {
			t.Fatalf("Encode(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}
