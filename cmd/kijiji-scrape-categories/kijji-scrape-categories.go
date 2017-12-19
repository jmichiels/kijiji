package main

import (
	"log"

	"fmt"

	"github.com/jmichiels/kijiji"
)

func main() {

	// Scrape categories from the Kijiji homepage.
	categories, err := kijiji.ScrapeCategories(kijiji.AllLocales)
	if err != nil {
		log.Fatal(err)
	}

	// Print out categories as an ascii tree.
	fmt.Println(categories.ToSlice().Sort().FormatTree())
}
