package kijiji

import (
	"context"

	"sort"

	"fmt"

	"github.com/jmichiels/tree"
	cdp "github.com/knq/chromedp"
	"github.com/pkg/errors"
)

const (
	FR = "fr_CA"
	EN = "en_CA"
)

var AllLocales = []string{EN, FR}

func withChromeInstance(do func(ctxt context.Context, instance *cdp.CDP) error) error {

	// Create context.
	ctxt, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create Chrome instance.
	instance, err := cdp.New(ctxt)
	if err != nil {
		return errors.Wrap(err, "startup chrome")
	}

	// Run action.
	if err = do(ctxt, instance); err != nil {
		return err
	}

	// Shutdown Chrome.
	if err := instance.Shutdown(ctxt); err != nil {
		return errors.Wrap(err, "shutdown chrome")
	}

	// Wait for Chrome to finish.
	if err := instance.Wait(); err != nil {
		return errors.Wrap(err, "wait chrome shutdown")
	}

	return nil
}

type CategoryID int

type CategoryName map[string]string

type Category struct {
	ID     CategoryID
	Name   CategoryName
	Parent CategoryID
}

func (category Category) String() string {
	return fmt.Sprintf("%s (%s, %d)", category.Name[EN], category.Name[FR], category.ID)
}

// IsTopLevel returns if this category is top level.
func (category Category) IsTopLevel() bool {
	return category.Parent == 0
}

// Category as it appears in the original data.
type windowDataCategory struct {
	ID       int                   `json:"categoryId"`
	Name     string                `json:"categoryName"`
	Children []*windowDataCategory `json:"children"`
}

type CategoryMap map[CategoryID]*Category

func (categories CategoryMap) TopLevelCategories() CategorySlice {
	slice := make(CategorySlice, 0)
	for _, category := range categories {
		if category.IsTopLevel() {
			slice = append(slice, category)
		}
	}
	return slice
}

func (categories CategoryMap) ToSlice() CategorySlice {
	slice := make(CategorySlice, 0, len(categories))
	for _, category := range categories {
		slice = append(slice, category)
	}
	return slice
}

type CategorySlice []*Category

// Returns the categories formatted in an ascii tree.
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

func (categories CategorySlice) Sort() CategorySlice {
	sort.Sort(categories)
	return categories
}

func (categories CategorySlice) Len() int {
	return len(categories)
}

func (categories CategorySlice) Less(i, j int) bool {
	return categories[i].ID < categories[j].ID
}

func (categories CategorySlice) Swap(i, j int) {
	categories[i], categories[j] = categories[j], categories[i]
}

// Adds the data to the categories list.
func (categories *CategoryMap) add(locale string, data *windowDataCategory, parent CategoryID) {

	if category, registered := (*categories)[CategoryID(data.ID)]; registered {
		// The category already registered, just save the name in the current locale.
		category.Name[locale] = data.Name

	} else {
		// The category does not registered yet, create it.
		(*categories)[CategoryID(data.ID)] = &Category{
			ID: CategoryID(data.ID),
			Name: CategoryName{
				// Set name in current locale.
				locale: data.Name,
			},
			Parent: parent,
		}
	}
	// Take care of the children.
	for _, child := range data.Children {
		categories.add(locale, child, CategoryID(data.ID))
	}
}

// chromedp action to go to the homepage.
func actionGoToHomePage(locale string) cdp.Action {
	return cdp.Navigate(`https://www.kijiji.ca/?siteLocale=` + locale)
}

// chromedp tasks list to evaluate a javascript expression on the homepage.
func taskEvaluate(locale, expression string, data interface{}) cdp.Tasks {
	return cdp.Tasks{
		actionGoToHomePage(locale),
		cdp.WaitReady(`window`),
		cdp.Evaluate(expression, &data),
	}
}

const categoriesExpression = `window.__data.categoryMenu.categories`

// Scrape all the categories.
func ScrapeCategories(locales []string) (CategoryMap, error) {
	categories := CategoryMap{}

	for _, locale := range locales {

		// Open a Chrome instance.
		err := withChromeInstance(func(ctxt context.Context, instance *cdp.CDP) error {

			dataCategories := make([]*windowDataCategory, 0)
			// Scrape `window.__data.categoryMenu.categories` from the Kijiji homepage.
			if err := instance.Run(ctxt, taskEvaluate(locale, categoriesExpression, &dataCategories)); err != nil {
				return errors.Wrap(err, "run tasks list")
			}

			for _, dataCategory := range dataCategories {
				// Format data in our own way.
				categories.add(locale, dataCategory, 0)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	return categories, nil
}
