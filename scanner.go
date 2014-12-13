package cmdlang

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strconv"
)

type Token int

/*
this is a  command
	,that continues to here

literals 'are in quotes' or these "quotes"

this 'is a command' that (some sub command for this param) calls sub commands

someaction
	,(depends on sub action)
	,(and this sub action)

# this is an eol commment

#(
	This is a block comment?

	)#

	#( something )#



*/

const (
	ILLEGAL Token = iota

	lit_beg
	TOK_IDENT
	lit_end

	special_beg
	TOK_WS
	TOK_EOF
	TOK_EOC
	TOK_BLOCK_START
	TOK_BLOCK_END
	special_end

	comment_beg
	TOK_COMMENT_BLOCK
	TOK_COMMENT_EOL
	comment_end
)

func (t Token) String() string {
	switch t {
	case ILLEGAL:
		return "ILLEGAL"
	case TOK_IDENT:
		return "IDENT"
	case TOK_WS:
		return "WHITESPACE"
	case TOK_EOF:
		return "EOF"
	case TOK_EOC:
		return "EOC"
	case TOK_COMMENT_BLOCK:
		return "COMMENT_BLOCK"
	case TOK_COMMENT_EOL:
		return "COMMENT_LINE"
	case TOK_BLOCK_START:
		return "BLOCK_START"
	case TOK_BLOCK_END:
		return "BLOCK_END"
	}

	return "unk"
}

var EOF_ARY = []byte{0}
var EOF = rune(0)
var CONTINUE = rune(',')

type posState struct {
	line int
	col  int
	pos  int
	last rune
}

type TokInfo struct {
	Lstart  int
	Cstart  int
	Pstart  int
	Lend    int
	Cend    int
	Pend    int
	Token   Token
	Literal []byte
}

func (t TokInfo) String() string {
	var buf bytes.Buffer

	buf.WriteString("Line: ")
	buf.WriteString(strconv.Itoa(t.Lstart))
	buf.WriteString(" - ")
	buf.WriteString(strconv.Itoa(t.Lend))
	buf.WriteString(" Col: ")
	buf.WriteString(strconv.Itoa(t.Cstart))
	buf.WriteString(" - ")
	buf.WriteString(strconv.Itoa(t.Cend))
	buf.WriteString(" Pos: ")
	buf.WriteString(strconv.Itoa(t.Pstart))
	buf.WriteString(" - ")
	buf.WriteString(strconv.Itoa(t.Pend))
	buf.WriteString(" ")
	buf.WriteString(t.Token.String())
	buf.WriteString(" ")
	buf.WriteString(fmt.Sprintf("%v", t.Literal))

	return buf.String()
}

type Scanner struct {
	r        *bufio.Reader
	ps       posState
	lineHist []posState
}

func NewScanner(r io.Reader) *Scanner {
	return &Scanner{r: bufio.NewReader(r)}
}

func (s *Scanner) read() rune {
	ch, _, err := s.r.ReadRune()
	if err != nil {
		return EOF
	}

	if ch == '\n' {
		s.lineHist = append(s.lineHist, s.ps)
		s.ps.line++
		s.ps.col = 0
	} else {
		s.ps.col++
	}
	s.ps.pos++
	s.ps.last = ch

	return ch
}

func (s *Scanner) unread() {
	if s.ps.last == '\n' {
		s.ps = s.lineHist[len(s.lineHist)-1]
		s.lineHist = s.lineHist[0 : len(s.lineHist)-1]
	} else {
		s.ps.col--
		s.ps.pos--
	}
	s.r.UnreadRune()
}

func (s *Scanner) Scan() TokInfo {
	ch := s.read()

	if ch == EOF {
		return TokInfo{Lstart: s.ps.line + 1,
			Cstart:  s.ps.col + 1,
			Pstart:  s.ps.pos,
			Lend:    s.ps.line + 1,
			Cend:    s.ps.col,
			Pend:    s.ps.pos,
			Token:   TOK_EOF,
			Literal: EOF_ARY}
	}

	if isWhite(ch) {
		s.unread()
		return s.scanWhitespace()
	}

	if isComment(ch) {
		s.unread()
		return s.scanComment()
	}

	if isBlock(ch) {
		s.unread()
		return s.scanBlock()
	}

	s.unread()
	return s.scanLiteral()
}

func (s *Scanner) scanBlock() TokInfo {
	var buf bytes.Buffer

	pbegin := s.ps

	ch := s.read()

	buf.WriteRune(ch)

	if ch == '(' {
		return TokInfo{Lstart: pbegin.line + 1,
			Cstart:  pbegin.col + 1,
			Pstart:  pbegin.pos,
			Lend:    s.ps.line + 1,
			Cend:    s.ps.col,
			Pend:    s.ps.pos,
			Token:   TOK_BLOCK_START,
			Literal: buf.Bytes()}
	}

	return TokInfo{Lstart: pbegin.line + 1,
		Cstart:  pbegin.col + 1,
		Pstart:  pbegin.pos,
		Lend:    s.ps.line + 1,
		Cend:    s.ps.col,
		Pend:    s.ps.pos,
		Token:   TOK_BLOCK_END,
		Literal: buf.Bytes()}
}

