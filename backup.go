package main

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/kennygrant/sanitize"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

type CompressionDefinition struct {
	Type string `yaml:"type"`
	Args string `yaml:"args"`
}

func DefaultCompressionDefinition() *CompressionDefinition {
	return &CompressionDefinition{
		Type: "xz",
		Args: "-9",
	}
}

type MySQLDefinition struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
}

type DumpCommand struct {
	Program string
	Args    []string
}

type Database interface {
	//GenerateArtifact(destPath string, artifactName *string) error
	ConstructDumpCommand() DumpCommand
}

type DockerDefinition struct {
	ContainerName string `yaml:"container"`
}

type DatabaseDefinition struct {
	Name   string `yaml:"name"`
	Format string `yaml:"format"`

	Database              Database
	DockerDefinition      *DockerDefinition      `yaml:"docker"`
	MySQLDefinition       *MySQLDefinition       `yaml:"mysql"`
	CompressionDefinition *CompressionDefinition `yaml:"compression"`
}

type VolumeDefinition struct {
	Name                  string                 `yaml:"name"`
	Format                string                 `yaml:"format"`
	Path                  string                 `yaml:"path"`
	CompressionDefinition *CompressionDefinition `yaml:"compression"`
}

type DataProviders struct {
	DatabaseDefinitions []*DatabaseDefinition `yaml:"databases"`
	VolumeDefinitions   []*VolumeDefinition   `yaml:"volumes"`
}

type Storage interface {
	Store(filepath string) error
}

type LocalStorageDefinition struct {
	Format string `yaml:"format"`
	Path   string `yaml:"path"`
}

type SFTPStorageDefinition struct {
	Format string `yaml:"format"`
	User   string `yaml:"user"`
	Host   string `yaml:"host"`
	Path   string `yaml:"path"`
	Port   int    `yaml:"port"`
	Key    string `yaml:"key"`
}

type StorageDefinition struct {
	Name                   string                  `yaml:"name"`
	LocalStorageDefinition *LocalStorageDefinition `yaml:"local"`
	SFTPStorageDefinition  *SFTPStorageDefinition  `yaml:"sftp"`
	Storage                Storage
}

type Backup struct {
	Name               string               `yaml:"name"`
	DataProviders      DataProviders        `yaml:"dataProviders"`
	StorageDefinitions []*StorageDefinition `yaml:"storageProviders"`
}

type BackupDefinition struct {
	Version int    `yaml:"version"`
	Backup  Backup `yaml:"backup"`
}

func ParseBackupFile(path string) (*BackupDefinition, error) {
	file, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, errors.Wrap(err, "reading backup file failed")
	}

	return ParseBackupFromString(string(file))
}

func ParseBackupFromString(s string) (*BackupDefinition, error) {
	var backupDefinition BackupDefinition
	err := yaml.Unmarshal([]byte(s), &backupDefinition)
	if err != nil {
		return nil, errors.Wrap(err, "invalid backup file format")
	}

	return &backupDefinition, nil
}

const VERSION = 1

func analyzeCompressionDefinition(def *CompressionDefinition) error {
	// TODO
	if len(def.Type) == 0 {
		return errors.New("missing type")
	}

	return nil
}

func analyzeMySQLDefinition(def *MySQLDefinition) error {
	if len(def.Host) == 0 {
		return errors.New("missing host")
	}

	if def.Port == 0 {
		return errors.New("missing port")
	}

	if len(def.User) == 0 {
		return errors.New("missing user")
	}

	if len(def.Password) == 0 {
		return errors.New("missing password")
	}

	// NOTE: we do not check Database, because supplying no database means
	// we will dump the entire MySQL database

	return nil
}

