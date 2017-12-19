package main

import (
	"fmt"
	"log"

	"github.com/jmichiels/kijiji"
)

func main() {

	// Scrape locations from the Kijiji homepage.
	locations, err := kijiji.ScrapeLocations(kijiji.AllLocales)
	if err != nil {
		log.Fatal(err)
	}

	// Print out locations as an ascii tree.
	fmt.Println(locations.ToSlice().Sort().FormatTree())
}
