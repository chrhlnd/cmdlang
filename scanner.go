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

	resume func() TokInfo
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
	if s.resume != nil {
		call := s.resume
		s.resume = nil
		return call()
	}

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
	escaped := false
	useEsc := false

	switch ch {
	case '\'':
		consumeLast = true
		useEsc = true
		literalDone = func(check rune) bool {
			return check == '\''
		}
	case '"':
		consumeLast = true
		useEsc = true
		literalDone = func(check rune) bool {
			return check == '"'
		}
	default:
		buf.WriteRune(ch)
	}

	if useEsc {
		for ch = s.read(); ch != EOF && (!literalDone(ch) || escaped); ch = s.read() {
			switch ch {
			case '\\':
				if escaped {
					buf.WriteRune(ch)
					escaped = false
				} else {
					escaped = true
				}
			default:
				escaped = false
				buf.WriteRune(ch)
			}
		}
	} else {
		for ch = s.read(); ch != EOF && !literalDone(ch); ch = s.read() {
			buf.WriteRune(ch)
		}
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

	for ; ch != EOF; ch = s.read() {
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
	pbegin := s.ps

	crPOS := -1

	buf := bytes.Buffer{}

	tokens := make([]TokInfo, 0)

	crToken := TokInfo{}

	for ch := s.read(); ch != EOF; ch = s.read() {
		if isWhite(ch) { // eat whitespace
			buf.WriteRune(ch)
			if ch == '\n' {
				crToken.Cstart = pbegin.col + 1
				crToken.Pstart = pbegin.pos
				crToken.Lend = s.ps.line + 1
				crToken.Cend = s.ps.col
				crToken.Pend = s.ps.pos
				crToken.Token = TOK_EOC
				crToken.Literal = buf.Bytes()[0:]

				crPOS = s.ps.pos
			}
			continue
		}

		if isComment(ch) {
			s.unread()
			tokens = append(tokens, s.scanComment())
			continue
		}

		if ch == CONTINUE {
			if crPOS > -1 { // eat up the , as a whitespace
				buf.WriteRune(ch)
				crPOS = -1
				continue
			}
		}

		s.unread()
		if crPOS > -1 {
			// we hit a non whitespace and we had a carrige return which means the last cr terminated the white space
			// prepend the CR token

			newtokens := make([]TokInfo, 1+len(tokens))
			newtokens[0] = crToken
			for i, v := range tokens {
				newtokens[i+1] = v
			}
			tokens = newtokens
		} else {
			tokens = append(tokens, TokInfo{Lstart: pbegin.line + 1,
				Cstart:  pbegin.col + 1,
				Pstart:  pbegin.pos,
				Lend:    s.ps.line + 1,
				Cend:    s.ps.col,
				Pend:    s.ps.pos,
				Token:   TOK_WS,
				Literal: buf.Bytes()})
		}

		break // normal char here
	}

	if len(tokens) == 0 {
		tokens = append(tokens, TokInfo{Lstart: s.ps.line + 1,
			Cstart:  pbegin.col + 1,
			Pstart:  pbegin.pos,
			Lend:    s.ps.line + 1,
			Cend:    s.ps.col,
			Pend:    s.ps.pos,
			Token:   TOK_WS,
			Literal: buf.Bytes()})
	}

	s.resume = makeDrain(s, tokens)
	return s.resume()
}

func makeDrain(s *Scanner, list []TokInfo) func() TokInfo {
	return func() TokInfo {
		if len(list) > 1 {
			s.resume = makeDrain(s, list[1:])
		} else {
			s.resume = nil
		}
		return list[0]
	}
}

func isWhite(ch rune) bool {
	return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r'
}

func isComment(ch rune) bool {
	return ch == '#'
}

func isBlock(ch rune) bool {
	return ch == '(' || ch == ')'
}
