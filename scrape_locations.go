package kijiji

import (
	"strconv"

	"fmt"
	"sort"

	"net/http"

	"io/ioutil"

	"strings"

	"encoding/json"

	"github.com/jmichiels/tree"
	"github.com/pkg/errors"
)

// LocationID uniquly identifies a location.
type LocationID int

// Implements 'encoding.TextMarshaler'.
func (id LocationID) MarshalText() (text []byte, err error) {
	return []byte(strconv.Itoa(int(id))), nil
}

// Implements 'encoding.TextUnmarshaler'.
func (id *LocationID) UnmarshalText(text []byte) error {
	idInt, err := strconv.Atoi(string(text))
	if err != nil {
		return err
	}
	*id = LocationID(idInt)
	return nil
}

// LocationName holds a location name in English and French.
type LocationName struct {
	En, Fr string
}

// Location represents a location (used in the queries).
type Location struct {
	ID     LocationID
	Name   LocationName
	Parent *LocationID // nil for top level locations.
}

// Implements 'fmt.Stringer'.
func (location Location) String() string {
	return fmt.Sprintf("%s (%s, %d)", location.Name.En, location.Name.Fr, location.ID)
}

// LocationMap represents a map of locations.
type LocationMap map[LocationID]*Location

// Source for the locations data.
const sourceLocations = `https://www.kijiji.ca/j-locations.json`

// location JSON as it appears in the source.
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

// Scrape all locations.
func ScrapeLocations() (LocationMap, error) {

	// Fetch source of locations.
	response, err := http.Get(sourceLocations)
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
