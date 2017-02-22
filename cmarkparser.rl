// -*-go-*-
//
// Commonmark link parser
// Copyright (c) 2017 Marius Orcisk <marius@habarnam.ro>
// MIT License
// 

package cmarkparser

import(
    "fmt"
    "errors"
)

func Parse (data []byte) (Document, error) {
    return parse(data)
}

%% machine parser;
%% include character_definitions "characters.rl";
%% include blocks "blocks.rl";

%% write data;

func arr_splice(dst []byte, src []byte, pos int) []byte {
    var ret = make([]byte, 0)
	for _, a := range dst[:pos] {
		ret = append (ret, a)
	}
	for _, b := range src {
		ret = append (ret, b)
	}
	for _, c := range dst[pos+1:]{
		ret = append (ret, c)
	}
    return ret
}

func parse(data []byte) (Document, error) {
    cs, p, pe := 0, 0, len(data)
    //ts, te, act := 0, 0, 0
    eof := len(data)

    var doc Document = Document{Children: []Node{Node{}}}
    if pe == 0 {
        return doc, errors.New("Empty document")
    }

    var node Node
    var heading_level uint;
    var nodes []Node;
    //fmt.Printf("Incoming str: %#v - len %d\n", data, len(data))

    var mark int
    var thematic_break_symbol byte;

    %%{
        action emit_eof {
            doc.Children = nodes
        }

        action emit_add_block {
            if !node.Empty() {
                nodes = append(nodes, node)
                mark = -1
            }
        }
        single_line_doc = (line_char | punctuation)+ >mark %emit_add_line;
        document = ((block %emit_add_block)* | (single_line_doc %emit_add_block));


        #main := |* 
        #    block => emit_add_block;
        #    line => emit_new_line;
        #*|;
        
        
        main := document %eof emit_eof;
 
        write init;
        write exec;
    }%%

    //fmt.Printf("last node: %v\n", node)
    //fmt.Printf("last mark: %v\n", mark)
    //fmt.Printf("last level: %v\n", header_level)

    //fmt.Printf("%d", ts)
    fmt.Printf("")

    return doc, nil
}
