/*
Copyright 2025 Mirantis IT.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package lcmcommon

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

const LoggerObjectField = "object"

func InitLogger(objectField bool) zerolog.Logger {
	zerolog.TimeFieldFormat = time.StampMicro
	output := zerolog.ConsoleWriter{Out: os.Stdout, NoColor: true}
	if objectField {
		// preserve default and use custom object field
		output.PartsOrder = []string{
			zerolog.TimestampFieldName,
			zerolog.LevelFieldName,
			zerolog.CallerFieldName,
			LoggerObjectField,
			zerolog.MessageFieldName,
		}
		output.FieldsExclude = []string{LoggerObjectField}
	}
	output.TimeLocation = time.Local
	output.FormatTimestamp = func(i interface{}) string {
		return fmt.Sprintf("%-6s |", i)
	}
	output.FormatLevel = func(i interface{}) string {
		return strings.ToUpper(fmt.Sprintf("%-4s |", i))
	}
	output.FormatCaller = func(i interface{}) string {
		if v, ok := i.(string); ok {
			return fmt.Sprintf("%-6s |", filepath.Base(v))
		}
		// just stub - should not happen
		return ""
	}
	// default log level - trace
	return zerolog.New(output).With().Timestamp().Caller().Logger()
}
