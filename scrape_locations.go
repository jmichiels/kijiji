package kijiji

import (
	"bytes"
	"os/exec"
	"strconv"
	"strings"

	"fmt"
	"sort"

	"github.com/PuerkitoBio/goquery"
	"github.com/jmichiels/tree"
	"github.com/pkg/errors"
)

type LocationID int

func (id LocationID) MarshalText() (text []byte, err error) {
	return []byte(strconv.Itoa(int(id))), nil
}

func (id *LocationID) UnmarshalText(text []byte) error {
	idInt, err := strconv.Atoi(string(text))
	if err != nil {
		return err
	}
	*id = LocationID(idInt)
	return nil
}

type LocationName map[string]string

type Location struct {
	ID     LocationID
	Name   LocationName
	Parent LocationID
}

func (location Location) String() string {
	return fmt.Sprintf("%s (%s, %d)", location.Name[EN], location.Name[FR], location.ID)
}

type LocationMap map[LocationID]*Location

const rootLocationUlSelector = `div[class*=locationListContainer] > ul[class*=locationList]`

func dumpDOM(url string) ([]byte, error) {

	// Run google chrome in headless mode and dumps the rendered html.
	return exec.Command(`google-chrome`, `--headless`, `--disable-gpu`, `--dump-dom`, url).Output()
}

// Scrape all locations.
func ScrapeLocations(locales []string) (LocationMap, error) {
	locations := LocationMap{}

	for _, locale := range locales {

		scraped, err := dumpDOM(`https://www.kijiji.ca/?siteLocale=` + locale)
		if err != nil {
			return nil, errors.Wrap(err, "dump dom")
		}

		// Initialize a new goquery document from the scraped html.
		doc, err := goquery.NewDocumentFromReader(bytes.NewReader(scraped))
		if err != nil {
			return nil, errors.Wrap(err, "init")
		}

		// Recursively scrape locations from the root ul.
		if err := locations.addLocationUl(locale, doc.Find(rootLocationUlSelector), 0); err != nil {
			return nil, errors.Wrap(err, "scrape")
		}
	}

	return locations, nil
}

func (locations *LocationMap) addLocationUl(locale string, ul *goquery.Selection, parent LocationID) (err error) {
	if ul != nil {
		// Scrape every children locations li.
		ul.Children().EachWithBreak(func(i int, li *goquery.Selection) bool {
			if err = locations.addLocationLi(locale, li, parent); err != nil {
				return false
			}
			return true
		})
	}
	return err
}

func (locations *LocationMap) addLocationLi(locale string, li *goquery.Selection, parent LocationID) error {

	// Extract location id.
	idStr, exists := li.Attr(`id`)
	if !exists {
		return errors.New("missing location id")
	}

	// Remove group prefix.
	idStr = strings.TrimPrefix(idStr, `group-`)

	// Unmarshal id.
	var id LocationID
	if err := id.UnmarshalText([]byte(idStr)); err != nil {
		return errors.Wrap(err, "unmarshal id")
	}

	// Extract location name.
	name, exists := li.Find(`a`).Attr(`title`)
	if !exists {
		return errors.New("missing location name")
	}

	if location, registered := (*locations)[id]; registered {
		// Location already registered
		if _, named := location.Name[locale]; !named {
			// Add name in current locale.
			location.Name[locale] = name
		}
	} else {
		// Register location.
		(*locations)[id] = &Location{
			ID: LocationID(id),
			Name: LocationName{
				// Set name in current locale.
				locale: name,
			},
			Parent: parent,
		}
	}

	// Take care of the children.
	return locations.addLocationUl(locale, li.Find(`ul`), id)
}

func (locations LocationMap) ToSlice() LocationSlice {
	slice := make(LocationSlice, 0, len(locations))
	for _, location := range locations {
		slice = append(slice, location)
	}
	return slice
}

type LocationSlice []*Location

// Returns the locations formatted in an ascii tree.
func (locations LocationSlice) FormatTree() string {
	return tree.String(locations)
}

// Implements 'tree.Tree'.
func (locations LocationSlice) RootNodes() []tree.Node {
	return locations.ChildrenNodes(&Location{})
}

// Implements 'tree.Tree'.
func (locations LocationSlice) ChildrenNodes(parent tree.Node) (children []tree.Node) {
	for _, location := range locations {
		if location.Parent == parent.(*Location).ID {
			children = append(children, location)
		}
	}
	return children
}

func (locations LocationSlice) Sort() LocationSlice {
	sort.Sort(locations)
	return locations
}

func (locations LocationSlice) Len() int {
	return len(locations)
}

func (locations LocationSlice) Less(i, j int) bool {
	return locations[i].ID < locations[j].ID
}

func (locations LocationSlice) Swap(i, j int) {
	locations[i], locations[j] = locations[j], locations[i]
}
