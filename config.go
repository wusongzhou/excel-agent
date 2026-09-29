/*
 * Copyright 2025 CloudWeGo Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

// loadDotEnv reads a project-local .env file so configuration can live in
// the working copy instead of the user environment. Variables already set in
// the environment take precedence, so .env never overrides an explicit
// setting. See .env.example for the supported keys.
func loadDotEnv(paths ...string) {
	if len(paths) == 0 {
		dirs := []string{"."}
		if exe, err := os.Executable(); err == nil {
			dirs = append(dirs, filepath.Dir(exe))
		}
		for i, dir := range dirs {
			dirs[i] = filepath.Join(dir, ".env")
		}
		paths = dirs
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err != nil {
			continue
		}
		if err := godotenv.Load(path); err != nil {
			log.Printf("[config] failed to load %s: %v", path, err)
		}
		return
	}
}
