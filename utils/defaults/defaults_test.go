package defaults

import "testing"

func TestGet(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		default_ any
		want     any
	}{
		{
			name:     "string: empty with default",
			input:    "",
			default_: "default",
			want:     "default",
		},
		{
			name:     "string: non-empty with default",
			input:    "value",
			default_: "default",
			want:     "value",
		},
		{
			name:     "int: zero with default",
			input:    0,
			default_: 42,
			want:     42,
		},
		{
			name:     "int: non-zero with default",
			input:    10,
			default_: 42,
			want:     10,
		},
		{
			name:     "bool: false with default",
			input:    false,
			default_: true,
			want:     true,
		},
		{
			name:     "bool: true with default",
			input:    true,
			default_: false,
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			switch v := tt.input.(type) {
			case string:
				got := Get(v, tt.default_.(string))
				if got != tt.want {
					t.Errorf("Get() = %v, want %v", got, tt.want)
				}
			case int:
				got := Get(v, tt.default_.(int))
				if got != tt.want {
					t.Errorf("Get() = %v, want %v", got, tt.want)
				}
			case bool:
				got := Get(v, tt.default_.(bool))
				if got != tt.want {
					t.Errorf("Get() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}
