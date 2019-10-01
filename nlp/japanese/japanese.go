package japanese

import (
	"sync"

	"github.com/ikawaha/kagome.ipadic/tokenizer"
	"github.com/shibukawa/watertower/nlp"
)
const Language = "ja"

func init() {
	// https://github.com/stopwords-iso/stopwords-ja
	stopWordsSrc := []string{
		"あそこ","あっ","あの","あのかた","あの人","あり","あります","ある","あれ","い","いう","います","いる","う","うち",
		"え","お","および","おり","おります","か","かつて","から","が","き","ここ","こちら","こと","この","これ","これら",
		"さ","さらに","し","しかし","する","ず","せ","せる","そこ","そして","その","その他","その後","それ","それぞれ",
		"それで","た","ただし","たち","ため","たり","だ","だっ","だれ","つ","て","で","でき","できる","です","では","でも",
		"と","という","といった","とき","ところ","として","とともに","とも","と共に","どこ","どの","な","ない","なお",
		"なかっ","ながら","なく","なっ","など","なに","なら","なり","なる","なん","に","において","における","について",
		"にて","によって","により","による","に対して","に対する","に関する","の","ので","のみ","は","ば","へ","ほか",
		"ほとんど","ほど","ます","また","または","まで","も","もの","ものの","や","よう","より","ら","られ","られる","れ",
		"れる","を","ん","何","及び","彼","彼女","我々","特に","私","私達","貴方","貴方方",
	}
	stopWords := make(map[string]bool)
	for _, stopWord := range stopWordsSrc {
		stopWords[stopWord] = true
	}
	nlp.RegisterTokenizer(Language, japaneseSplitter, japaneseStemmer, stopWords)
}

var once sync.Once
var kagomeTokenizer tokenizer.Tokenizer

func japaneseSplitter(content string) []string {
	once.Do(func() {
		var dic = tokenizer.SysDicIPASimple()
		//fmt.Println(dic)
		kagomeTokenizer = tokenizer.NewWithDic(dic)
	})
	var result []string
	tokens := kagomeTokenizer.Analyze(content, tokenizer.Search)
	for _, token := range tokens {
		if token.Class == tokenizer.DUMMY {
			continue
		}
		features := token.Features()
		//fmt.Printf("%s\t%v\n", token.Surface, features)
		if features[0] == "助詞" || features[0] == "記号" {
			continue
		}
		// use normalized word?
		result = append(result, token.Surface)
	}
	return result
}

func japaneseStemmer(content string) string {
	return content
}

