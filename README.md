# Versioned migrations for MongoDB
[![Build Status](https://travis-ci.org/xakep666/mongo-migrate.svg?branch=master)](https://travis-ci.org/xakep666/mongo-migrate)
[![codecov](https://codecov.io/gh/xakep666/mongo-migrate/branch/master/graph/badge.svg)](https://codecov.io/gh/xakep666/mongo-migrate)
[![Go Report Card](https://goreportcard.com/badge/github.com/xakep666/mongo-migrate)](https://goreportcard.com/report/github.com/xakep666/mongo-migrate)
[![GoDoc](https://godoc.org/github.com/xakep666/mongo-migrate?status.svg)](https://godoc.org/github.com/xakep666/mongo-migrate)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

This package allows to perform versioned migrations on your MongoDB using [mgo driver](https://github.com/globalsign/mgo).
It depends only on standard library and mgo driver.
Inspired by [go-pg migrations](https://github.com/go-pg/migrations).

Table of Contents
=================

* [Prerequisites](#prerequisites)
* [Installation](#installation)
* [Usage](#usage)
  * [Use case \#1\. Migrations in files\.](#use-case-1-migrations-in-files)
  * [Use case \#2\. Migrations in application code\.](#use-case-2-migrations-in-application-code)
* [How it works?](#how-it-works)
* [License](#license)

## Prerequisites
* Golang >= 1.10 or Vgo

## Installation
```bash
go get -v -u github.com/xakep666/mongo-migrate
```

## Usage
### Use case #1. Migrations in files.

* Create a package with migration files.
File name should be like `<version>_<description>.go`.

`1_add-my-index.go`

```go
package migrations

import (
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	migrate "github.com/xakep666/mongo-migrate"
)

func init() {
	migrate.Register(func(db *mgo.Database) error {
		return db.C("my-coll").Insert(EnsureIndex(mgo.Index{Name: "my-index", Key: []string{"my-key"}}))
	}, func(db *mgo.Database) error {
		return db.C("my-coll").DropIndexName("my-index")
	})
}
```

* Import it in your application.
```go
import (
    ...
    migrate "github.com/xakep666/mongo-migrate"
    _ "path/to/migrations_package" // database migrations
    ...
)
```

* Run migrations.
```go
func MongoConnect(host, user, password, database string) (*mgo.Database, error) {
    session, err := mgo.DialWithInfo(&mgo.DialInfo{
        Addrs: []string{host},
        Database: database,
        Username: user,
        Password: password,
    })
    if err != nil {
        return nil, err
    }
    db := session.DB("")
    migrate.SetDatabase(db)
    if err := migrate.Up(migrate.AllAvailable); err != nil {
        return nil, err
    }
    return db, nil
}
```

### Use case #2. Migrations in application code.
* Just define it anywhere you want and run it.
```go
func MongoConnect(host, user, password, database string) (*mgo.Database, error) {
    session, err := mgo.DialWithInfo(&mgo.DialInfo{
        Addrs: []string{host},
        Database: database,
        Username: user,
        Password: password,
    })
    if err != nil {
        return nil, err
    }
    db := session.DB("")
    m := migrate.NewMigrate(db, migrate.Migration{
        Version: 1,
        Description: "add my-index",
        Up: func(db *mgo.Database) error {
            return db.C("my-coll").EnsureIndex(mgo.Index{Name: "my-index", Key: []string{"my-key"}})
        },
        Down: func(db *mgo.Database) error {
            return db.C("my-coll").DropIndexName("my-index")
        },
    })
    if err := m.Up(migrate.AllAvailable); err != nil {
        return nil, err
    }
    return db, nil
}
```

## How it works?
This package creates a special collection (by default it`s name is "migrations") for versioning.
In this collection stored documents like
```json
{
    "_id": "<mongodb-generated id>",
    "version": 1,
    "description": "add my-index",
    "timestamp": "<when applied>"
}
```
Current database version determined as version from latest inserted document.

You can change collection name using `SetMigrationsCollection` methods.
Remember that if you want to use custom collection name you need to set it before running migrations.

## License
mongo-migrate project is licensed under the terms of the MIT license. Please see LICENSE in this repository for more details.
