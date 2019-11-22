package main

import (
	"bufio"
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
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

	"github.com/briandowns/spinner"
)

var log = logrus.New()

func SetVerbose() {
	log.SetLevel(logrus.DebugLevel)
}

func CreateSpinner() *spinner.Spinner {
	return spinner.New(spinner.CharSets[0], 100*time.Millisecond)
}

type CompressionDefinition struct {
	Command   string `yaml:"cmd"`
	Extension string `yaml:"ext"`
	Args      string `yaml:"args"`
}

func DefaultCompressionDefinition() *CompressionDefinition {
	return &CompressionDefinition{
		Command:   "xz",
		Extension: "xz",
	}
}

type MySQLDefinition struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
}

type PostgreSQLDefinition struct {
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
	PostgreSQLDefinition  *PostgreSQLDefinition  `yaml:"postgres"`
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

func which(file string) (string, error) {
	bytes, err := exec.Command("which", file).Output()

	path := string(bytes)
	path = strings.TrimSuffix(path, "\n")

	return string(path), err

	// 	var stdout bytes.Buffer
	// 	cmd.Stdout = &stdout

	// 	err := cmd.Run()
	// 	if err != nil {
	// 		return "", err
	// 	}
}

func analyzeCompressionDefinition(def *CompressionDefinition) error {
	// TODO
	if len(def.Command) == 0 {
		return errors.New("missing command")
	}

	if len(def.Extension) == 0 {
		def.Extension = def.Command
	}

	path, err := which(def.Command)
	if err != nil {
		// TODO: return stderr
		return errors.Wrap(err, "could not find executable")
	}

	def.Command = path

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

func analyzePostgreSQLDefinition(def *PostgreSQLDefinition) error {
	if len(def.Host) == 0 {
		return errors.New("missing host")
	}

	if def.Port == 0 {
		return errors.New("missing port")
	}

	if len(def.User) == 0 {
		return errors.New("missing user")
	}

	// 	if len(def.Password) == 0 {
	// 		return errors.New("missing password")
	// 	}

	// NOTE: we do not check Database, because supplying no database means
	// we will dump the entire Postgres database

	return nil
}

func (def *DatabaseDefinition) SetPrimaryDatabase(db Database) error {
	if def.Database != nil {
		return errors.New("cannot define multiple databases")
	}

	def.Database = db
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

	if def.MySQLDefinition != nil {
		err := analyzeMySQLDefinition(def.MySQLDefinition)
		if err != nil {
			return errors.Wrap(err, "invalid MySQL definition")
		}

		def.SetPrimaryDatabase(def.MySQLDefinition)
	}

	if def.PostgreSQLDefinition != nil {
		err := analyzePostgreSQLDefinition(def.PostgreSQLDefinition)
		if err != nil {
			return errors.Wrap(err, "invalid PostgreSQL definition")
		}

		def.SetPrimaryDatabase(def.PostgreSQLDefinition)
	}

	if def.Database == nil {
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
		return err
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
	tmpPath, err := ioutil.TempDir("/tmp", "rika")
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

	args := []string{"-h", def.Host, "-u", def.User, "-P", strconv.Itoa(def.Port)}

	if len(def.Password) > 0 {
		args = append(args, fmt.Sprintf("--password=%s", def.Password))
	}

	if len(def.Database) == 0 {
		args = append(args, "--all-databases")
	} else {
		args = append(args, def.Database)
	}

	return DumpCommand{
		Program: program,
		Args:    args,
	}
}

func (def *PostgreSQLDefinition) ConstructDumpCommand() DumpCommand {
	var program string

	args := []string{"-h", def.Host, "-U", def.User, "-p", strconv.Itoa(def.Port)}

	// FIXME: no passwords supported

	if len(def.Database) == 0 {
		program = "pg_dumpall"
	} else {
		program = "pg_dump"
		args = append(args, def.Database)
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
	args := []string{"--stdout"}

	for _, additionalArg := range strings.Fields(cdef.Args) {
		args = append(args, additionalArg)
	}

	compressCmd := exec.Command(cdef.Command, args...)

	log.WithFields(logrus.Fields{
		"cmd":         cmd,
		"compressCmd": compressCmd,
	}).Debug("Running command and compressing")

	if GetOptions().DryRun {
		return nil
	}

	compressCmd.Stderr = os.Stderr

	if GetOptions().Verbose {
		cmd.Stderr = os.Stderr
	}
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

	fileName := runner.ConstructArtifactName(def.Name, def.Format, "sql", def.CompressionDefinition.Extension)
	*artifactName = fileName
	fullPath := path.Join(destPath, fileName)

	osCmd := DumpCommandToOSCommand(dumpCmd, def)

	return RunCommandWithCompressedStdout(osCmd, def.CompressionDefinition, fullPath)
}

func (runner *BackupRunner) GenerateVolumeArtifact(def *VolumeDefinition, destPath string, artifactName *string) error {
	if def.CompressionDefinition.Command == "none" {
		// Simple tar creation
		fileName := runner.ConstructArtifactName(def.Name, def.Format, "tar", def.CompressionDefinition.Extension)
		fullPath := path.Join(destPath, fileName)
		cmd := exec.Command("tar", "cvf", fullPath, def.Path)
		*artifactName = fileName
		log.Infoln(cmd)

		if !GetOptions().DryRun {
			return cmd.Run()
		}

		return nil
	} else {
		tarCmd := exec.Command("tar", "cvf", "-", def.Path)

		fileName := runner.ConstructArtifactName(def.Name, def.Format, "tar", def.CompressionDefinition.Extension)
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

	//log.Debugf("Local: Copying %s to %s\n", fullpath, destFullPath)

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

	//log.Debugln(cmd)

	if !GetOptions().DryRun {
		// if GetOptions().Verbose {
		// 	cmd.Stderr = os.Stderr
		// 	cmd.Stdout = os.Stdout
		// }
		return cmd.Run()
	}

	return nil
}

func (runner *BackupRunner) Run() error {
	log.WithFields(logrus.Fields{
		"backup": runner.Backup.Name,
	}).Info("Started backup")

	defer os.RemoveAll(runner.TempPath)

	var artifacts []string
	spinner := CreateSpinner()

	log.Infof("Generating database artifacts ")
	spinner.Start()
	for _, db := range runner.Backup.DataProviders.DatabaseDefinitions {
		var artifactName string

		err := runner.GenerateDatabaseArtifact(db, runner.TempPath, &artifactName)
		if err != nil {
			return err
		}

		log.WithFields(logrus.Fields{
			"name": artifactName,
		}).Debug("Generated artifact")

		if len(artifactName) > 0 {
			artifacts = append(artifacts, artifactName)
		}
	}

	spinner.Stop()

	log.Info("Generating volume artifacts")
	spinner.Start()
	for _, volume := range runner.Backup.DataProviders.VolumeDefinitions {
		var artifactName string

		err := runner.GenerateVolumeArtifact(volume, runner.TempPath, &artifactName)
		if err != nil {
			return err
		}

		log.WithFields(logrus.Fields{
			"name": artifactName,
		}).Debug("Generated artifact")

		if len(artifactName) > 0 {
			artifacts = append(artifacts, artifactName)

		}
	}

	spinner.Stop()

	// Store artifacts
	for _, storage := range runner.Backup.StorageDefinitions {
		log.WithFields(logrus.Fields{
			"provider": storage.Name,
		}).Info("Storing artifacts")

		spinner.Start()

		for _, artifact := range artifacts {
			artifactFullPath := path.Join(runner.TempPath, artifact)
			err := storage.Storage.Store(artifactFullPath)
			if err != nil {
				return errors.Wrapf(err, "failed storing %s", artifact)
			}
		}

		spinner.Stop()
	}

	log.WithFields(logrus.Fields{
		"backup": runner.Backup.Name,
	}).Info("Backup has finished")

	return nil
}
