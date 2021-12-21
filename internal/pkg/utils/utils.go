package utils

import (
	"io/ioutil"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

type Secret struct {
	Name             string
	SourceNamespace  string   `yaml:"sourceNamespace"`
	TargetNamespaces []string `yaml:"targetNamespaces"`
}

type secretList struct {
	Secrets []Secret
}

// GetConfigFromFile reads secret configuration from file
func GetConfigFromFile(fName string) (*secretList, error) {
	l := &secretList{}
	yamlFile, err := ioutil.ReadFile(fName)
	if err != nil {
		log.WithFields(log.Fields{}).Error("Error open config file: ", err)
		return nil, err
	}
	err = yaml.Unmarshal(yamlFile, l)
	if err != nil {
		log.WithFields(log.Fields{}).Error("Error unmarschal secrets: ", err)
		return nil, err
	}
	log.WithFields(log.Fields{}).Debug("Secrets: ", l)

	return l, nil
}
