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

	"github.com/xeipuuv/gojsonschema"
)

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

type CheckFileIsValidJSON struct {
	Path string
}

func (c *CheckFileIsValidJSON) Execute() (Result, error) {
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

	var js map[string]interface{}
	err = json.Unmarshal(b, &js)
	if err == nil {
		return Result{
			Outcome: "success",
		}, nil
	}

	return Result{
		Outcome: "failure",
		Details: "file is not valid JSON",
	}, nil
}

type CheckFileHasJSONSchema struct {
	Path       string
	SchemaPath string
}

func (c *CheckFileHasJSONSchema) Execute() (Result, error) {
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
	documentLoader := gojsonschema.NewStringLoader(string(b))

	b, err = ioutil.ReadFile(c.SchemaPath)
	if err != nil {
		if strings.Contains(err.Error(), "no such file") {
			return Result{
				Outcome: "failure",
				Details: "no such file",
			}, nil
		}
		return Result{}, err
	}
	schemaLoader := gojsonschema.NewStringLoader(string(b))

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		fmt.Println(err)
		panic(err.Error())
	}

	if result.Valid() {
		return Result{
			Outcome: "success",
		}, nil
	} else {
		errDetails := []string{}
		for _, desc := range result.Errors() {
			errDetails = append(errDetails, desc.String())
		}
		return Result{
			Outcome: "failure",
			Details: strings.Join(errDetails, " | "),
		}, nil
	}

}
