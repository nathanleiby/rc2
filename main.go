package main

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/go-yaml/yaml"
)

func main() {
	// TODO: allow passing in workdir
	err := os.Chdir("example/")
	if err != nil {
		log.Fatal(err)
	}

	// Read config file
	// TODO: allow passing path to report-card.yml
	config, err := readReportCardConfig("report-card.yml")
	if err != nil {
		log.Fatal(err)
	}

	// For each check, execute the check
	results, err := runChecks(config)
	if err != nil {
		log.Fatal(err)
	}

	// Print results
	out, err := json.MarshalIndent(results, "", "    ")
	if err != nil {
		log.Fatal(err)
	}

	// Compute score
	score := computeScore(results)
	fmt.Println(string(out))
	fmt.Printf("Score = %.1f%%\n", score)
}

func computeScore(results map[string]Result) float64 {
	failures := 0
	for _, r := range results {
		if r.Outcome == "failure" {
			failures++
		}
	}

	return (1 - float64(failures)/float64(len(results))) * 100
}

type ConfigCheck struct {
	Type   string                 `yaml:"Type"`
	Config map[string]interface{} `yaml:"Config"` // Will need different configuration depending on check type
}

type ReportCardConfig struct {
	Version string                 `yaml:"Version"`
	Checks  map[string]ConfigCheck `yaml:"Checks"`
}

func readReportCardConfig(path string) (ReportCardConfig, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return ReportCardConfig{}, err
	}

	// parse checks
	conf := ReportCardConfig{}

	err = yaml.Unmarshal(data, &conf)
	if err != nil {
		return ReportCardConfig{}, err
	}

	// validate checks (e.g. perhaps a check takes a whitelist or blacklist, but not both)
	// TODO

	return conf, nil
}

// TODO: Add outcomes enum
//- success
//- warning
//- failure

func runChecks(conf ReportCardConfig) (map[string]Result, error) {
	// TODO: parallelize
	results := map[string]Result{}
	for title, c := range conf.Checks {
		var check Check
		switch c.Type {
		case "CheckFileExists":
			// TODO: check for errors instead of type assertion, since this is user config
			// Maybe: do a bunch of validation upfront?
			check = &CheckFileExists{
				Path: c.Config["Path"].(string),
			}
		case "CheckFileMD5":
			check = &CheckFileMD5{
				Path: c.Config["Path"].(string),
				Hash: c.Config["Hash"].(string),
			}
		case "CheckFileHasString":
			check = &CheckFileHasString{
				Path:   c.Config["Path"].(string),
				String: c.Config["String"].(string),
			}
		case "CheckNodeDependencies":
			blacklist := []string{}
			for _, item := range c.Config["Blacklist"].([]interface{}) {
				blacklist = append(blacklist, item.(string))
			}
			check = &CheckNodeDependencies{
				Blacklist: blacklist,
			}
		case "CheckDockerBaseImage":
			whitelist := []string{}
			for _, item := range c.Config["Whitelist"].([]interface{}) {
				whitelist = append(whitelist, item.(string))
			}
			check = &CheckDockerBaseImage{
				Whitelist: whitelist,
			}
		default:
			fmt.Printf("skipping %s...\n", title)
			continue
		}

		// Do whatever
		r, err := check.Execute()
		if err != nil {
			return nil, err
		}
		results[title] = r
	}

	return results, nil
}

type Check interface {
	Execute() (Result, error)
}

type Result struct {
	Outcome string
	Details string
}

type CheckNodeDependencies struct {
	Blacklist []string
}

func (c *CheckNodeDependencies) Execute() (Result, error) {
	data, err := ioutil.ReadFile("package.json")
	if err != nil {
		return Result{}, err
	}

	var packageJSON map[string]interface{}
	err = json.Unmarshal(data, &packageJSON)
	if err != nil {
		panic(err)
		//return Result{}, err
	}

	found := []string{}
	for _, key := range []string{"dependencies", "devDependencies"} {
		deps, ok := packageJSON[key].(map[string]interface{})
		if ok {
			for _, black := range c.Blacklist {
				if _, ok = deps[black]; ok {
					found = append(found, black)
				}
			}
		}
	}

	if len(found) > 0 {
		return Result{
			Outcome: "failure",
			Details: "found the following blacklisted packages: " + strings.Join(found, ","),
		}, nil
	}

	return Result{
		Outcome: "success",
	}, nil
}

type CheckDockerBaseImage struct {
	Whitelist []string
}

func (c *CheckDockerBaseImage) Execute() (Result, error) {
	data, err := ioutil.ReadFile("Dockerfile")
	if err != nil {
		return Result{}, err
	}

	lines := strings.Split(string(data), "\n")
	if !strings.HasPrefix(lines[0], "FROM") {
		return Result{}, fmt.Errorf("unable to determine base image from Dockerfile")
	}

	image := strings.TrimSpace(strings.Trim(lines[0], "FROM"))
	found := false
	for _, white := range c.Whitelist {
		if image == white {
			found = true
			break
		}
	}

	if !found {
		return Result{
			Outcome: "failure",
			Details: "dockerfile uses base image not found in whitelist: " + image,
		}, nil
	}

	return Result{
		Outcome: "success",
	}, nil
}

type CheckFileExists struct {
	Path string
}

func (c *CheckFileExists) Execute() (Result, error) {
	_, err := os.Stat(c.Path)
	if err != nil {
		if strings.Contains(err.Error(), "no such file") {
			return Result{
				Outcome: "failure",
			}, nil
		}
		return Result{}, err
	}

	return Result{
		Outcome: "success",
	}, nil
}

type CheckFileMD5 struct {
	Path string
	Hash string
}

func (c *CheckFileMD5) Execute() (Result, error) {
	f, err := os.Open(c.Path)
	if err != nil {
		if strings.Contains(err.Error(), "no such file") {
			return Result{
				Outcome: "failure",
				Details: "no such file",
			}, nil
		}
		return Result{}, err
	}
	defer f.Close()

	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		log.Fatal(err)
	}

	actual := fmt.Sprintf("%x", h.Sum(nil))
	if actual == c.Hash {
		return Result{
			Outcome: "success",
		}, nil
	}

	return Result{
		Outcome: "failure",
		Details: fmt.Sprint("actual md5 was: ", actual),
	}, nil
}

type CheckFileHasString struct {
	Path   string
	String string
}

func (c *CheckFileHasString) Execute() (Result, error) {
	b, err := ioutil.ReadFile(c.Path)
	if err != nil {
		if strings.Contains(err.Error(), "no such file") {
			return Result{
				Outcome: "failure",
				Details: "no such file",
			}, nil
		}
		return Result{}, err
	}

	if strings.Contains(string(b), c.String) {
		return Result{
			Outcome: "success",
		}, nil
	}

	return Result{
		Outcome: "failure",
	}, nil
}
