# Versioned migrations for MongoDB
[![Build Status](https://travis-ci.org/xakep666/mongo-migrate.svg?branch=master)](https://travis-ci.org/xakep666/mongo-migrate)
[![codecov](https://codecov.io/gh/xakep666/mongo-migrate/branch/master/graph/badge.svg)](https://codecov.io/gh/xakep666/mongo-migrate)
[![Go Report Card](https://goreportcard.com/badge/github.com/xakep666/mongo-migrate)](https://goreportcard.com/report/github.com/xakep666/mongo-migrate)
[![GoDoc](https://godoc.org/github.com/xakep666/mongo-migrate?status.svg)](https://godoc.org/github.com/xakep666/mongo-migrate)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

This package allows to perform versioned migrations on your MongoDB using [mongo-go-driver](https://github.com/mongodb/mongo-go-driver).
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
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	migrate "github.com/xakep666/mongo-migrate"
)

func init() {
	migrate.Register(func(db *mongo.Database) error {
		opt := options.Index().SetName("my-index")
		keys := bson.D{{"my-key", 1}}
		model := mongo.IndexModel{Keys: keys, Options: opt}
		_, err := db.Collection("my-coll").Indexes().CreateOne(context.TODO(), model)
		if err != nil {
			return err
		}

		return nil
	}, func(db *mongo.Database) error {
		_, err := db.Collection("my-coll").Indexes().DropOne(context.TODO(), "my-index")
		if err != nil {
			return err
		}
		return nil
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
func MongoConnect(host, user, password, database string) (*mongo.Database, error) {
	uri := fmt.Sprintf("mongodb://%s:%s@%s:27017", user, password, host)
	opt := options.Client().ApplyURI(uri)
	client, err := mongo.NewClient(opt)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	err = client.Connect(ctx)
	if err != nil {
		return nil, err
	}
	db = client.Database(database)
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
func MongoConnect(host, user, password, database string) (*mongo.Database, error) {
	uri := fmt.Sprintf("mongodb://%s:%s@%s:27017", user, password, host)
	opt := options.Client().ApplyURI(uri)
	client, err := mongo.NewClient(opt)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	err = client.Connect(ctx)
	if err != nil {
		return nil, err
	}
	db = client.Database(database)
	m := migrate.NewMigrate(db, migrate.Migration{
		Version: 1,
		Description: "add my-index",
		Up: func(db *mongo.Database) error {
			opt := options.Index().SetName("my-index")
			keys := bson.D{{"my-key", 1}}
			model := mongo.IndexModel{Keys: keys, Options: opt}
			_, err := db.Collection("my-coll").Indexes().CreateOne(context.TODO(), model)
			if err != nil {
				return err
			}

			return nil
		},
		Down: func(db *mongo.Database) error {
			_, err := db.Collection("my-coll").Indexes().DropOne(context.TODO(), "my-index")
			if err != nil {
				return err
			}
			return nil
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
