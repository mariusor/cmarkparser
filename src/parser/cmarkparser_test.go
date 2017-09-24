package parser

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	m "markdown"
	"os"
	"path/filepath"
	"testing"
)

var stopOnFailure = false

type testPair struct {
	text     string
	expected bool
	doc      m.Document
}

type tests map[string]testPair

var emptyDoc = m.NewDocument()

func newNode(t m.NodeType, s string, c m.Nodes) m.Node {
	return m.Node{Type: t, Content: []byte(s), Children: c}
}

func newDoc(n m.Nodes) m.Document {
	return m.Document{Type: m.Doc, Children: n}
}

var someTests = tests{
	// empty doc
	"empty": {
		"",
		false,
		emptyDoc,
	},
	"line": {
		"some text",
		true,
		newDoc(m.Nodes{newNode(m.Par, "some text", nil)}),
	},
	// null char
	"null_char": {
		"\x00",
		true,
		newDoc(m.Nodes{newNode(m.Par, "\ufffd", nil)}),
	},
	// spaces
	"space#1": {
		"\uc2a0",
		true,
		newDoc(m.Nodes{newNode(m.Par, "\uc2a0", nil)}),
	},
	"space#2": {
		"\u2000",
		true,
		newDoc(m.Nodes{newNode(m.Par, "\u2000", nil)}),
	},
	"space#3": {
		"\u2001",
		true,
		newDoc(m.Nodes{newNode(m.Par, "\u2001", nil)}),
	},
	// links, for now treated as paragraphs
	"link#1": {
		"[ana](httpslittrme)",
		true,
		newDoc(m.Nodes{newNode(m.Par, "[ana](httpslittrme)", nil)}),
	},
	"link#2": {
		"[ana](https://littr.me)\n",
		true,
		newDoc(m.Nodes{newNode(m.Par, "[ana](https://littr.me)\n", nil)}),
	},
	"link_after_text": {
		"some text before [test 123](https://littr.me)\n",
		true,
		newDoc(m.Nodes{newNode(m.Par, "some text before [test 123](https://littr.me)\n", nil)}),
	},
	"link_before_text": {
		"[test 123](https://littr.me) some text after\n",
		true,
		newDoc(m.Nodes{newNode(m.Par, "[test 123](https://littr.me) some text after\n", nil)}),
	},
	"link_inside_text": {
		"some text before [test 123](https://littr.me) some text after\n",
		true,
		newDoc(m.Nodes{newNode(m.Par, "some text before [test 123](https://littr.me) some text after\n", nil)}),
	},
	// utf8 only characters
	"utf8#1": {
		"𐍈ᏚᎢᎵᎬᎢᎬᏒăîțș",
		true,
		newDoc(m.Nodes{newNode(m.Par, "𐍈ᏚᎢᎵᎬᎢᎬᏒăîțș", nil)}),
	},
	// thematic breaks
	"break#1:-": {
		" ---\n",
		true,
		newDoc(m.Nodes{newNode(m.TBreak, "-", nil)}),
	},
	"break#2:*": {
		"  ***\n",
		true,
		newDoc(m.Nodes{newNode(m.TBreak, "*", nil)}),
	},
	"break#3:*": {
		"  * * * *\n",
		true,
		newDoc(m.Nodes{newNode(m.TBreak, "*", nil)}),
	},
	"break#4:-": {
		"   ___\r",
		true,
		newDoc(m.Nodes{newNode(m.TBreak, "_", nil)}),
	},
	// misleading thematic break
	"not_a_break": {
		"   _*-*__",
		true,
		newDoc(m.Nodes{newNode(m.Par, "   _*-*__", nil)}),
	},
	// headings
	"h1": {
		" # ana are mere\n",
		true,
		newDoc(m.Nodes{newNode(m.H1, "ana are mere", nil)}),
	},
	"h2": {
		"## ana are mere\n",
		true,
		newDoc(m.Nodes{newNode(m.H2, "ana are mere", nil)}),
	},
	"h3": {
		"  ### ana are mere\n",
		true,
		newDoc(m.Nodes{newNode(m.H3, "ana are mere", nil)}),
	},
	"h4": {
		"#### ana are mere\n",
		true,
		newDoc(m.Nodes{newNode(m.H4, "ana are mere", nil)}),
	},
	"h5": {
		"   #####  ana-are-mere\n",
		true,
		newDoc(m.Nodes{newNode(m.H5, "ana-are-mere", nil)}),
	},
	"h6": {
		" ###### ana-are-mere\n",
		true,
		newDoc(m.Nodes{newNode(m.H6, "ana-are-mere", nil)}),
	},
}

