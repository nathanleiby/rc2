package main

import (
	"fmt"
	"sort"

	"github.com/fatih/color"
)

func prettyPrintOutput(o Output) {
	fmt.Println("================================================================================")
	fmt.Println(`                            _                       _
  _ __ ___ _ __   ___  _ __| |_    ___ __ _ _ __ __| |
 | '__/ _ \ '_ \ / _ \| '__| __|  / __/ _\ | '__/ _| |
 | | |  __/ |_) | (_) | |  | |_  | (_| (_| | | | (_| |
 |_|  \___| .__/ \___/|_|   \__|  \___\__,_|_|  \__,_|
          |_|                                         `)
	fmt.Println("================================================================================")

	// UI Inspiration: https://github.com/ValeLint/vale
	failures := 0
	warnings := 0
	successes := 0

	// sort results
	var keys []string
	for k := range o.Results {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Print failures and warnings
	for _, title := range keys {
		result := o.Results[title]
		// TODO: Hide successes?
		if result.Outcome == "success" {
			successes++
			fmt.Printf("%s\t%s\n", color.GreenString(result.Outcome), title)
		}
		if result.Outcome == "failure" {
			failures++
			fmt.Printf("%s\t%s\n", color.RedString(result.Outcome), title)
			fmt.Printf("%s\t-> %s\n", "       ", result.Details)
		}
		if result.Outcome == "warning" {
			warnings++
			fmt.Printf("%s\t%s\n", color.YellowString(result.Outcome), title)
			fmt.Printf("%s\t-> %s\n", "       ", result.Details)
		}
	}

	// Print final summary
	fmt.Println("")
	scoreStr := color.CyanString("%.0f%%", o.Score)
	failuresStr := color.RedString(fmt.Sprintf("%d failures", failures))
	warningsStr := color.YellowString(fmt.Sprintf("%d warnings", warnings))
	fmt.Printf("%s\t%s, %s\n", scoreStr, failuresStr, warningsStr)
}
