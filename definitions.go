package monday

type User struct {
	Id    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type Board struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type Group struct {
	Id    string `json:"id"`
	Title string `json:"title"`
}

type Column struct {
	Id       string `json:"id"`
	Title    string `json:"title"`
	Type     string `json:"type"`         // text, boolean, color, ...
	Settings string `json:"settings_str"` // used to get label index values for color(status) and dropdown column types
}

type ColumnMap map[string]Column // key is column id, provides easy access to a board's column info using column id

type ColumnValue struct {
	Id    string `json:"id"`    // column id
	Value string `json:"value"` // see func DecodeValue below
}

type Item struct {
	Id           string
	GroupId      string
	Name         string
	ColumnValues []ColumnValue
}

// following types used to convert value from/to specific Monday value type
type DateTime struct {
	Date string `json:"date"`
	Time string `json:"time"`
}

type StatusIndex struct {
	Index int `json:"index"`
}

type PersonTeam struct {
	Id   int    `json:"id"`
	Kind string `json:"kind"` // "person" or "team"
}

type People struct {
	PersonsAndTeams []PersonTeam `json:"personsAndTeams"`
}

type Checkbox struct {
	Checked string `json:"checked"`
}
