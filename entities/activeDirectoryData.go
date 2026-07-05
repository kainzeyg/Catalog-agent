package entities

type ADUser struct {
	Username string
	UPN      string
	DN       string
	Groups   []string
	Domain   string
}

type ADGroup struct {
	CN   string
	DN   string
	Name string
}
