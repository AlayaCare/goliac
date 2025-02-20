package entity

import (
	"github.com/go-git/go-billy/v5"
	"github.com/goliac-project/goliac/internal/utils"
	"gopkg.in/yaml.v3"
)

type Warning error

type Entity struct {
	ApiVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Name       string `yaml:"name"`
}

/*
 * parseEntity is a generic function that is used to parse any JSON file
 * and discover the apiVersion and kind of the file.
 */
func parseEntity(fs billy.Filesystem, filename string) (*Entity, error) {
	filecontent, err := utils.ReadFile(fs, filename)
	if err != nil {
		return nil, err
	}

	yamldata := &Entity{}
	err = yaml.Unmarshal(filecontent, yamldata)
	if err != nil {
		return nil, err
	}

	return yamldata, nil
}
