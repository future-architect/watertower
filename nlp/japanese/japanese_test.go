package japanese

import (
	"reflect"
	"testing"
)

func Test_japaneseSplitter(t *testing.T) {
	type args struct {
		content string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "empty string",
			args: args{
				content: "",
			},
			want: nil,
		},
		{
			name: "すもももももももものうち",
			args: args{
				content: "すもももももももものうち",
			},
			want: []string{"すもも","もも","もも","うち"},
		},
		{
			name: "人魚は、南の方の海にばかり棲んでいるのではありません。",
			args: args{
				content: "人魚は、南の方の海にばかり棲んでいるのではありません。",
			},
			want: []string{"人魚", "南", "方", "海", "棲ん", "いる", "の", "で", "あり","ませ", "ん"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := japaneseSplitter(tt.args.content); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("japaneseSplitter() = %v, want %v", got, tt.want)
			}
		})
	}
}

