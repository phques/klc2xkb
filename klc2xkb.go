// Klc2Xkb project
// Copyright 2016 Philippe Quesnel
// Licensed under the Academic Free License version 3.0
package main

import (
	"fmt"
	"os"
	"strings"
	"text/scanner"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

type KlcScanner struct {
	scanner.Scanner
	filename string //input
	outfile  *os.File

	scancodeToKey map[string]string

	token      rune
	tokenText  string
	pushedBack bool

	description   string
	layoutColumns [10]bool // output column X if true
	hasAltGr      bool
}

//---------

func newScanner(filename, outFilename string) (*KlcScanner, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	outfile, err := os.Create(outFilename)
	if err != nil {
		file.Close()
		return nil, err
	}

	scanr := &KlcScanner{filename: filename, outfile: outfile}
	scanr.Init(file)
	scanr.Mode = scanner.ScanIdents | scanner.ScanFloats | scanner.ScanStrings
	scanr.Mode |= scanner.ScanComments | scanner.SkipComments

	scanr.initScancodes()

	return scanr, nil
}

func (scanr *KlcScanner) initScancodes() {

	codes := make(map[string]string)
	scanr.scancodeToKey = codes

	codes["29"] = "TLDE"
	codes["02"] = "AE01"
	codes["03"] = "AE02"
	codes["04"] = "AE03"
	codes["05"] = "AE04"
	codes["06"] = "AE05"
	codes["07"] = "AE06"
	codes["08"] = "AE07"
	codes["09"] = "AE08"
	codes["0a"] = "AE09"
	codes["0b"] = "AE10"
	codes["0c"] = "AE11"
	codes["0d"] = "AE12"

	codes["10"] = "AD01"
	codes["11"] = "AD02"
	codes["12"] = "AD03"
	codes["13"] = "AD04"
	codes["14"] = "AD05"
	codes["15"] = "AD06"
	codes["16"] = "AD07"
	codes["17"] = "AD08"
	codes["18"] = "AD09"
	codes["19"] = "AD10"
	codes["1a"] = "AD11"
	codes["1b"] = "AD12"
	codes["2b"] = "BKSL"

	codes["1e"] = "AC01"
	codes["1f"] = "AC02"
	codes["20"] = "AC03"
	codes["21"] = "AC04"
	codes["22"] = "AC05"
	codes["23"] = "AC06"
	codes["24"] = "AC07"
	codes["25"] = "AC08"
	codes["26"] = "AC09"
	codes["27"] = "AC10"
	codes["28"] = "AC11"

	codes["2c"] = "AB01"
	codes["2d"] = "AB02"
	codes["2e"] = "AB03"
	codes["2f"] = "AB04"
	codes["30"] = "AB05"
	codes["31"] = "AB06"
	codes["32"] = "AB07"
	codes["33"] = "AB08"
	codes["34"] = "AB09"
	codes["35"] = "AB10"

	// dont output
	codes["39"] = "-"
	codes["53"] = "-"
	codes["56"] = "-"

}

func (scanr *KlcScanner) close() {
	scanr.outfile.Close()
	scanr.outfile = nil
}

//-----------

// returns true if current token is IDENT(name)
func (scanr *KlcScanner) isIdent(name string) bool {
	return scanr.token == scanner.Ident && scanr.tokenText == name
}

// read next token into token/tokenText
func (scanr *KlcScanner) getNextToken() {
	// use previoulsy 'pushed back' token
	if scanr.pushedBack {
		scanr.pushedBack = false
		return
	}

	for scanr.token != scanner.EOF {
		scanr.token = scanr.Scan()

		// ';' isa comment to end of line
		if scanr.token == ';' {
			// skip rest of chars on line
			for scanr.token != '\n' && scanr.token != scanner.EOF {
				scanr.token = scanr.Next()
			}
			continue
		}

		scanr.tokenText = scanr.TokenText()
		break
	}
}

// set the ident func so that everything is an ident,
// separated by whitespace (supports comments)
func (scanr *KlcScanner) setWhitespaceMode() {

	// set func to accept everything except spaces or comments as valid ident
	scanr.IsIdentRune = func(ch rune, i int) bool {
		// start of comment
		if ch == '/' || ch == ';' {
			return false
		}

		// whitespace
		if ch <= ' ' {
			if (scanr.Whitespace & (1 << uint(ch))) != 0 {
				return false
			}
		}

		// anything else is valid 'ident'
		return true
	}
}

// read next token, check that it is IDENT(expected)
func (scanr *KlcScanner) expectIdent(expected string) {
	scanr.getNextToken()
	if !scanr.isIdent(expected) {
		fmt.Printf("%v expecting ident %s\n", scanr.Pos(), expected)
		fmt.Printf("got %s %s\n", scanner.TokenString(scanr.token), scanr.tokenText)
		os.Exit(-1)
	}
}

// read next token, check that it is a string
// name is used in error output
func (scanr *KlcScanner) expectString(name string) {
	scanr.getNextToken()
	if scanr.token != scanner.String {
		fmt.Printf("%v expecting %s string\n", scanr.Pos(), name)
		fmt.Printf("got %s %s\n", scanner.TokenString(scanr.token), scanr.tokenText)
		os.Exit(-1)
	}
}

// skip next data until a string
func (scanr *KlcScanner) skipToString(name string) {
	for scanr.token != scanner.EOF && scanr.Peek() != '"' {
		scanr.token = scanr.Next()
	}

	scanr.expectString(name)
}

// skip next data until IDENT(skipTo)
func (scanr *KlcScanner) skipToIdent(skipTo string) {
	scanr.getNextToken()
	for !scanr.isIdent(skipTo) {
		scanr.getNextToken()
	}

	if scanr.tokenText != skipTo {
		fmt.Printf("%v expecting ident %s\n", scanr.Pos(), skipTo)
		fmt.Printf("got %s %s\n", scanner.TokenString(scanr.token), scanr.tokenText)
		os.Exit(-1)
	}
}

// push back current token,
// will be used as next token
func (scanr *KlcScanner) pushBack() {
	scanr.pushedBack = true
}

//------------------

// scan KBD	kbdnam	"Canadian French - Custom"
func (scanr *KlcScanner) scanKbd() string {
	fmt.Println("*scanKbd")

	// dont try to parse the name,
	// just skip until the description string
	scanr.skipToString("description")
	scanr.description = scanr.tokenText

	//output header
	fmt.Fprintln(scanr.outfile, "partial alphanumeric_keys")
	fmt.Fprintln(scanr.outfile, "xkb_symbols \"??\" {")
	fmt.Fprintf(scanr.outfile, "  name[Group1]= %s;\n", scanr.description)

	return "SHIFTSTATE"
}

// scan SHIFTSTATE section
/*
SHIFTSTATE

0	//Column 4
1	//Column 5 : Shft
2	//Column 6 :       Ctrl
6	//Column 7 :       Ctrl Alt
7	//Column 8 : Shft  Ctrl Alt
*/
func (scanr *KlcScanner) scanShiftStates() string {
	fmt.Println("*scanShiftStates")

	columnNo := 4 // actual keys start at col 4

	// read next integer / shiftstate
	scanr.getNextToken()
	for scanr.token == scanner.Int {
		// is it one of the shiftstates we want ?
		if strings.Contains("0167", scanr.tokenText) {
			// mark this column to be outputed
			scanr.layoutColumns[columnNo] = true

			if scanr.tokenText == "6" || scanr.tokenText == "7" {
				scanr.hasAltGr = true
			}
		}

		scanr.getNextToken()
		columnNo += 1
	}

	scanr.pushBack()
	return "LAYOUT"
}

// scan the keys / layout
/*
LAYOUT		;an extra '@' at the end is a dead key

//SC	VK_		Cap	0	1	2	6
//--	----		----	----	----	----	----

02	1		0	1	0021	-1	00b1		// DIGIT ONE, EXCLAMATION MARK, <none>, PLUS-MINUS SIGN
03	2		0	2	0022	-1	0040		// DIGIT TWO, QUOTATION MARK, <none>, COMMERCIAL AT
*/
func (scanr *KlcScanner) scanLayout() string {
	fmt.Println("*scanLayout")

	// dont try to parse integers etc
	scanr.setWhitespaceMode()

	// read next scancode (skips comments)
	scanr.getNextToken()

	//  dont skip comments from here on !
	scanr.Mode &= ^uint(scanner.SkipComments)

	for scanr.token != scanner.EOF {
		// check scancode, get xkb keyname
		scanCode := scanr.tokenText
		keyname := scanr.scancodeToKey[scanCode]

		// not a known scancode, we're done
		if keyname == "" {
			// done with layout
			if scanr.hasAltGr {
				// need to include altgr stuff
				fmt.Fprintln(scanr.outfile, "  include \"level3(ralt_switch)\"")
			}
			fmt.Fprintln(scanr.outfile, "};")

			scanr.pushBack()
			return "STOP"
		}

		//éé debug
		fmt.Printf("scan %v key %v\n", scanCode, keyname)

		// some scancodes/keys we just skip
		if keyname == "-" {
			scanr.getNextToken()
			for scanr.token != scanner.Comment {
				scanr.getNextToken()
			}

			scanr.getNextToken()
			continue
		}

		scanr.scanLayoutKeyRow(keyname)

		scanr.getNextToken()
	}

	fmt.Printf("%v unexpected end of file\n", scanr.Pos())
	return "STOP"
}

func (scanr *KlcScanner) scanLayoutKeyRow(keyname string) {
	// output beginning of new key
	fmt.Fprintf(scanr.outfile, "  key <%s> { [ ", keyname)

	// get next columns, output until last one = comment
	firstCol := true
	columnNo := 2
	scanr.getNextToken()
	for scanr.token != scanner.Comment {

		// do  we want this column ?
		if scanr.layoutColumns[columnNo] {

			// check for '@' suffix for deakeys
			char := scanr.tokenText

			if strings.HasSuffix(char, "@") {
				charStripped := char[:len(char)-1]
				fmt.Printf("%v warning, deadkeys not supported %s => %s\n",
					scanr.Pos(), char, charStripped)
				char = charStripped
			}

			// -1 => NoSymbol
			if char == "-1" {
				char = "NoSymbol"
			} else if len(char) == 4 {
				// must be unicode value '002b', prefix with U
				char = "U" + char
			}

			if !firstCol {
				fmt.Fprint(scanr.outfile, ", ")
			}
			fmt.Fprint(scanr.outfile, char)
			firstCol = false
		}

		// next column
		columnNo += 1
		scanr.getNextToken()
	}

	// last column should be a comment
	if scanr.token != scanner.Comment {
		fmt.Printf("%v expecting comment as last column in layout section\n", scanr.Pos())
		fmt.Printf(" got %v\n", scanr.tokenText)
		os.Exit(-1)
	}

	// output end of key line with comment
	fmt.Fprintf(scanr.outfile, " ] };  %s\n", scanr.tokenText)

}

func (scanr *KlcScanner) scan() {
	state := ""

	for scanr.token != scanner.EOF {
		switch state {
		case "":
			scanr.expectIdent("KBD")
			state = scanr.scanKbd()

		case "SHIFTSTATE":
			scanr.skipToIdent("SHIFTSTATE")
			state = scanr.scanShiftStates()

		case "LAYOUT":
			scanr.expectIdent("LAYOUT")
			state = scanr.scanLayout()

		case "STOP":
			return

		default:
			return

		}
	}
}

func (scanr *KlcScanner) scan_() {

	scanr.getNextToken()

	for scanr.token != scanner.EOF {

		fmt.Println("At position", scanr.Pos(), ":",
			scanner.TokenString(scanr.token), scanr.tokenText)

		if scanr.token != scanner.Ident {
			tokenString := scanner.TokenString(scanr.token)
			fmt.Println("Unexpected token ", tokenString, " : ", scanr.tokenText)
			fmt.Println(" at ", scanr.Pos())
			return
		}

		switch scanr.tokenText {
		case "KBD":
			scanr.scanKbd()

		case "SHIFTSTATE":
			scanr.scanShiftStates()

		case "LAYOUT":
			scanr.scanLayout()

		case "DEADKEY":
			scanr.IsIdentRune = nil
			scanr.Mode |= scanner.SkipComments
			fmt.Println("**out of layout")

			// we actually just stop here ;-P
			return

		default:
			//ignore this keyword, skip to eol
			fmt.Printf("skiping (%s) to eol ", scanr.tokenText)
			for scanr.token != '\n' && scanr.token != scanner.EOF {
				scanr.token = scanr.Next()
				fmt.Printf("%c", scanr.token)
			}
		}

		scanr.getNextToken()
	}
}

func main() {
	// Open the file.
	if len(os.Args) != 3 {
		fmt.Println("parameters: inputKlcFile outputFile")
		return
	}

	scanr, err := newScanner(os.Args[1], os.Args[2])
	check(err)

	//scan
	scanr.scan()
	scanr.close()
}