func analyzeDatabaseDefinition(def *DatabaseDefinition) error {
	if len(def.Name) == 0 {
		return errors.New("database definition is missing name")
	}

	// TODO: parse format

	if def.CompressionDefinition != nil {
		err := analyzeCompressionDefinition(def.CompressionDefinition)
		if err != nil {
			return errors.Wrap(err, "invalid compression definition")
		}
	} else {
		def.CompressionDefinition = DefaultCompressionDefinition()
	}

	// Only one database must be defined
	databaseDefined := false

	if def.MySQLDefinition != nil {
		err := analyzeMySQLDefinition(def.MySQLDefinition)
		if err != nil {
			return errors.Wrap(err, "invalid MySQL definition")
		}

		def.Database = def.MySQLDefinition

		databaseDefined = true
	}

	if !databaseDefined {
		return errors.New("no database specified")
	}

	return nil
}

func analyzeVolumeDefinition(def *VolumeDefinition) error {
	if len(def.Name) == 0 {
		return errors.New("missing name")
	}

	if len(def.Path) == 0 {
		return errors.New("missing path")
	}

	if _, err := os.Stat(def.Path); err != nil {
		return errors.Wrapf(err, "could not stat %s", def.Path)
	}

	if def.CompressionDefinition != nil {
		err := analyzeCompressionDefinition(def.CompressionDefinition)
		if err != nil {
			return errors.Wrap(err, "invalid compression definition")
		}
	} else {
		def.CompressionDefinition = DefaultCompressionDefinition()
	}

	// TODO: parse format

	return nil
}

func analyzeLocalStorageDefinition(def *LocalStorageDefinition) error {
	if len(def.Path) == 0 {
		return errors.New("missing path")
	}

	if _, err := os.Stat(def.Path); os.IsNotExist(err) {
		err := os.MkdirAll(def.Path, os.ModePerm)
		if err != nil {
			return errors.Wrap(err, "could not create local storage path")
		}
	}

	// TODO: parse format

	return nil
}

func analyzeSFTPStorageDefinition(def *SFTPStorageDefinition) error {
	if len(def.User) == 0 {
		return errors.New("missing user")
	}

	if len(def.Host) == 0 {
		return errors.New("missing host")
	}

	if len(def.Path) == 0 {
		return errors.New("missing remote path")
	}

	// 	if def.Port == 0 {
	// 		return errors.New("missing port")
	// 	}

	if def.Port == 0 {
		def.Port = 22
	}

	return nil
}

func analyzeStorageDefinition(def *StorageDefinition) error {
	if len(def.Name) == 0 {
		return errors.New("missing name")
	}

	if def.Storage == nil && def.LocalStorageDefinition != nil {
		err := analyzeLocalStorageDefinition(def.LocalStorageDefinition)
		if err != nil {
			return errors.Wrapf(err, "invalid local storage definition")
		}

		def.Storage = def.LocalStorageDefinition
	}

	if def.Storage == nil && def.SFTPStorageDefinition != nil {
		err := analyzeSFTPStorageDefinition(def.SFTPStorageDefinition)
		if err != nil {
			return errors.Wrapf(err, "invalid SFTP storage definition")
		}

		def.Storage = def.SFTPStorageDefinition
	}

	// TODO: parse more storage definitions

	return nil
}

func AnalyzeBackupDefinition(def *BackupDefinition) error {
	if def.Version != VERSION {
		return errors.New("invalid version")
	}

	backup := def.Backup
	if len(backup.Name) == 0 {
		return errors.New("backup is missing name")
	}

	if len(backup.DataProviders.DatabaseDefinitions) == 0 && len(backup.DataProviders.VolumeDefinitions) == 0 {
		return errors.New("you have neither specified a database or a volume: there is nothing to back up!")
	}

	for _, databaseDefinition := range backup.DataProviders.DatabaseDefinitions {
		err := analyzeDatabaseDefinition(databaseDefinition)
		if err != nil {
			return errors.Wrapf(err, "database '%s' has invalid definition", databaseDefinition.Name)
		}
	}

	for _, volumeDefinition := range backup.DataProviders.VolumeDefinitions {
		err := analyzeVolumeDefinition(volumeDefinition)
		if err != nil {
			return errors.Wrapf(err, "volume '%s' has invalid definition", volumeDefinition.Name)
		}
	}

	for _, storageDefinition := range backup.StorageDefinitions {
		err := analyzeStorageDefinition(storageDefinition)
		if err != nil {
			return errors.Wrapf(err, "storage '%s' has invalid definition", storageDefinition.Name)
		}
	}

	return nil
}

