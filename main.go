package main

import (
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
	"log"
	"os"
)

// func main() {
// 	file := "backup.yaml"
// 	backup, err := ParseBackupFile(file)
// 	if err != nil {
// 		log.Fatal(errors.Wrapf(err, "failed reading backup definition '%s'", file))
// 	}

// 	err = AnalyzeBackupDefinition(backup)
// 	if err != nil {
// 		log.Fatal(errors.Wrap(err, "failed analyzing backup"))
// 	} else {
// 		log.Println("File format is OK")
// 	}

// 	log.Println(backup)

// 	runner, err := NewBackupRunner(&backup.Backup)
// 	if err != nil {
// 		log.Fatal(errors.Wrap(err, "failed creating backup runner"))
// 	}

// 	err = runner.Run()
// 	if err != nil {
// 		log.Fatal(errors.Wrap(err, "failed running backup"))
// 	}
// }

func RunCmd(file string) error {
	backup, err := ParseBackupFile(file)
	if err != nil {
		return errors.Wrapf(err, "failed reading backup definition '%s'", file)
	}

	err = AnalyzeBackupDefinition(backup)
	if err != nil {
		return errors.Wrap(err, "failed analyzing backup")
	}

	runner, err := NewBackupRunner(&backup.Backup)
	if err != nil {
		return errors.Wrap(err, "failed creating backup runner")
	}

	err = runner.Run()
	if err != nil {
		return errors.Wrap(err, "failed running backup")
	}

	return nil
}

type Options struct {
	DryRun bool
}

var options Options

func GetOptions() Options {
	return options
}

func main() {
	app := &cli.App{
		Name:    "rika",
		Version: "v0.0.1",
		Authors: []*cli.Author{
			&cli.Author{
				Name:  "Gian Sass",
				Email: "gian.sass@outlook.de",
			},
		},
		Copyright: "(c) 2019 Gian Sass",
		Usage:     "run simple declarative backups",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:        "dry-run",
				Usage:       "do not touch anything, only prints commands",
				Destination: &options.DryRun,
			},
		},
		Commands: []*cli.Command{
			{
				Name:      "run",
				Usage:     "runs a backup from a given YAML file",
				ArgsUsage: "[FILE]",
				Action: func(c *cli.Context) error {
					if c.NArg() == 0 {
						return errors.New("run: expected filename")
					}

					file := c.Args().Get(0)
					return RunCmd(file)
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
