/*
	Copyright (c) 2016, Percona LLC and/or its affiliates. All rights reserved.

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU Affero General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU Affero General Public License for more details.

	You should have received a copy of the GNU Affero General Public License
	along with this program.  If not, see <http://www.gnu.org/licenses/>
*/

package postgresql

import (
	"context"
	"testing"
	"time"

	"github.com/percona/pmm-client/pmm/plugin"
	"github.com/stretchr/testify/assert"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
)

func TestMakeGrants(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("error opening a stub database connection: %s", err)
	}
	defer db.Close()

	{
		columns := []string{"exists"}
		rows := sqlmock.NewRows(columns)
		mock.ExpectQuery("SELECT 1 FROM pg_roles WHERE rolname = \\$1").WithArgs("root").WillReturnRows(rows)
	}

	{
		columns := []string{"exists"}
		rows := sqlmock.NewRows(columns)
		mock.ExpectQuery("SELECT 1 FROM pg_roles WHERE rolname = \\$1").WithArgs("admin").WillReturnRows(rows)
	}

	type sample struct {
		dsn    DSN
		grants []Exec
	}
	samples := []sample{
		{
			dsn: DSN{User: "root", Password: "abc123"},
			grants: []Exec{
				{
					Query: "CREATE USER $1 PASSWORD $2",
					Args:  []interface{}{"root", "abc123"},
				},
				{
					Query: "ALTER USER $1 SET SEARCH_PATH TO $1,pg_catalog",
					Args:  []interface{}{"root"},
				},
				{
					Query: "CREATE SCHEMA $1 AUTHORIZATION $1",
					Args:  []interface{}{"root"},
				},
				{
					Query: "CREATE VIEW $1.pg_stat_activity AS SELECT * from pg_catalog.pg_stat_activity",
					Args:  []interface{}{"root"},
				},
				{
					Query: "GRANT SELECT $1.pg_stat_activity TO $1",
					Args:  []interface{}{"root"},
				},
				{
					Query: "CREATE VIEW $1.pg_stat_replication AS SELECT * from pg_catalog.pg_stat_replication",
					Args:  []interface{}{"root"},
				},
				{
					Query: "GRANT SELECT ON $1.pg_stat_replication TO $1",
					Args:  []interface{}{"root"},
				},
			},
		},
		{
			dsn: DSN{User: "admin", Password: "23;,_-asd"},
			grants: []Exec{
				{
					Query: "CREATE USER $1 PASSWORD $2",
					Args:  []interface{}{"admin", "23;,_-asd"},
				},
				{
					Query: "ALTER USER $1 SET SEARCH_PATH TO $1,pg_catalog",
					Args:  []interface{}{"admin"},
				},
				{
					Query: "CREATE SCHEMA $1 AUTHORIZATION $1",
					Args:  []interface{}{"admin"},
				},
				{
					Query: "CREATE VIEW $1.pg_stat_activity AS SELECT * from pg_catalog.pg_stat_activity",
					Args:  []interface{}{"admin"},
				},
				{
					Query: "GRANT SELECT $1.pg_stat_activity TO $1",
					Args:  []interface{}{"admin"},
				},
				{
					Query: "CREATE VIEW $1.pg_stat_replication AS SELECT * from pg_catalog.pg_stat_replication",
					Args:  []interface{}{"admin"},
				},
				{
					Query: "GRANT SELECT ON $1.pg_stat_replication TO $1",
					Args:  []interface{}{"admin"},
				},
			},
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	for _, s := range samples {
		grants, err := makeGrants(ctx, db, s.dsn)
		assert.NoError(t, err)
		assert.Equal(t, s.grants, grants)
	}
}

func TestGetInfo(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("error opening a stub database connection: %s", err)
	}
	defer db.Close()

	columns := []string{"@@hostname", "@@port", "@@version"}
	rows := sqlmock.NewRows(columns).AddRow("db01", "3306", "1.2.3")
	mock.ExpectQuery(`SELECT inet_server_addr\(\), inet_server_port\(\), version\(\)`).WillReturnRows(rows)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	info, err := getInfo(ctx, db)
	assert.NoError(t, err)
	expected := plugin.Info{
		Hostname: "db01",
		Port:     "3306",
		Distro:   "PostgreSQL",
		Version:  "1.2.3",
	}
	assert.Equal(t, expected, *info)

	// Ensure all SQL queries were executed
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}