func DefaultFileFormat(name string) string {
	s := strings.ToLower(name)
	s = strings.ReplaceAll(s, " ", "_")
	s = sanitize.Name(s)
	return s
}

func GetFormattedName(name string, format string) string {
	if len(format) == 0 {
		return DefaultFileFormat(name)
	}

	// TODO:
	return name
}

type BackupRunner struct {
	TempPath string
	Backup   *Backup
	Time     time.Time
}

func NewBackupRunner(Backup *Backup) (*BackupRunner, error) {
	tmpPath, err := ioutil.TempDir("/tmp", "rika-backup")
	if err != nil {
		return nil, err
	}

	return &BackupRunner{
		Backup:   Backup,
		TempPath: tmpPath,
		Time:     time.Now(),
	}, nil
}

func (runner *BackupRunner) GetTimestampString() string {
	return runner.Time.Format("20060102150405")
}
func (def *MySQLDefinition) ConstructDumpCommand() DumpCommand {
	const program = "mysqldump"
	var args []string

	if len(def.Database) == 0 {
		args = []string{"-h", def.Host, "-u", def.User, fmt.Sprintf("--password=%s", def.Password), "-P", strconv.Itoa(def.Port), "--all-databases"}
	} else {
		args = []string{"-h", def.Host, "-u", def.User, fmt.Sprintf("--password=%s", def.Password), "-P", strconv.Itoa(def.Port), def.Database}
	}

	return DumpCommand{
		Program: program,
		Args:    args,
	}
}

func DumpCommandToOSCommand(dumpCmd DumpCommand, def *DatabaseDefinition) *exec.Cmd {
	if def.DockerDefinition == nil {
		return exec.Command(dumpCmd.Program, dumpCmd.Args...)
	} else {
		// prepend docker command
		dockerArgs := []string{
			"exec",
			"-t",
			def.DockerDefinition.ContainerName,
			dumpCmd.Program,
		}

		for _, arg := range dumpCmd.Args {
			dockerArgs = append(dockerArgs, arg)
		}

		return exec.Command("docker", dockerArgs...)
	}
}

func (runner *BackupRunner) ConstructArtifactName(name, format, filetype, compressionType string) string {
	if len(format) == 0 {
		name = DefaultFileFormat(name)
	}

	return fmt.Sprintf("%s-%s.%s.%s", name, runner.GetTimestampString(), filetype, compressionType)
}

func RunCommandWithCompressedStdout(cmd *exec.Cmd, cdef *CompressionDefinition, destPath string) error {
	args := []string{"-z", "--stdout"}

	for _, additionalArg := range strings.Fields(cdef.Args) {
		args = append(args, additionalArg)
	}

	compressCmd := exec.Command(cdef.Type, args...)
	log.Printf("Running '%v' and compressing with '%v'", cmd, compressCmd)

	if GetOptions().DryRun {
		return nil
	}

	compressCmd.Stderr = os.Stderr
	cmd.Stderr = os.Stderr

	var err error
	compressCmd.Stdin, err = cmd.StdoutPipe()
	if err != nil {
		return err
	}

	compressStdout, err := compressCmd.StdoutPipe()
	if err != nil {
		return err
	}

	outfile, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer outfile.Close()

	err = compressCmd.Start()
	if err != nil {
		return errors.Wrap(err, "failed to run compression cmd")
	}

	err = cmd.Run()
	if err != nil {
		return errors.Wrap(err, "failed to run cmd")
	}

	fileWriter := bufio.NewWriter(outfile)
	go io.Copy(fileWriter, compressStdout)
	defer fileWriter.Flush()

	return compressCmd.Wait()
}

func (runner *BackupRunner) GenerateDatabaseArtifact(def *DatabaseDefinition, destPath string, artifactName *string) error {
	dumpCmd := def.Database.ConstructDumpCommand()

	fileName := runner.ConstructArtifactName(def.Name, def.Format, "sql", def.CompressionDefinition.Type)
	*artifactName = fileName
	fullPath := path.Join(destPath, fileName)

	osCmd := DumpCommandToOSCommand(dumpCmd, def)

	return RunCommandWithCompressedStdout(osCmd, def.CompressionDefinition, fullPath)
}

