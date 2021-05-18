package bigram

import (
	"reflect"
	"testing"
)

func Test_bigramSplitter(t *testing.T) {
	type args struct {
		content string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "standard",
			args: args{
				content: "hello",
			},
			want: []string{"he", "el", "ll", "lo"},
		},
		{
			name: "empty",
			args: args{
				content: "",
			},
			want: []string{},
		},
		{
			name: "one char string",
			args: args{
				content: "a",
			},
			want: []string{},
		},
		{
			name: "emoji",
			args: args{
				content: "🐸🐍",
			},
			want: []string{"🐸🐍"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := bigramSplitter(tt.args.content); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("bigramSplitter() = %v, want %v", got, tt.want)
			}
		})
	}
}