func (s *Scanner) scanLiteral() TokInfo {
	var buf bytes.Buffer

	pbegin := s.ps

	ch := s.read()

	literalDone := func(check rune) bool {
		return isWhite(check) || isBlock(check)
	}

	consumeLast := false

	switch ch {
	case '\'':
		consumeLast = true
		literalDone = func(check rune) bool {
			return check == '\''
		}
	case '"':
		consumeLast = true
		literalDone = func(check rune) bool {
			return check == '"'
		}
	default:
		buf.WriteRune(ch)
	}

	for ch = s.read(); ch != EOF && !literalDone(ch); ch = s.read() {
		buf.WriteRune(ch)
	}

	if !consumeLast {
		s.unread()
	}

	return TokInfo{Lstart: pbegin.line + 1,
		Cstart:  pbegin.col + 1,
		Pstart:  pbegin.pos,
		Lend:    s.ps.line + 1,
		Cend:    s.ps.col,
		Pend:    s.ps.pos,
		Token:   TOK_IDENT,
		Literal: buf.Bytes()}
}

func (s *Scanner) scanComment() TokInfo {
	var buf bytes.Buffer

	pbegin := s.ps

	buf.WriteRune(s.read())

	isBlock := false

	ch := s.read()
	if ch == '(' {
		isBlock = true
		buf.WriteRune(ch)
	}

	last := rune(0)

	for ch = s.read(); ch != EOF; ch = s.read() {
		if ch == '\n' && !isBlock {
			buf.WriteRune(ch)
			return TokInfo{Lstart: pbegin.line + 1,
				Cstart:  pbegin.col + 1,
				Pstart:  pbegin.pos,
				Lend:    s.ps.line + 1,
				Cend:    s.ps.col,
				Pend:    s.ps.pos,
				Token:   TOK_COMMENT_EOL,
				Literal: buf.Bytes()}
		}
		if ch == '#' && last == ')' && isBlock {
			buf.WriteRune(ch)
			return TokInfo{Lstart: pbegin.line + 1,
				Cstart:  pbegin.col + 1,
				Pstart:  pbegin.pos,
				Lend:    s.ps.line + 1,
				Cend:    s.ps.col,
				Pend:    s.ps.pos,
				Token:   TOK_COMMENT_BLOCK,
				Literal: buf.Bytes()}
		}
		buf.WriteRune(ch)
		last = ch
	}

	return TokInfo{Lstart: pbegin.line + 1,
		Cstart:  pbegin.col + 1,
		Pstart:  pbegin.pos,
		Lend:    s.ps.line + 1,
		Cend:    s.ps.col,
		Pend:    s.ps.pos,
		Token:   TOK_COMMENT_EOL,
		Literal: buf.Bytes()}
}

// scanWhitespace eat all whitespace, \n in is special its only whitespace if the next non ws char is a CONTINUE char
func (s *Scanner) scanWhitespace() TokInfo {
	var buf bytes.Buffer

	pbegin := s.ps

	crPOS := -1

	for ch := s.read(); ch != EOF; ch = s.read() {
		if isWhite(ch) {
			buf.WriteRune(ch)
			if ch == '\n' {
				crPOS = s.ps.pos
			}
			continue
		}

		if !isWhite(ch) && crPOS > -1 && ch == CONTINUE {
			buf.WriteRune(ch)
			crPOS = -1
			continue
		}

		if crPOS > -1 {
			back := s.ps.pos - crPOS
			// rollback to the carrage return this is actually a command delimiter
			for i := 0; i < back; i++ {
				s.unread()
			}
			return TokInfo{Lstart: pbegin.line + 1,
				Cstart:  pbegin.col + 1,
				Pstart:  pbegin.pos,
				Lend:    s.ps.line + 1,
				Cend:    s.ps.col,
				Pend:    s.ps.pos,
				Token:   TOK_EOC,
				Literal: buf.Bytes()[0 : buf.Len()-back]}
		}

		s.unread()
		break // normal char here
	}

	return TokInfo{Lstart: pbegin.line + 1,
		Cstart:  pbegin.col + 1,
		Pstart:  pbegin.pos,
		Lend:    s.ps.line + 1,
		Cend:    s.ps.col,
		Pend:    s.ps.pos,
		Token:   TOK_WS,
		Literal: buf.Bytes()}
}

func isWhite(ch rune) bool {
	return ch == ' ' || ch == '\t' || ch == '\n'
}

func isComment(ch rune) bool {
	return ch == '#'
}

func isBlock(ch rune) bool {
	return ch == '(' || ch == ')'
}
