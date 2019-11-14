package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDefaultFileFormat(t *testing.T) {
	const name = "WordPress Database"
	formatted := DefaultFileFormat(name)

	assert.Equal(t, formatted, "wordpress-database", "The formatted file name should match the expected string")
}

func parseAndAnalyze(s string) error {
	backup, err := ParseBackupFromString(s)
	if err != nil {
		return err
	}
	return AnalyzeBackupDefinition(backup)
}

func TestBackupDefinition(t *testing.T) {
	const file = `
version: 1
backup:
  name: Site Backup
  dataProviders:
    databases:
    - name: WordPress Database
      format: wp-{date}
      mysql:
        host: localhost
        port: 5432
        user: wordpress
        password: wordpress
      compression:
        type: xz
        args: -9
    volumes:
    - name: WordPress Uploads
      path: /home/nuke/uploads
      compression:
        type: gz
  storageProviders:
  - name: Test
    local:
      path: /home/nuke/backups
`

	err := parseAndAnalyze(file)
	assert.Nil(t, err)
}
