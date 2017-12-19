package kijiji

import (
	"io/ioutil"
	"net/http"

	"fmt"

	"github.com/pkg/errors"
)

func get(url string) (string, error) {

	// Fetch the specified url.
	response, err := http.Get(url)
	if err != nil {
		return "", errors.Wrap(err, "fetch")
	}
	defer response.Body.Close()

	// Buffer the whole body.
	bodyBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", errors.Wrap(err, "read")
	}

	return string(bodyBytes), nil
}

func closingCharIndex(text string, openingCharIndex int) (int, error) {

	// Find opening char in text.
	openingChar := rune(text[openingCharIndex])

	// Deduce corresponding closing char.
	closingChar, supported := (map[rune]rune{
		'{': '}',
		'(': ')',
		'[': ']',
	})[openingChar]
	if !supported {
		return -1, fmt.Errorf("invalid opening char '%s'")
	}
	var depth int
	for idx, char := range text[openingCharIndex:] {
		if char == openingChar {
			depth++
		}
		if char == closingChar {
			depth--
		}
		if depth == 0 {
			return idx + openingCharIndex, nil
		}
	}
	return -1, fmt.Errorf("closing char not found")
}
