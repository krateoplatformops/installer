package expand

import "testing"

func TestExpand(t *testing.T) {
	table := []struct {
		in   string
		want string
	}{
		{
			in:   "https://$HOST:$PORT",
			want: "https://domain.com:8080",
		},
		{
			in:   "https://$HOST_2:$PORT",
			want: "https://$HOST_2:8080",
		},
	}

	env := map[string]string{
		"HOST": "domain.com",
		"PORT": "8080",
	}

	subst := func(k string) string {
		if v, ok := env[k]; ok {
			return v
		}
		return "$" + k
	}

	for i, tc := range table {
		got := Expand(tc.in, "", subst)
		if got != tc.want {
			t.Fatalf("[tc: %d] - got: %v, expected: %v", i, got, tc.want)
		}
	}
}
