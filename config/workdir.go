package config

import "os"

func initWorkDir() {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	WORKDIR = wd
}
