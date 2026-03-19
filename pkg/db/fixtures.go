// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package db

import (
	"path/filepath"
	"runtime"

	testfixtures "github.com/go-testfixtures/testfixtures/v3"
)

// InitFixtures loads test fixtures into the database.
// If tablenames are provided, only those tables are loaded.
func InitFixtures(tablenames ...string) error {
	_, filename, _, _ := runtime.Caller(0)
	fixturesDir := filepath.Join(filepath.Dir(filename), "fixtures")

	opts := []func(*testfixtures.Loader) error{
		testfixtures.Database(x.DB().DB),
		testfixtures.Dialect(x.DriverName()),
	}

	if len(tablenames) > 0 {
		files := make([]string, len(tablenames))
		for i, t := range tablenames {
			files[i] = filepath.Join(fixturesDir, t+".yml")
		}
		opts = append(opts, testfixtures.Files(files...))
	} else {
		opts = append(opts, testfixtures.Directory(fixturesDir))
	}

	loader, err := testfixtures.New(opts...)
	if err != nil {
		return err
	}

	return loader.Load()
}
