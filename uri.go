// Copyright 2018 Jacob Dufault
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import "strings"

func pathToURI(absolutePath string) LsDocumentURI {
	m := map[rune](string){
		' ': "%20",
		'#': "%23",
		'$': "%24",
		'&': "%26",
		'(': "%28",
		')': "%29",
		'+': "%2B",
		',': "%2C",
		':': "%3A",
		';': "%3B",
		'?': "%3F",
		'@': "%40",
	}

	result := ""
	for _, c := range absolutePath {
		if _, has := m[c]; has {
			result += m[c]
		} else {
			result += string(c)
		}
	}

	return LsDocumentURI("file://" + strings.Replace(result, "\\", "/", -1))
}
