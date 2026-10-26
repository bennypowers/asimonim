/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GNU General Public License
as published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
See the LICENSE file for details.
*/

// Command asimonim parses and works with DTCG design tokens.
package main

import (
	"os"

	"bennypowers.dev/asimonim/cmd"
	_ "bennypowers.dev/asimonim/cmd/lsp" // registers lsp command when CGO is available
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
