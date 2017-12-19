package kijiji

import (
	"os/exec"
	"strconv"

	"fmt"
	"sort"

	"net/http"

	"io/ioutil"

	"strings"

	"encoding/json"

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

type LocationName struct {
	En, Fr string
}

type Location struct {
	ID     LocationID
	Name   LocationName
	Parent *LocationID // nil for root locations.
}

func (location Location) String() string {
	return fmt.Sprintf("%s (%s, %d)", location.Name.En, location.Name.Fr, location.ID)
}

type LocationMap map[LocationID]*Location

const rootLocationUlSelector = `div[class*=locationListContainer] > ul[class*=locationList]`

func dumpDOM(url string) ([]byte, error) {

	// Run google chrome in headless mode and dumps the rendered html.
	return exec.Command(`google-chrome`, `--headless`, `--disable-gpu`, `--dump-dom`, url).Output()
}

type sourceLocationJSON struct {
	ID               int                   `json:"id"`
	NameFr           string                `json:"nameFr"`
	NameEn           string                `json:"nameEn"`
	Leaf             bool                  `json:"leaf"`
	Homepage         string                `json:"homePageSEOUrl"`
	RegionLabel      string                `json:"regionLabel,omitempty"`
	Children         []*sourceLocationJSON `json:"children,omitempty"`
	MigratedLocation bool                  `json:"migratedLocation"`
}

const pathLocations = `https://www.kijiji.ca/j-locations.json`

// Scrape all locations.
func ScrapeLocations() (LocationMap, error) {

	// Fetch source of locations.
	response, err := http.Get(pathLocations)
	if err != nil {
		return nil, errors.Wrap(err, "fetch")
	}

	// Buffer the whole body.
	bodyBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, errors.Wrap(err, "read")
	}
	body := string(bodyBytes)

	// Find the opening brace.
	openingBrace := strings.IndexByte(body, '{')
	if openingBrace < 0 {
		return nil, errors.New("no opening brace in source")
	}

	// Find the closing brace.
	closingBrace := strings.LastIndexByte(body, '}')
	if closingBrace < 0 {
		return nil, errors.New("no closing brace in source")
	}

	// Unmarshal JSON from source.
	var unmarshalled sourceLocationJSON
	if err := json.Unmarshal([]byte(body[openingBrace:closingBrace+1]), &unmarshalled); err != nil {
		return nil, errors.Wrap(err, "unmarshal json")
	}

	locations := LocationMap{}
	// Add locations to locations map.
	locations.fromSourceJSON(&unmarshalled, nil)

	return locations, nil
}

func (locations LocationMap) fromSourceJSON(location *sourceLocationJSON, parent *LocationID) {
	if location != nil {
		id := LocationID(location.ID)
		// Add new location to the map.
		locations[id] = &Location{
			ID: id,
			Name: LocationName{
				Fr: location.NameFr,
				En: location.NameEn,
			},
			Parent: parent,
		}
		for _, child := range location.Children {
			// Add children locations to the map.
			locations.fromSourceJSON(child, &id)
		}
	}
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

	//// Extract location id.
	//idStr, exists := li.Attr(`id`)
	//if !exists {
	//	return errors.New("missing location id")
	//}
	//
	//// Remove "group" prefix.
	//idStr = strings.TrimPrefix(idStr, `group-`)
	//
	//// Unmarshal id.
	//var id LocationID
	//if err := id.UnmarshalText([]byte(idStr)); err != nil {
	//	return errors.Wrap(err, "unmarshal id")
	//}
	//
	//// Extract location name.
	//name, exists := li.Find(`a`).Attr(`title`)
	//if !exists {
	//	return errors.New("missing location name")
	//}

	//if location, registered := (*locations)[id]; registered {
	//	// Location already registered
	//	if _, named := location.Name[locale]; !named {
	//		// Add name in current locale.
	//		location.Name[locale] = name
	//	}
	//} else {
	//	// Register location.
	//	(*locations)[id] = &Location{
	//		ID: LocationID(id),
	//		Name: LocationName{
	//			// Set name in current locale.
	//			locale: name,
	//		},
	//		Parent: parent,
	//	}
	//}

	// Take care of the children.
	//return locations.addLocationUl(locale, li.Find(`ul`), id)
	return nil
}

// ToSlice returns the locations as a slice.
func (locations LocationMap) ToSlice() LocationSlice {
	slice := make(LocationSlice, 0, len(locations))
	for _, location := range locations {
		slice = append(slice, location)
	}
	return slice
}

// LocationSlice represents a list of locations.
type LocationSlice []*Location

// Sort sorts the locations by increasing ID.
func (locations LocationSlice) Sort() LocationSlice {
	sort.Sort(locations)
	return locations
}

// Implements 'sort.Interface'.
func (locations LocationSlice) Len() int {
	return len(locations)
}

// Implements 'sort.Interface'.
func (locations LocationSlice) Less(i, j int) bool {
	return locations[i].ID < locations[j].ID
}

// Implements 'sort.Interface'.
func (locations LocationSlice) Swap(i, j int) {
	locations[i], locations[j] = locations[j], locations[i]
}

// Returns the locations formatted as a tree.
func (locations LocationSlice) FormatTree() string {
	return tree.String(locations)
}

// Implements 'tree.Tree'.
func (locations LocationSlice) RootNodes() (nodes []tree.Node) {
	for _, location := range locations {
		if location.Parent == nil {
			nodes = append(nodes, location)
		}
	}
	return nodes
}

// Implements 'tree.Tree'.
func (locations LocationSlice) ChildrenNodes(parent tree.Node) (children []tree.Node) {
	for _, location := range locations {
		if location.Parent != nil && *location.Parent == parent.(*Location).ID {
			children = append(children, location)
		}
	}
	return children
}
