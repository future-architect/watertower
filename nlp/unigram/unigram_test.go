package unigram

import (
	"reflect"
	"testing"
)

func Test_unigramSplitter(t *testing.T) {
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
			want: []string{"h", "e", "l", "l", "o"},
		},
		{
			name: "empty",
			args: args{
				content: "",
			},
			want: []string{},
		},
		{
			name: "emoji",
			args: args{
				content: "🐸🐍",
			},
			want: []string{"🐸", "🐍"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := unigramSplitter(tt.args.content); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("unigramSplitter() = %v, want %v", got, tt.want)
			}
		})
	}
}
