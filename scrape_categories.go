package kijiji

import (
	"sort"

	"fmt"

	"strings"

	"encoding/json"

	"github.com/jmichiels/tree"
	"github.com/pkg/errors"
)

type CategoryID int

type CategoryName struct {
	En, Fr string
}

const (
	localeFr = `fr_CA`
	localeEn = `en_CA`
)

// Set the name in the specified locale.
func (name *CategoryName) set(locale, value string) {
	switch locale {
	case localeFr:
		name.Fr = value
	case localeEn:
		name.En = value
	}
}

type Category struct {
	ID     CategoryID
	Name   CategoryName
	Parent CategoryID
}

func (category Category) String() string {
	return fmt.Sprintf("%s (%s, %d)", category.Name.En, category.Name.Fr, category.ID)
}

// IsTopLevel returns if this category is top level.
func (category Category) IsTopLevel() bool {
	return category.Parent == 0
}

type CategoryMap map[CategoryID]*Category

// ToSlice returns the categories as a slice.
func (categories CategoryMap) ToSlice() CategorySlice {
	slice := make(CategorySlice, 0, len(categories))
	for _, category := range categories {
		slice = append(slice, category)
	}
	return slice
}

type CategorySlice []*Category

// Returns the categories formatted as tree.
func (categories CategorySlice) FormatTree() string {
	return tree.String(categories)
}

// Implements 'tree.Tree'.
func (categories CategorySlice) RootNodes() []tree.Node {
	return categories.ChildrenNodes(&Category{})
}

// Implements 'tree.Tree'.
func (categories CategorySlice) ChildrenNodes(parent tree.Node) (children []tree.Node) {
	for _, category := range categories {
		if category.Parent == parent.(*Category).ID {
			children = append(children, category)
		}
	}
	return children
}

// Sort sorts the categories by increasing ID.
func (categories CategorySlice) Sort() CategorySlice {
	sort.Sort(categories)
	return categories
}

// Implements 'sort.Interface'.
func (categories CategorySlice) Len() int {
	return len(categories)
}

// Implements 'sort.Interface'.
func (categories CategorySlice) Less(i, j int) bool {
	return categories[i].ID < categories[j].ID
}

// Implements 'sort.Interface'.
func (categories CategorySlice) Swap(i, j int) {
	categories[i], categories[j] = categories[j], categories[i]
}

// Category as it appears in the source.
type sourceCategoryJSON struct {
	ID       int                   `json:"categoryId"`
	Name     string                `json:"categoryName"`
	Children []*sourceCategoryJSON `json:"children"`
}

// Adds the data to the categories list.
func (categories CategoryMap) fromSourceJSON(locale string, source *sourceCategoryJSON, parent CategoryID) {
	id := CategoryID(source.ID)

	category, registered := categories[id]
	if !registered {
		// The category is not registered yet, create it.
		category = &Category{
			ID:     id,
			Parent: parent,
		}
		categories[id] = category
	}

	// Save the name in the current locale.
	category.Name.set(locale, source.Name)

	// Take care of the children.
	for _, child := range source.Children {
		categories.fromSourceJSON(locale, child, id)
	}
}

// Scrape all the categories.
func ScrapeCategories() (CategoryMap, error) {
	categories := CategoryMap{}

	for _, locale := range []string{localeEn, localeFr} {
		// Get the categories source.
		body, err := get(`https://www.kijiji.ca/?userLocale=fr_CA`)
		if err != nil {
			return nil, err
		}

		// Find the start of the categories array.
		openingBracket := strings.Index(body, `[{"categoryName":`)
		if openingBracket < 0 {
			return nil, errors.New("opening bracket not found")
		}

		// Find the end of the categories array.
		closingBracket, err := closingCharIndex(body, openingBracket)
		if err != nil {
			return nil, err
		}

		// Unmarshal JSON from source.
		var unmarshalled []*sourceCategoryJSON
		if err := json.Unmarshal([]byte(body[openingBracket:closingBracket+1]), &unmarshalled); err != nil {
			return nil, errors.Wrap(err, "unmarshal json")
		}

		for _, category := range unmarshalled {
			// Add locations to locations map.
			categories.fromSourceJSON(locale, category, 0)
		}
	}

	return categories, nil
}
