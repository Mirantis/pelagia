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
	"crypto/sha256"
	"fmt"
	"sort"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"k8s.io/apimachinery/pkg/api/resource"
)

func Contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

var GetCurrentTimeString = getCurrentTimeString
var GetCurrentUnixTimeString = getCurrentUnixTimeString

func getCurrentTimeString() string {
	return time.Now().Format(time.RFC3339)
}

func getCurrentUnixTimeString() string {
	return fmt.Sprintf("%v", time.Now().Unix())
}

func RunFuncWithRetry(times int, interval time.Duration, funcToRun func() (interface{}, error)) (interface{}, error) {
	tries := 0
	var err error
	var output interface{}
	for tries < times {
		output, err = funcToRun()
		if err == nil {
			return output, nil
		}
		tries++
		if tries < times {
			time.Sleep(interval)
		}
	}
	return output, errors.Wrapf(err, "Retries (%d/%d) exceeded", tries, times)
}

func ShowObjectDiff(l zerolog.Logger, oldObject, newObject interface{}) {
	oldObjectType := fmt.Sprintf("%T", oldObject)
	newObjectType := fmt.Sprintf("%T", newObject)
	if oldObjectType != newObjectType {
		l.Error().Msgf("can't compare two different object types: %s and %s", oldObjectType, newObjectType)
		return
	}
	resourceQtyComparer := cmp.Comparer(func(x, y resource.Quantity) bool { return x.Cmp(y) == 0 })
	diff := cmp.Diff(oldObject, newObject, resourceQtyComparer)
	if diff != "" {
		l.Trace().Msgf("object %s has changed, diff:\n%s", oldObjectType, diff)
	}
}

func SortedMapKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func GetStringSha256(s string) string {
	h := sha256.New()
	h.Write([]byte(s))
	bs := h.Sum(nil)
	return fmt.Sprintf("%x", bs)
}