func TestParse(t *testing.T) {
	var err error
	var doc m.Document
	f := t.Errorf
	if stopOnFailure {
		f = t.Fatalf
	}
	for k, curTest := range someTests {
		t.Logf("Testing: %s", k)
		doc, err = Parse([]byte(curTest.text))

		if err != nil && curTest.expected {
			f("Parse failed and success was expected %s\n %s", err, curTest.text)
		}
		j_t_doc, _ := json.Marshal(curTest.doc)
		j_doc, _ := json.Marshal(doc)
		if string(j_t_doc) != string(j_doc) {
			f("\n%s_________________\n%s", doc, curTest.doc)
		}
	}
}

func load_files(ext string) ([]string, error) {
	var files []string

	dir := "./tests"
	err := filepath.Walk(dir, func(path string, f os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !f.IsDir() && path[len(path)-len(ext):] == ext {
			files = append(files, path)
		}

		return nil
	})

	if err != nil {
		return nil, errors.New(fmt.Sprintf("Could not load files %s/*%s", dir, ext))
	}

	return files, nil
}

func get_file_contents(path string) []byte {
	f, _ := os.Open(path)

	data := make([]byte, 512)
	io.ReadFull(f, data)
	data = bytes.Trim(data, "\x00")

	return data
}

type testDoc struct {
	Children []testNode
}

type testNode struct {
	Type     string
	Content  []string
	Children []testNode
}

func (t *testDoc) Document() m.Document {
	d := m.Document{}
	for _, v := range t.Children {
		d.Children = append(d.Children, *v.Node())
	}
	return d
}

func (t *testNode) Node() *m.Node {
	n := m.Node{}
	for _, v := range t.Children {
		n.Children = append(n.Children, *v.Node())
	}
	for k, s := range t.Content {
		if k > 0 {
			n.Content = append(n.Content, byte('\n'))
		}
		for _, b := range s {
			n.Content = append(n.Content, byte(b))
		}
	}
	n.Type = m.GetNodeType(t.Type)
	return &n
}

func TestWithFiles(t *testing.T) {
	var tests []string
	var err error

	tests, err = load_files(".md")
	f := t.Errorf
	if stopOnFailure {
		f = t.Fatalf
	}
	for _, path := range tests {
		var doc m.Document
		var t_doc testDoc
		data := get_file_contents(path)
		t.Logf("Testing: %s", path)
		res_path := fmt.Sprintf("%s.json", path[:len(path)-3])
		json.Unmarshal(get_file_contents(res_path), &t_doc)

		res_doc := t_doc.Document()
		doc, err = Parse(data)

		if err != nil {
			f("%s", err)
		}
		j_res_doc, _ := json.Marshal(res_doc)
		j_doc, _ := json.Marshal(doc)
		if string(j_res_doc) != string(j_doc) {
			f("\n%s_________________\n%s", doc, res_doc)
		}
	}
}

func TestMain(m *testing.M) {
	f := func(slice []string, s string) bool {
		for _, el := range slice {
			if s == el {
				return true
			}
		}
		return false
	}
	if f(os.Args, "stop-on-fail") {
		stopOnFailure = true
	}
	if f(os.Args, "quiet") {
		log.SetOutput(ioutil.Discard)
	}
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	os.Exit(m.Run())
}
