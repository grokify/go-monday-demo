package monday

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"

	"github.com/machinebox/graphql"
)

// use Monday.com app to create a new token and paste here
const token = "Basic xxxxxx"

// CreateClient creates a graphql client (safe to share across requests)
func CreateClient() *graphql.Client {
	return graphql.NewClient("https://api.monday.com/v2/")
}

// RunRequest executes request and decodes response into response parm (address of object)
func RunRequest(client *graphql.Client, req *graphql.Request, response interface{}) error {
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Authorization", token)
	req.Header.Set("Content-Type", "application/json")
	ctx := context.Background()
	err := client.Run(ctx, req, response)
	return err
}

// GetUsers returns []User for all users.
func GetUsers(client *graphql.Client) ([]User, error) {
	req := graphql.NewRequest(`
	    query {
            users {
                id name email
            }
        }
    `)
	var response struct {
		Users []User `json:"users"`
	}
	err := RunRequest(client, req, &response)
	return response.Users, err
}

// GetBoards returns []Board for all boards.
func GetBoards(client *graphql.Client) ([]Board, error) {
	req := graphql.NewRequest(`
	    query {
            boards {
                id name
            }
        }
    `)
	var response struct {
		Boards []Board `json:"boards"`
	}
	err := RunRequest(client, req, &response)
	return response.Boards, err
}

// GetGroups returns []Group for specified board.
func GetGroups(client *graphql.Client, boardID int) ([]Group, error) {
	req := graphql.NewRequest(`
		query ($boardId: [Int]) {
			boards (ids: $boardId) {
				groups {
					id title
				}	
            }
        }
	`)
	req.Var("boardId", []int{boardID})
	type board struct {
		Groups []Group `json:"groups"`
	}
	var response struct {
		Boards []board `json:"boards"`
	}
	err := RunRequest(client, req, &response)
	return response.Boards[0].Groups, err
}

// GetColumns returns []Column for specified `boardID“.
func GetColumns(client *graphql.Client, boardID int) ([]Column, error) {
	req := graphql.NewRequest(`
	    query ($boardId: [Int]) {
            boards (ids: $boardId) {
                columns {id title type settings_str}
            }
        }
    `)
	req.Var("boardId", []int{boardID})
	type board struct {
		Columns []Column `json:"columns"`
	}
	var response struct {
		Boards []board `json:"boards"`
	}
	err := RunRequest(client, req, &response)
	return response.Boards[0].Columns, err
}

// CreateColumnMap returns map[string]Column for specified boardId. Key is columnId.
func CreateColumnMap(client *graphql.Client, boardID int) (ColumnMap, error) {
	var columns []Column
	columnMap := make(ColumnMap)

	columns, err := GetColumns(client, boardID)
	if err != nil {
		return columnMap, err
	}
	for _, column := range columns {
		columnMap[column.ID] = column
	}
	return columnMap, nil
}

// Example of creating columnValues for AddItem
// map entry key is column id; run GetColumns to get column id's
/*
	columnValues := map[string]interface{}{
		"text":   "have a nice day",
		"date":   monday.BuildDate("2019-05-22"),
		"status": monday.BuildStatusIndex(2),
		"people": monday.BuildPeople(123456, 987654),   // parameters are user ids
	}
*/

func BuildDate(date string) DateTime {
	return DateTime{Date: date}
}
func BuildDateTime(date, time string) DateTime {
	return DateTime{Date: date, Time: time}
}
func BuildStatusIndex(index int) StatusIndex {
	return StatusIndex{index}
}
func BuildCheckbox(checked string) Checkbox {
	return Checkbox{checked}
}
func BuildPeople(userIds ...int) People {
	response := People{}
	response.PersonsAndTeams = make([]PersonTeam, len(userIds))
	for i, id := range userIds {
		response.PersonsAndTeams[i] = PersonTeam{id, "person"}
	}
	return response
}

// AddItem adds 1 item to specified board/group. The id of the added item is returned.
func AddItem(client *graphql.Client, boardID int, groupID string, itemName string, columnValues map[string]interface{}) (string, error) {
	req := graphql.NewRequest(`
        mutation ($boardId: Int!, $groupId: String!, $itemName: String!, $colValues: JSON!) {
            create_item (board_id: $boardId, group_id: $groupId, item_name: $itemName, column_values: $colValues ) {
                id
            }
        }
    `)
	jsonValues, _ := json.Marshal(&columnValues)
	log.Println(string(jsonValues))

	req.Var("boardId", boardID)
	req.Var("groupId", groupID)
	req.Var("itemName", itemName)
	req.Var("colValues", string(jsonValues))

	type ItemId struct {
		Id string `json:"id"` // Note value is numeric and not enclosed in quotes, but does not work with type int
	}
	var response struct {
		CreateItem ItemId `json:"create_item"`
	}
	err := RunRequest(client, req, &response)
	return response.CreateItem.Id, err
}

// AddItemUpdate adds an update entry to specified item.
func AddItemUpdate(client *graphql.Client, itemID string, msg string) error {
	intItemID, err := strconv.Atoi(itemID)
	if err != nil {
		log.Println("AddItemUpdate - bad itemId", err)
		return err
	}
	req := graphql.NewRequest(`
		mutation ($itemId: Int!, $body: String!) {
			create_update (item_id: $itemId, body: $body ) {
				id
			}
		}
	`)
	req.Var("itemId", intItemID)
	req.Var("body", msg)

	type UpdateId struct {
		Id string `json:"id"`
	}
	var response struct {
		CreateUpdate UpdateId `json:"create_update"`
	}
	err = RunRequest(client, req, &response)
	return err
}

