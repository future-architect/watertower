module github.com/shibukawa/watertower

go 1.13

replace go.pyspa.org/brbundle => ../../../go.pyspa.org/brbundle

require (
	github.com/alecthomas/jsonschema v0.0.0-20190626084004-00dfc6288dec
	github.com/dsnet/compress v0.0.1 // indirect
	github.com/fatih/color v1.7.0
	github.com/gabriel-vasile/mimetype v0.3.18 // indirect
	github.com/hashicorp/golang-lru v0.5.3 // indirect
	github.com/ikawaha/kagome.ipadic v1.1.0
	github.com/kljensen/snowball v0.6.0
	github.com/rs/xid v1.2.1
	github.com/shibukawa/cloudcounter v0.0.2
	github.com/shibukawa/compints v0.1.0
	github.com/shibukawa/gocloudurls v1.0.3
	github.com/stretchr/testify v1.4.0
	go.pyspa.org/brbundle v1.1.2
	gocloud.dev v0.17.0
	golang.org/x/sync v0.0.0-20190423024810-112230192c58
	gopkg.in/alecthomas/kingpin.v2 v2.2.6
)