func (runner *BackupRunner) GenerateVolumeArtifact(def *VolumeDefinition, destPath string, artifactName *string) error {
	if def.CompressionDefinition.Type == "none" {
		// Simple tar creation
		fileName := runner.ConstructArtifactName(def.Name, def.Format, "tar", def.CompressionDefinition.Type)
		fullPath := path.Join(destPath, fileName)
		cmd := exec.Command("tar", "cvf", fullPath, def.Path)
		*artifactName = fileName
		log.Println(cmd)

		if !GetOptions().DryRun {
			return cmd.Run()
		}

		return nil
	} else {
		tarCmd := exec.Command("tar", "cvf", "-", def.Path)

		fileName := runner.ConstructArtifactName(def.Name, def.Format, "tar", def.CompressionDefinition.Type)
		*artifactName = fileName
		fullPath := path.Join(destPath, fileName)

		return RunCommandWithCompressedStdout(tarCmd, def.CompressionDefinition, fullPath)
	}
}

func copy(src, dst string) (int64, error) {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return 0, err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return 0, fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer destination.Close()
	nBytes, err := io.Copy(destination, source)
	return nBytes, err
}

func (local *LocalStorageDefinition) Store(fullpath string) error {
	artifact := filepath.Base(fullpath)
	destFullPath := path.Join(local.Path, artifact)

	log.Printf("copying %s to %s\n", fullpath, destFullPath)

	if !GetOptions().DryRun {
		_, err := copy(fullpath, destFullPath)
		return err
	}

	return nil
}

func (sftp *SFTPStorageDefinition) Store(fullpath string) error {
	artifact := filepath.Base(fullpath)

	var args []string

	if len(sftp.Key) > 0 {
		args = []string{fmt.Sprintf("-P%d", sftp.Port), "-i", sftp.Key, fullpath, fmt.Sprintf("%s@%s:%s/%s", sftp.User, sftp.Host, sftp.Path, artifact)}
	} else {

		args = []string{fmt.Sprintf("-P%d", sftp.Port), fullpath, fmt.Sprintf("%s@%s:%s/%s", sftp.User, sftp.Host, sftp.Path, artifact)}
	}

	cmd := exec.Command("scp", args...)

	log.Println(cmd)

	if !GetOptions().DryRun {
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
		return cmd.Run()
	}

	return nil
}

func (runner *BackupRunner) Run() error {
	log.Printf("Running backup '%s'\n", runner.Backup.Name)

	defer os.RemoveAll(runner.TempPath)

	var artifacts []string

	log.Println("Generating database artifacts")
	for _, db := range runner.Backup.DataProviders.DatabaseDefinitions {
		var artifactName string

		err := runner.GenerateDatabaseArtifact(db, runner.TempPath, &artifactName)
		if err != nil {
			return err
		}

		log.Printf("Generated '%s'\n", artifactName)
		if len(artifactName) > 0 {
			artifacts = append(artifacts, artifactName)
		}
	}

	log.Println("Generating volume artifacts")
	for _, volume := range runner.Backup.DataProviders.VolumeDefinitions {
		var artifactName string

		err := runner.GenerateVolumeArtifact(volume, runner.TempPath, &artifactName)
		if err != nil {
			return err
		}

		log.Printf("Generated '%s'\n", artifactName)
		if len(artifactName) > 0 {
			artifacts = append(artifacts, artifactName)

		}
	}

	// Store artifacts
	for _, storage := range runner.Backup.StorageDefinitions {
		log.Printf("Storing artifacts in provider '%s'", storage.Name)

		for _, artifact := range artifacts {
			artifactFullPath := path.Join(runner.TempPath, artifact)
			err := storage.Storage.Store(artifactFullPath)
			if err != nil {
				return err
			}
		}
	}

	log.Printf("Backup '%s' has completed\n", runner.Backup.Name)
	return nil
}