// GetItems returns []Item for all items in specified board.
func GetItems(client *graphql.Client, boardID int) ([]Item, error) {
	req := graphql.NewRequest(`	
		query ($boardId: [Int]) {
			boards (ids: $boardId){
				# items (limit: 10) {
				items () {
					id
					group {	id }
					name
					# column_values (ids: ["text", "status", "check"]) {  -- to get specific columns  
					column_values { 
						id value
					}
				}	
			}
		}	
	`)
	req.Var("boardId", []int{boardID})

	type group struct {
		ID string `json:"id"`
	}
	type itemData struct {
		ID           string        `json:"id"`
		Group        group         `json:"group"`
		Name         string        `json:"name"`
		ColumnValues []ColumnValue `json:"column_values"`
	}
	type boardItems struct {
		Items []itemData `json:"items"`
	}
	var response struct {
		Boards []boardItems `json:"boards"`
	}
	items := make([]Item, 0, 1000)
	err := RunRequest(client, req, &response)
	if err != nil {
		fmt.Println("GetItems Failed -", err)
		return items, err
	}
	var responseItems []itemData = response.Boards[0].Items
	for _, responseItem := range responseItems {
		items = append(items, Item{
			ID:           responseItem.ID,
			GroupID:      responseItem.Group.ID,
			Name:         responseItem.Name,
			ColumnValues: responseItem.ColumnValues,
		})
	}
	return items, nil
}

// DecodeValues converts column value returned from Monday to a string value
//
//	color(status) returns index of label chosen, ex. "3"
//	boolean(checkbox) returns "true" or "false"
//	date returns "2019-05-22"
//
// Types "multi-person" and "dropdown" may have multiple values.
//
//	for these, a slice of strings is returned
//
// Use CreateColumnMap to create the columnMap (contains info for all columns in board)
func DecodeValue(columnMap ColumnMap, columnValue ColumnValue) (result1 string, result2 []string, err error) {
	if columnValue.Value == "" {
		return
	}
	column, found := columnMap[columnValue.ID]
	if !found {
		err = errors.New("invalid column id - " + columnValue.ID)
		return
	}
	inVal := []byte(columnValue.Value) // convert input value (string) to []byte, required by json.Unmarshal
	switch column.Type {
	case "text":
		result1 = columnValue.Value
	case "color": // status, return index of value
		var val StatusIndex
		err = json.Unmarshal(inVal, &val)
		result1 = strconv.Itoa(val.Index)
	case "boolean": // checkbox, return true or false
		var val Checkbox
		err = json.Unmarshal(inVal, &val)
		result1 = val.Checked
	case "date":
		var val DateTime
		err = json.Unmarshal(inVal, &val)
		result1 = val.Date
	case "multiple-person":
		result2 = DecodePeople(columnValue.Value)
	case "dropdown":
		result2 = DecodeDropDown(columnValue.Value)
	default:
		err = errors.New("value type not handled - " + column.Type)
	}
	return
}

// DecodePeople returns user id of each person assigned. Use GetUsers to get all user id values.
func DecodePeople(valueIn string) []string {
	var val People
	err := json.Unmarshal([]byte(valueIn), &val)
	if err != nil {
		log.Println("DecodePeople Unmarshal Failed, ", err)
		return nil
	}
	result := make([]string, len(val.PersonsAndTeams))
	for i, person := range val.PersonsAndTeams {
		result[i] = strconv.Itoa(person.ID)
	}
	return result
}

// DecodeDropDown returns ids of value selections. Use DecodeLabels to list Index value for each dropdown label.
func DecodeDropDown(valueIn string) []string {
	var val struct {
		Ids []int `json:"ids"`
	}
	err := json.Unmarshal([]byte(valueIn), &val)
	if err != nil {
		log.Println("DecodeDropDown Unmarshal Failed, ", err)
		return nil
	}
	result := make([]string, len(val.Ids))
	for i, id := range val.Ids {
		result[i] = strconv.Itoa(id)
	}
	return result
}

// DecodeLabels displays index value of all labels for a column. Uses column settings_str (see GetColumns).
// Use for Status(color) and Dropdown fields.
func DecodeLabels(settingsStr, columnType string) {
	var statusLabels struct {
		Labels         map[string]string `json:"labels"`             // index: label
		LabelPositions map[string]int    `json:"label_positions_v2"` // index: position
	}
	type dropdownEntry struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	var dropdownLabels struct {
		Labels []dropdownEntry `json:"labels"`
	}

	if columnType == "color" {
		err := json.Unmarshal([]byte(settingsStr), &statusLabels)
		if err != nil {
			log.Fatal("DecodeLabels Failed", err)
		}
		for index, label := range statusLabels.Labels {
			fmt.Println(index, label)
		}
	}
	if columnType == "dropdown" {
		err := json.Unmarshal([]byte(settingsStr), &dropdownLabels)
		if err != nil {
			log.Fatal("DecodeLabels Failed", err)
		}
		for _, label := range dropdownLabels.Labels {
			fmt.Println(label.ID, label.Name)
		}
	}
}
