/*
   Copyright 2024 The olive Authors

   This program is offered under a commercial and under the AGPL license.
   For AGPL licensing, see below.

   AGPL licensing:
   This program is free software: you can redistribute it and/or modify
   it under the terms of the GNU Affero General Public License as published by
   the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU Affero General Public License for more details.

   You should have received a copy of the GNU Affero General Public License
   along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package console

import (
	"embed"
	"io/fs"
)

//go:embed third_party/swagger
var embedFS embed.FS

// GetSwaggerStatic get fs.FS contains swagger dist
func GetSwaggerStatic() (fs.FS, error) {
	sub, err := fs.Sub(embedFS, "third_party/swagger")
	return sub, err
}

// GetSwaggerHtml reads swagger index.html
func GetSwaggerHtml() ([]byte, error) {
	html, err := embedFS.ReadFile("third_party/swagger/index.html")
	return html, err
}
