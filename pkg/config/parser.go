/*
Copyright 2020 Daniël Franke

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package config manages configuration, via files, env vars and command line flags.
package config

import (
	"github.com/spf13/viper"
)

const (
	ConfigName = "clientconfig"
)

var ConfigPaths = [...]string{
	".",
	"/etc/mediasync",
	"~/.config/mediasync",
}

func GetConfig() (*Configuration, error) {
	viper.SetConfigName(ConfigName)
	for _, cp := range ConfigPaths {
		viper.AddConfigPath(cp)
	}

	err := viper.ReadInConfig()
	if err != nil {
		return &Configuration{}, err
	}

	var c Configuration
	err = viper.Unmarshal(&c)

	if err != nil {
		return &Configuration{}, err
	}

	return &c, nil
}
