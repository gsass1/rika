package main

import (
	"github.com/pkg/errors"
	"log"
)

func main() {
	file := "backup.yaml"
	backup, err := ParseBackupFile(file)
	if err != nil {
		log.Fatal(errors.Wrapf(err, "failed reading backup definition '%s'", file))
	}

	err = AnalyzeBackupDefinition(backup)
	if err != nil {
		log.Fatal(errors.Wrap(err, "failed analyzing backup"))
	} else {
		log.Println("File format is OK")
	}

	log.Println(backup)

	runner, err := NewBackupRunner(&backup.Backup)
	if err != nil {
		log.Fatal(errors.Wrap(err, "failed creating backup runner"))
	}

	err = runner.Run()
	if err != nil {
		log.Fatal(errors.Wrap(err, "failed running backup"))
	}
}
