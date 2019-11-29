# Rika

![Build Status](https://travis-ci.org/Nuke928/rika.svg?branch=master)

Simple but powerful declarative backups.


Rika allows you to define all of your data providers (databases and volumes) and storage providers (SFTP servers, local directories, etc.) conveniently in one file.

## Example

Locally we have a MySQL Database and one volume, and we want to back thse up on a SFTP server.

```yaml
version: 1
backup:
  name: Site Backup
  dataProviders:
    databases:
    - name: MySQL
      mysql:
        host: localhost
        port: 3306
        user: test
        password: test
        database: test
      compression:
        cmd: gzip
        ext: gz
    volumes:
    - name: WordPress Uploads
      path: /var/www/blog/uploads
  storageProviders:
  - name: Hetzner Storage Box
    sftp:
      user: u215873
      host: u215873.your-storagebox.de
      path: wordpress
      port: 23
```

Run with `rika run backup.yaml`

As a result the following two artifacts are generated and stored on the SFTP server.

* wordpress-uploads-yyyymmddHHMMSS.tar.xz
* wordpress-database-yyyymmddHHMMSS.sql.gz

## Using Docker

If you're running your database inside a Docker container, there's a way for Rika to run dump commands inside the container as well:

```yaml
databases:
- name: MySQL
  docker:
    container: test-mysql
  mysql:
    host: localhost
    port: 3306
    user: test
    password: test
    database: test
```

## Supported data providers

* MySQL
* PostgreSQL

More coming soon!

## Supported storage providers

* Local
* SFTP

## Compression

The `compression` key allows you to tune compression.

```
compression:
  cmd: bzip2
  args: -9
```
