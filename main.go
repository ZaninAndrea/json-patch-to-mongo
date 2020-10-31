package jsonpatchtomongo

// code converted from js to golang, the original source is https://github.com/mongodb-js/jsonpatch-to-mongodb

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func main() {
	patches := []byte(`[
  		{ "op": "add", "path": "/hello/0/hi/5", "value": 1 },
  		{ "op": "add", "path": "/hello/0/hi/5", "value": 2 },
  		{ "op": "add", "path": "/hello/0/hi/5", "value": 3 },
  		{ "op": "add", "path": "/hello/0/hi/5", "value": 4 }
	]`)
	fmt.Println(ParsePatches(patches))
}

type patchesList []struct {
	Op    string
	Path  string
	Value *interface{}
}

func toDot(path string) string {
	m1 := regexp.MustCompile(`^\/`)
	path = m1.ReplaceAllString(path, "")

	m2 := regexp.MustCompile(`\/`)
	path = m2.ReplaceAllString(path, ".")

	m3 := regexp.MustCompile(`~1`)
	path = m3.ReplaceAllString(path, "/")

	m4 := regexp.MustCompile(`~0`)
	path = m4.ReplaceAllString(path, "~")

	return path
}
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ParsePatches accepts the JSON Patches as []byte and returns the equivalent MongoDB update query as bson.M
func ParsePatches(patches []byte) (bson.M, error) {
	return ParsePatchesWithPrefix(patches, "")
}

// ParsePatchesWithPrefix accepts the JSON Patches as []byte and returns the equivalent MongoDB update query as bson.M, all the paths are prepended with the prefix passed
func ParsePatchesWithPrefix(patches []byte, prefixPath string) (bson.M, error) {
	// parse patches json
	var parsedPatches patchesList
	err := json.Unmarshal(patches, &parsedPatches)
	if err != nil {
		return nil, err
	}

	update := bson.M{}

	// iterate patches and add each operation to update query
	for _, patch := range parsedPatches {
		switch patch.Op {
		// parse the add op as a push to the array in the corresponding location
		case "add":
			if _, ok := update["$push"]; !ok {
				update["$push"] = bson.M{}
			}

			// parse path dividing key of the array and location inside the array
			path := prefixPath + toDot(patch.Path)
			parts := strings.Split(path, ".")

			positionPart := ""
			if len(parts) > 1 {
				positionPart = parts[len(parts)-1]
			} else {
				return nil, fmt.Errorf("Unsupported Operation! can't use add op without position")
			}

			addToEnd := positionPart == "-"
			key := strings.Join(parts[0:len(parts)-1], ".")

			if addToEnd {
				// handle appends to the end of the array
				if _, ok := update["$push"].(bson.M)[key]; ok {
					// another add operation to this same array has been parsed
					// convert the pushed content from a single value to a list of values
					if update["$push"].(bson.M)[key] == nil {
						val := bson.A{update["$push"].(bson.M)[key]}
						update["$push"].(bson.M)[key] = bson.M{"$each": val}
					} else if fmt.Sprintf("%T", update["$push"].(bson.M)[key]) != "primitive.M" {
						val := bson.A{update["$push"].(bson.M)[key]}
						update["$push"].(bson.M)[key] = bson.M{"$each": val}
					} else if _, ok := update["$push"].(bson.M)[key].(bson.M)["$each"]; !ok {
						val := bson.A{update["$push"].(bson.M)[key]}
						update["$push"].(bson.M)[key] = bson.M{"$each": val}
					}

					// adding both to specific locations and to the end of the array is not supported
					if _, ok := update["$push"].(bson.M)[key].(bson.M)["$position"]; ok {
						return nil, fmt.Errorf("Unsupported Operation! can't use add op with mixed positions")
					}

					// add the value passed to the list of the values to be pushed
					if patch.Value == nil {
						update["$push"].(bson.M)[key].(bson.M)["$each"] = append(update["$push"].(bson.M)[key].(bson.M)["$each"].(bson.A), nil)
					} else {
						update["$push"].(bson.M)[key].(bson.M)["$each"] = append(update["$push"].(bson.M)[key].(bson.M)["$each"].(bson.A), *patch.Value)
					}
				} else {
					// no other add operations to this same array have been parsed yet
					// simply push the value passed

					if patch.Value == nil {
						update["$push"].(bson.M)[key] = nil
					} else {
						update["$push"].(bson.M)[key] = *patch.Value
					}
				}
			} else {
				i1, err := strconv.Atoi(positionPart)
				if err != nil {
					return nil, err
				}
				position := i1

				if _, ok := update["$push"].(bson.M)[key]; ok {
					// Return error if previous operations added to the end of the array
					if update["$push"].(bson.M)[key] == nil || fmt.Sprintf("%T", update["$push"].(bson.M)[key]) != "primitive.M" {
						return nil, fmt.Errorf("Unsupported Operation! can't use add op with mixed positions")
					} else if _, ok := update["$push"].(bson.M)[key].(bson.M)["$position"]; !ok {
						return nil, fmt.Errorf("Unsupported Operation! can't use add op with mixed positions")
					}

					// The items inserted must be in contigous positions
					posDiff := position - update["$push"].(bson.M)[key].(bson.M)["$position"].(int)
					if posDiff > len(update["$push"].(bson.M)[key].(bson.M)["$each"].(primitive.A)) {
						return nil, fmt.Errorf("Unsupported Operation! can use add op only with contiguous positions")
					}

					// current list of items to push and value to push
					currEach := update["$push"].(bson.M)[key].(bson.M)["$each"].(bson.A)
					var val interface{} = nil
					if patch.Value != nil {
						val = *patch.Value
					}

					// insert val in currEach in the right position
					newEach := append(currEach, nil)
					copy(newEach[posDiff+1:], newEach[posDiff:])
					newEach[posDiff] = val
					update["$push"].(bson.M)[key].(bson.M)["$each"] = newEach

					update["$push"].(bson.M)[key].(bson.M)["$position"] = min(position, update["$push"].(bson.M)[key].(bson.M)["$position"].(int))
				} else {
					val := bson.A{nil}
					if patch.Value != nil {
						val = bson.A{*patch.Value}
					}

					update["$push"].(bson.M)[key] = bson.M{"$each": val, "$position": position}
				}
			}

		// parse remove op adding the removed path to the $unset query
		case "remove":
			if _, ok := update["$unset"]; !ok {
				update["$unset"] = bson.M{}
			}

			update["$unset"].(bson.M)[toDot(patch.Path)] = 1

		// parse replace op adding the replaced path to the $set query with the correct value
		case "replace":
			if _, ok := update["$set"]; !ok {
				update["$set"] = bson.M{}
			}
			update["$set"].(bson.M)[toDot(patch.Path)] = *patch.Value

		// the test op does not change the query
		case "test":

		default:
			return nil, fmt.Errorf("Unsupported Operation! op = " + patch.Op)
		}
	}

	return update, nil
}
