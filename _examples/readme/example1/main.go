package main

import (
	"fmt"
	"github.com/go-andiamo/iccarus"
	"github.com/go-andiamo/iccarus/_test_data/profiles"
	"log"
)

func main() {
	f, err := profiles.Open("default/ISOcoated_v2_300_eci.icc")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	profile, err := iccarus.ParseProfile(f, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Color space: %s\n", profile.Header.ColorSpace)
	fmt.Printf("    Version: %s\n", profile.Header.Version)
	if cprt, err := profile.TagValue(iccarus.TagHeaderCopyright); err == nil {
		fmt.Printf("  Copyright: %s\n", cprt)
	}
}
