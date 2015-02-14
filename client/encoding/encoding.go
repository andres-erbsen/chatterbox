package encoding

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

var replace = map[rune]string{
	'\x00': "zero",
	'\\':   "backslash",
	'/':    "slash",
	':':    "colon",
	'*':    "asterisk",
	'?':    "questionmark",
	'"':    "doublequote",
	'<':    "lessthan",
	'>':    "greaterthan",
	'|':    "verticalbar",
	'%':    "percent",
}

// EscapeFilename computes an invertible, human readable mapping between
// arbitrary strings and (longer) strings that do not contain \/:*?"<>| and
// non-printable charactes, as defined by unicode.IsPrint
func EscapeFilename(s string) string {
	const hextable = "0123456789abcdef"
	ret := ""
	for _, c := range s {
		if r, ok := replace[c]; ok {
			ret += "%" + r
		} else if !unicode.IsPrint(c) {
			ret += "%"
			var bs [4]byte
			n := utf8.EncodeRune(bs[:], c)
			for i := 0; i < n; i++ {
				ret += string([]byte{hextable[bs[i]>>4], hextable[bs[i]&0x0f]})
			}
		} else {
			ret += string(c)
		}
	}
	return ret
}

// UnescapeFilename computes the inverse of EscapeFilename
func UnescapeFilename(s string) (string, error) {
	ret := ""
	skipRemaining := 0
parse:
	for i, c := range s {
		if skipRemaining != 0 {
			skipRemaining--
			continue
		}
		if c != '%' {
			ret += string(c)
			continue parse
		}
		remainder := s[i+1:]
		for match, pattern := range replace {
			if strings.HasPrefix(remainder, pattern) {
				ret += string(match)
				skipRemaining = len(pattern)
				continue parse
			}
		}
		c, l, bad := decodeHexUtf8Rune(remainder)
		if bad {
			return ret, fmt.Errorf("Invalid hex-encoded utf8 codepoint at index %d", i)
		}
		skipRemaining = l
		ret += string(c)
	}
	return ret, nil
}

func unhex(c byte) (v byte, ok bool) {
	switch {
	case '0' <= c && c <= '9':
		return byte(c - '0'), true
	case 'a' <= c && c <= 'f':
		return byte(c - 'a' + 10), true
	case 'A' <= c && c <= 'F':
		return byte(c - 'A' + 10), true
	}
	return
}

func unhexByte(s string) (r byte, ok bool) {
	var l byte
	if l, ok = unhex(s[0]); !ok {
		return 0, false
	}
	if r, ok = unhex(s[1]); !ok {
		return 0, false
	}
	return (l << 4) | r, ok
}

func decodeHexUtf8Rune(s string) (r rune, size int, short bool) {
	const (
		surrogateMin = 0xD800
		surrogateMax = 0xDFFF
	)

	const (
		t1 = 0x00 // 0000 0000
		tx = 0x80 // 1000 0000
		t2 = 0xC0 // 1100 0000
		t3 = 0xE0 // 1110 0000
		t4 = 0xF0 // 1111 0000
		t5 = 0xF8 // 1111 1000

		maskx = 0x3F // 0011 1111
		mask2 = 0x1F // 0001 1111
		mask3 = 0x0F // 0000 1111
		mask4 = 0x07 // 0000 0111

		rune1Max = 1<<7 - 1
		rune2Max = 1<<11 - 1
		rune3Max = 1<<16 - 1
	)
	n := len(s) / 2
	if n < 1 {
		return utf8.RuneError, 0, true
	}
	c0, ok := unhexByte(s[:2])
	if !ok {
		return utf8.RuneError, 0, true
	}

	// 1-byte, 7-bit sequence?
	if c0 < tx {
		return rune(c0), 2, false
	}

	// unexpected continuation byte?
	if c0 < t2 {
		return utf8.RuneError, 2, false
	}

	// need first continuation byte
	if n < 2 {
		return utf8.RuneError, 2, true
	}
	c1, ok := unhexByte(s[2:4])
	if !ok {
		return utf8.RuneError, 0, true
	}
	if c1 < tx || t2 <= c1 {
		return utf8.RuneError, 2, false
	}

	// 2-byte, 11-bit sequence?
	if c0 < t3 {
		r = rune(c0&mask2)<<6 | rune(c1&maskx)
		if r <= rune1Max {
			return utf8.RuneError, 2, false
		}
		return r, 4, false
	}

	// need second continuation byte
	if n < 3 {
		return utf8.RuneError, 2, true
	}
	c2, ok := unhexByte(s[4:6])
	if !ok {
		return utf8.RuneError, 0, true
	}
	if c2 < tx || t2 <= c2 {
		return utf8.RuneError, 2, false
	}

	// 3-byte, 16-bit sequence?
	if c0 < t4 {
		r = rune(c0&mask3)<<12 | rune(c1&maskx)<<6 | rune(c2&maskx)
		if r <= rune2Max {
			return utf8.RuneError, 2, false
		}
		if surrogateMin <= r && r <= surrogateMax {
			return utf8.RuneError, 2, false
		}
		return r, 6, false
	}

	// need third continuation byte
	if n < 4 {
		return utf8.RuneError, 2, true
	}
	c3, ok := unhexByte(s[6:8])
	if !ok {
		return utf8.RuneError, 0, true
	}
	if c3 < tx || t2 <= c3 {
		return utf8.RuneError, 2, false
	}

	// 4-byte, 21-bit sequence?
	if c0 < t5 {
		r = rune(c0&mask4)<<18 | rune(c1&maskx)<<12 | rune(c2&maskx)<<6 | rune(c3&maskx)
		if r <= rune3Max || utf8.MaxRune < r {
			return utf8.RuneError, 2, false
		}
		return r, 8, false
	}

	// error
	return utf8.RuneError, 2, false
}

func EscapeFilenames(ss []string) []string {
	ret := make([]string, 0, len(ss))
	for _, s := range ss {
		ret = append(ret, EscapeFilename(s))
	}
	return ret
}
